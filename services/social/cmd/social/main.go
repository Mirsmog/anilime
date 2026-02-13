package main

import (
	"context"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/social/internal/grpcapi"
	"github.com/example/anime-platform/services/social/internal/handlers"
	"github.com/example/anime-platform/services/social/internal/store"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/services/social/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log, err := logging.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	ratings, closePool := initRatings(log)
	if closePool != nil {
		defer closePool()
	}

	comments, closeCommentPool := initComments(log)
	if closeCommentPool != nil {
		defer closeCommentPool()
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	verifier := auth.JWTVerifier{Secret: []byte(jwtSecret)}

	r := chi.NewRouter()
	httpserver.SetupRouter(r)
	r.Get("/v1/ratings/{anime_id}", handlers.GetRatings(ratings))
	r.Post("/v1/ratings/{anime_id}", handlers.PostRating(ratings))

	// Comment routes (public read, auth required for write)
	r.Get("/v1/comments/{anime_id}", handlers.GetThread(comments))
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireUser(verifier))
		r.Post("/v1/comments/{anime_id}", handlers.CreateComment(comments))
		r.Post("/v1/comments/{comment_id}/vote", handlers.VoteComment(comments))
		r.Put("/v1/comments/{comment_id}", handlers.UpdateComment(comments))
		r.Delete("/v1/comments/{comment_id}", handlers.DeleteComment(comments))
	})

	srv := httpserver.New(httpserver.Options{Addr: cfg.HTTP.Addr, ServiceName: cfg.ServiceName, Logger: log, Router: r})

	// gRPC server
	grpcAddr := strings.TrimSpace(os.Getenv("GRPC_ADDR"))
	if grpcAddr == "" {
		grpcAddr = ":9090"
	}
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc listen", zap.Error(err))
		run.Exit(1)
	}
	grpcSrv := grpc.NewServer()
	socialv1.RegisterSocialServiceServer(grpcSrv, &grpcapi.SocialService{Comments: comments})
	reflection.Register(grpcSrv)
	go func() {
		log.Info("grpc server starting", zap.String("addr", grpcAddr))
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("grpc serve", zap.Error(err))
		}
	}()

	runner := run.New(log)
	code := runner.WithSignals(func(ctx context.Context) error {
		// start comments consumer (non-fatal if NATS unavailable)
		nc, err := natsconn.Connect(natsconn.Options{})
		if err != nil {
			log.Error("nats connect", zap.Error(err))
		} else {
			go worker.StartCommentsConsumer(ctx, nc)
			defer nc.Close()
		}

		go func() {
			<-ctx.Done()
			stopped := make(chan struct{})
			go func() {
				grpcSrv.GracefulStop()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-time.After(10 * time.Second):
				grpcSrv.Stop()
			}
			_ = srv.Shutdown(context.Background())
		}()
		return srv.Start(log)
	})

	log.Info("exit", zap.Int("code", code))
	run.Exit(code)
}

// initRatings selects the RatingStore backend.
// In production (APP_ENV=production) it requires a working Postgres connection
// and terminates the process otherwise.
func initRatings(log *zap.Logger) (store.RatingStore, func()) {
	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		if isProd {
			log.Error("DATABASE_URL is required in production")
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("DATABASE_URL not set, using in-memory rating store (development only)")
		return store.NewInMemoryRatingStore(), nil
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		if isProd {
			log.Error("postgres is required in production but unavailable", zap.Error(err))
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("postgres unavailable, falling back to in-memory store", zap.Error(err))
		return store.NewInMemoryRatingStore(), nil
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		if isProd {
			log.Error("postgres ping failed in production", zap.Error(err))
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("postgres ping failed, falling back to in-memory store", zap.Error(err))
		return store.NewInMemoryRatingStore(), nil
	}

	log.Info("ratings store: postgres")
	return store.NewPostgresRatingStore(pool), pool.Close
}

// initComments selects the CommentStore backend.
func initComments(log *zap.Logger) (store.CommentStore, func()) {
	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		if isProd {
			log.Error("DATABASE_URL is required in production")
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("DATABASE_URL not set, using in-memory comment store (development only)")
		return store.NewInMemoryCommentStore(), nil
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		if isProd {
			log.Error("postgres is required in production but unavailable", zap.Error(err))
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("postgres unavailable, falling back to in-memory comment store", zap.Error(err))
		return store.NewInMemoryCommentStore(), nil
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		if isProd {
			log.Error("postgres ping failed in production", zap.Error(err))
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("postgres ping failed, falling back to in-memory comment store", zap.Error(err))
		return store.NewInMemoryCommentStore(), nil
	}

	log.Info("comments store: postgres")
	return store.NewPostgresCommentStore(pool), pool.Close
}

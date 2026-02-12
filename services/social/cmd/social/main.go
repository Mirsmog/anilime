package main

import (
	"context"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/social/internal/handlers"
	"github.com/example/anime-platform/services/social/internal/store"
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

	r := chi.NewRouter()
	httpserver.SetupRouter(r)
	r.Get("/v1/ratings/{anime_id}", handlers.GetRatings(ratings))
	r.Post("/v1/ratings/{anime_id}", handlers.PostRating(ratings))

	srv := httpserver.New(httpserver.Options{Addr: cfg.HTTP.Addr, ServiceName: cfg.ServiceName, Logger: log, Router: r})

	runner := run.New(log)
	code := runner.WithSignals(func(ctx context.Context) error {
		go func() {
			<-ctx.Done()
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

	log.Info("ratings store: postgres")
	return store.NewPostgresRatingStore(pool), pool.Close
}

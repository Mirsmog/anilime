package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/analytics"
	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/bff/internal/admin"
	bffconfig "github.com/example/anime-platform/services/bff/internal/config"
	"github.com/example/anime-platform/services/bff/internal/grpcclient"

	bffhandlers "github.com/example/anime-platform/services/bff/internal/handlers"
	bffhttp "github.com/example/anime-platform/services/bff/internal/http"
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

	bffCfg, err := bffconfig.LoadBFF()
	if err != nil {
		log.Error("load bff config", zap.Error(err))
		run.Exit(1)
	}

	r := chi.NewRouter()
	httpserver.SetupRouter(r)

	verifier := auth.JWTVerifier{Secret: bffCfg.JWTSecret}

	nc, err := natsconn.Connect(natsconn.Options{URL: bffCfg.NATSURL})
	if err != nil {
		log.Error("nats connect", zap.Error(err))
		run.Exit(1)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		log.Error("jetstream", zap.Error(err))
		run.Exit(1)
	}

	// init bff cache with NATS invalidation
	bffCache := bffhandlers.NewTTLCache(bffCfg.CacheTTLSeconds, nc, bffCfg.CacheInvalidationSubj)
	eventPublisher := bffhandlers.NewEventPublisher(js)
	analyticsPublisher := analytics.New(js, log)
	authc, err := grpcclient.NewAuthClient(bffCfg.AuthGRPCAddr)
	if err != nil {
		log.Error("init auth grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer authc.Conn.Close()

	catalogc, err := grpcclient.NewCatalogClient(bffCfg.CatalogGRPCAddr)
	if err != nil {
		log.Error("init catalog grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer catalogc.Conn.Close()

	activityc, err := grpcclient.NewActivityClient(bffCfg.ActivityGRPCAddr)
	if err != nil {
		log.Error("init activity grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer activityc.Conn.Close()

	searchc, err := grpcclient.NewSearchClient(bffCfg.SearchGRPCAddr)
	if err != nil {
		log.Error("init search grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer searchc.Conn.Close()

	streamingc, err := grpcclient.NewStreamingResolverClient(bffCfg.StreamingGRPCAddr)
	if err != nil {
		log.Error("init streaming grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer streamingc.Conn.Close()

	socialc, err := grpcclient.NewSocialClient(bffCfg.SocialGRPCAddr)
	if err != nil {
		log.Error("init social grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer socialc.Conn.Close()

	// Example route: in real BFF you aggregate from other services.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("anime-platform bff"))
	})

	// Auth endpoints with rate limiting (5 req/s, burst 10)
	authLimiter := bffhttp.NewRateLimiter(5, 10)
	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Middleware)
		r.Post("/v1/auth/register", bffhandlers.Register(authc.Client, analyticsPublisher))
		r.Post("/v1/auth/login", bffhandlers.Login(authc.Client, analyticsPublisher))
		r.Post("/v1/auth/refresh", bffhandlers.Refresh(authc.Client))
		r.Post("/v1/auth/logout", bffhandlers.Logout(authc.Client))
	})

	r.Get("/v1/search", bffhandlers.Search(searchc.Client, bffCache))

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireUser(verifier))
		r.Get("/v1/watch/{episode_id}", bffhandlers.Watch(streamingc.Client, bffCfg.HLSProxyBaseURL, bffCfg.HLSProxySigningSecret, analyticsPublisher))
	})

	r.Route("/v1/admin", func(r chi.Router) {
		r.Use(auth.RequireUser(verifier))
		r.Use(auth.RequireAdmin)
		admin.BackfillHandler{JikanBaseURL: bffCfg.JikanBaseURL, JS: js}.Register(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireUser(verifier))

		r.Get("/v1/me", bffhandlers.Me(authc.Client))

		r.Post("/v1/activity/progress", bffhandlers.UpsertProgress(activityc.Client, eventPublisher))
		r.Get("/v1/activity/continue", bffhandlers.ContinueWatching(activityc.Client, catalogc.Client))

		r.Post("/v1/comments/{anime_id}", bffhandlers.CreateComment(socialc.Client, eventPublisher))
		r.Post("/v1/comments/{comment_id}/vote", bffhandlers.VoteComment(socialc.Client, eventPublisher))
		r.Put("/v1/comments/{comment_id}", bffhandlers.UpdateComment(socialc.Client, eventPublisher))
		r.Delete("/v1/comments/{comment_id}", bffhandlers.DeleteComment(socialc.Client, eventPublisher))
	})

	// Public comment listing (no auth required)
	r.Get("/v1/comments/{anime_id}", bffhandlers.ListComments(socialc.Client))

	// Public catalog endpoints (no auth required)
	r.Get("/v1/anime", bffhandlers.ListAnime(catalogc.Client, bffCache))
	r.Get("/v1/anime/{anime_id}", bffhandlers.GetAnime(catalogc.Client, analyticsPublisher))
	r.Get("/v1/anime/{anime_id}/episodes", bffhandlers.GetEpisodesByAnime(catalogc.Client))
	r.Get("/v1/episodes/{episode_id}", bffhandlers.GetEpisode(catalogc.Client))

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

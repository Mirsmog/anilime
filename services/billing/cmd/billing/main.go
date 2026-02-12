package main

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	billingconfig "github.com/example/anime-platform/services/billing/internal/config"
	"github.com/example/anime-platform/services/billing/internal/handlers"
	"github.com/example/anime-platform/services/billing/internal/idempotency"
	"github.com/example/anime-platform/services/billing/internal/publisher"
	billingstore "github.com/example/anime-platform/services/billing/internal/store"
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

	billingCfg, err := billingconfig.Load()
	if err != nil {
		log.Error("billing config", zap.Error(err))
		panic(err)
	}

	// Optional Postgres pool for billing persistence.
	var pool *pgxpool.Pool
	if billingCfg.DatabaseURL != "" {
		p, err := pgxpool.New(context.Background(), billingCfg.DatabaseURL)
		if err != nil {
			log.Warn("postgres unavailable, billing will run without persistence", zap.Error(err))
		} else {
			pool = p
			defer pool.Close()
			log.Info("postgres connected for billing")
		}
	}

	idem := idempotency.NewStore(billingCfg.RedisDSN, billingCfg.DatabaseURL, billingCfg.IdempotencyTTL)
	log.Info("idempotency store initialised",
		zap.Bool("redis", billingCfg.RedisDSN != ""),
		zap.Bool("postgres", billingCfg.DatabaseURL != ""),
	)

	st := billingstore.New(pool)

	pub, err := publisher.New(billingCfg.NATSURL, log)
	if err != nil {
		log.Warn("NATS unavailable, billing events will not be published", zap.Error(err))
		pub, _ = publisher.New("", log) // stub
	}

	webhookHandler := handlers.NewWebhookHandler(billingCfg.StripeWebhookSecret, log, idem, st, pub)

	r := chi.NewRouter()
	httpserver.SetupRouter(r)
	r.Post("/v1/stripe/webhook", webhookHandler.ServeHTTP)

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

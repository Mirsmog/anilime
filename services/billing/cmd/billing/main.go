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

	pool, closePool := initPool(log, billingCfg)
	if closePool != nil {
		defer closePool()
	}

	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
	idem, err := idempotency.NewStore(billingCfg.RedisDSN, billingCfg.DatabaseURL, billingCfg.IdempotencyTTL, isProd)
	if err != nil {
		log.Error("idempotency store", zap.Error(err))
		run.Exit(1)
	}
	log.Info("idempotency store initialised",
		zap.Bool("redis", billingCfg.RedisDSN != ""),
		zap.Bool("postgres", billingCfg.DatabaseURL != ""),
	)

	st := billingstore.New(pool)

	pub, err := publisher.New(billingCfg.NATSURL, log)
	if err != nil {
		if isProd {
			log.Error("NATS is required in production", zap.Error(err))
			run.Exit(1)
		}
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

// initPool initialises the Postgres connection pool for billing.
// In production (APP_ENV=production) it requires a working connection and
// terminates the process otherwise.
func initPool(log *zap.Logger, billingCfg billingconfig.Config) (*pgxpool.Pool, func()) {
	isProd := strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")

	if billingCfg.DatabaseURL == "" {
		if isProd {
			log.Error("DATABASE_URL is required in production")
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("DATABASE_URL not set, billing will run without persistence (development only)")
		return nil, nil
	}

	pool, err := pgxpool.New(context.Background(), billingCfg.DatabaseURL)
	if err != nil {
		if isProd {
			log.Error("DATABASE_URL is set but Postgres is unreachable in production", zap.Error(err))
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("postgres unavailable, billing will run without persistence", zap.Error(err))
		return nil, nil
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		if isProd {
			log.Error("Postgres ping failed in production", zap.Error(err))
			_ = log.Sync()
			os.Exit(1)
		}
		log.Warn("postgres ping failed, billing will run without persistence", zap.Error(err))
		return nil, nil
	}

	log.Info("postgres connected for billing")
	return pool, pool.Close
}

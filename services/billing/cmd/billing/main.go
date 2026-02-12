package main

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	billingconfig "github.com/example/anime-platform/services/billing/internal/config"
	"github.com/example/anime-platform/services/billing/internal/handlers"
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

	billingCfg := billingconfig.Load()

	webhookHandler := handlers.NewWebhookHandler(billingCfg.StripeWebhookSecret, log)

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

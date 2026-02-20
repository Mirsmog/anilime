package main

import (
	"context"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/analytics/internal/config"
	"github.com/example/anime-platform/services/analytics/internal/consumer"
	"github.com/example/anime-platform/services/analytics/internal/handler"
	"github.com/example/anime-platform/services/analytics/internal/posthog"
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

	ph, err := posthog.New(cfg.PostHogAPIKey, cfg.PostHogHost, cfg.FlushInterval, cfg.PostHogBatchSize, log)
	if err != nil {
		log.Error("posthog init", zap.Error(err))
		run.Exit(1)
	}
	defer func() {
		if err := ph.Close(); err != nil {
			log.Warn("posthog close", zap.Error(err))
		}
	}()

	nc, err := natsconn.Connect(natsconn.Options{URL: cfg.NATSURL})
	if err != nil {
		log.Error("nats connect", zap.Error(err))
		run.Exit(1)
	}
	defer nc.Close()

	dispatcher := handler.New(ph, log)

	c, err := consumer.New(nc, dispatcher, cfg.NATSBatchSize, cfg.BatchIntervalMs, log)
	if err != nil {
		log.Error("consumer init", zap.Error(err))
		run.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("analytics consumer started")
	c.Run(ctx)
	log.Info("analytics consumer stopped")
	run.Exit(0)
}

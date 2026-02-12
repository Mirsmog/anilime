package main

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	streamingv1 "github.com/example/anime-platform/gen/streaming/v1"
	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	svcconfig "github.com/example/anime-platform/services/streaming/internal/config"
	"github.com/example/anime-platform/services/streaming/internal/handlers"
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

	svcCfg := svcconfig.Load()

	conn, err := grpc.NewClient(svcCfg.StreamingResolverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("streaming-resolver grpc dial", zap.Error(err))
		run.Exit(1)
	}
	defer conn.Close()
	resolverClient := streamingv1.NewStreamingResolverServiceClient(conn)

	r := chi.NewRouter()
	httpserver.SetupRouter(r)
	r.Get("/v1/sources/{anime_id}", handlers.Sources(resolverClient, log))

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

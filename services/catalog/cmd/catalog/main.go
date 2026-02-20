package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/internal/platform/db"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/internal/platform/run"
	catalogconfig "github.com/example/anime-platform/services/catalog/internal/config"
	grpcapi "github.com/example/anime-platform/services/catalog/internal/grpc"
	"github.com/example/anime-platform/services/catalog/internal/outbox"
	catalogstore "github.com/example/anime-platform/services/catalog/internal/store"
)

func main() {
	// logger
	cfgService := os.Getenv("SERVICE_NAME")
	logLevel := os.Getenv("LOG_LEVEL")
	if cfgService == "" {
		cfgService = "catalog"
	}
	log, err := logging.New(logLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	_ = cfgService

	// db
	pool, err := db.Open(context.Background())
	if err != nil {
		log.Error("db open", zap.Error(err))
		run.Exit(1)
	}
	defer pool.Close()

	grpcCfg := catalogconfig.LoadGRPC()
	outboxCfg := catalogconfig.LoadOutbox()

	lis, err := net.Listen("tcp", grpcCfg.Addr)
	if err != nil {
		log.Error("listen", zap.Error(err))
		run.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(grpcSrv, &grpcapi.CatalogService{
		Store: catalogstore.NewPostgresCatalogStore(pool),
	})
	reflection.Register(grpcSrv)

	nc, err := natsconn.Connect(natsconn.Options{URL: outboxCfg.NATSURL})
	if err != nil {
		log.Error("nats connect", zap.Error(err))
		run.Exit(1)
	}
	defer nc.Close()

	publisher, err := outbox.NewPublisher(log, pool, nc)
	if err != nil {
		log.Error("outbox publisher", zap.Error(err))
		run.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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
	}()

	go func() {
		if err := publisher.Run(ctx); err != nil {
			log.Error("outbox publisher stopped", zap.Error(err))
			stop()
		}
	}()

	log.Info("grpc server starting", zap.String("addr", grpcCfg.Addr))

	if err := grpcSrv.Serve(lis); err != nil {
		log.Error("grpc serve", zap.Error(err))
		run.Exit(1)
	}

	run.Exit(0)
}

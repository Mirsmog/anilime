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

	"google.golang.org/grpc/credentials/insecure"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	searchv1 "github.com/example/anime-platform/gen/search/v1"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/search/internal/config"
	"github.com/example/anime-platform/services/search/internal/grpcapi"
	"github.com/example/anime-platform/services/search/internal/indexer"
	"github.com/example/anime-platform/services/search/internal/meili"
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

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("grpc listen", zap.Error(err))
		run.Exit(1)
	}

	nc, err := natsconn.Connect(natsconn.Options{URL: cfg.NATSURL})
	if err != nil {
		log.Error("nats connect", zap.Error(err))
		run.Exit(1)
	}
	defer nc.Close()

	catalogConn, err := grpc.NewClient(cfg.CatalogGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("catalog grpc", zap.Error(err))
		run.Exit(1)
	}
	defer catalogConn.Close()

	catalogClient := catalogv1.NewCatalogServiceClient(catalogConn)
	meiliClient := meili.New(cfg.MeiliURL, cfg.MeiliAPIKey)

	idx := &indexer.Config{CatalogClient: catalogClient, Meili: meiliClient, Log: log, NATS: nc, ReindexEvery: cfg.ReindexInterval}
	if cfg.ReindexOnce {
		if err := idx.ReindexAll(context.Background()); err != nil {
			log.Error("reindex failed", zap.Error(err))
			run.Exit(1)
		}
	}
	go func() {
		if err := idx.Run(context.Background()); err != nil {
			log.Error("indexer stopped", zap.Error(err))
			os.Exit(1)
		}
	}()

	grpcSrv := grpc.NewServer()
	searchv1.RegisterSearchServiceServer(grpcSrv, &grpcapi.SearchService{Meili: meiliClient})
	reflection.Register(grpcSrv)

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

	log.Info("grpc server starting", zap.String("addr", cfg.GRPCAddr))
	if err := grpcSrv.Serve(lis); err != nil {
		log.Error("grpc serve", zap.Error(err))
		run.Exit(1)
	}
}

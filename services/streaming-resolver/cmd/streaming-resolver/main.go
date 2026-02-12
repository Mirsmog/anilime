package main

import (
	"context"
	"net"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/sony/gobreaker"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	streamingv1 "github.com/example/anime-platform/gen/streaming/v1"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/streaming-resolver/internal/cache"
	"github.com/example/anime-platform/services/streaming-resolver/internal/config"
	"github.com/example/anime-platform/services/streaming-resolver/internal/grpcapi"
	"github.com/example/anime-platform/services/streaming-resolver/internal/hianime"
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

	catalogConn, err := grpc.NewClient(cfg.CatalogGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("catalog grpc", zap.Error(err))
		run.Exit(1)
	}
	defer catalogConn.Close()

	catalogClient := catalogv1.NewCatalogServiceClient(catalogConn)
	cacheClient, err := cache.NewRedisCache(cfg.RedisURL, cfg.CacheTTL)
	if err != nil {
		log.Error("redis", zap.Error(err))
		run.Exit(1)
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "hianime",
		MaxRequests: cfg.CBMaxRequests,
		Interval:    cfg.CBInterval,
		Timeout:     cfg.CBTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.CBFailureThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Info("circuit-breaker state change", zap.String("name", name), zap.String("from", from.String()), zap.String("to", to.String()))
		},
	})

	hiAnimeClient := hianime.New(cfg.HiAnimeBaseURL, hianime.ClientConfig{
		UserAgent:      cfg.HiAnimeUserAgent,
		MaxRetries:     cfg.MaxRetries,
		RetryBaseDelay: cfg.RetryBaseDelay,
	}, hianime.WithCircuitBreaker(cb), hianime.WithLogger(log))

	resolver := &grpcapi.ResolverService{Catalog: catalogClient, HiAnime: hiAnimeClient, Cache: cacheClient, Log: log}
	grpcSrv := grpc.NewServer()
	streamingv1.RegisterStreamingResolverServiceServer(grpcSrv, resolver)
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

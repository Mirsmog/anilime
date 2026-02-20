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

	activityv1 "github.com/example/anime-platform/gen/activity/v1"
	"github.com/example/anime-platform/internal/platform/db"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/internal/platform/run"
	grpcconfig "github.com/example/anime-platform/services/activity/internal/config"
	grpcapi "github.com/example/anime-platform/services/activity/internal/grpc"
	activitystore "github.com/example/anime-platform/services/activity/internal/store"
	"github.com/example/anime-platform/services/activity/internal/worker"
)

func main() {
	cfgService := os.Getenv("SERVICE_NAME")
	logLevel := os.Getenv("LOG_LEVEL")
	if cfgService == "" {
		cfgService = "activity"
	}
	log, err := logging.New(logLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	_ = cfgService

	pool, err := db.Open(context.Background())
	if err != nil {
		log.Error("db open", zap.Error(err))
		run.Exit(1)
	}
	defer pool.Close()

	grpcCfg := grpcconfig.LoadGRPC()
	lis, err := net.Listen("tcp", grpcCfg.Addr)
	if err != nil {
		log.Error("listen", zap.Error(err))
		run.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	activityv1.RegisterActivityServiceServer(grpcSrv, &grpcapi.ActivityService{
		Progress: activitystore.NewPostgresProgressRepository(pool),
	})
	reflection.Register(grpcSrv)

	log.Info("grpc server starting", zap.String("addr", grpcCfg.Addr))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	nc, err := natsconn.Connect(natsconn.Options{})
	if err != nil {
		log.Error("nats connect", zap.Error(err))
	} else {
		go worker.StartProgressConsumer(ctx, nc, pool, log)
		defer nc.Close()
	}

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

	if err := grpcSrv.Serve(lis); err != nil {
		log.Error("grpc serve", zap.Error(err))
		run.Exit(1)
	}

	run.Exit(0)
}

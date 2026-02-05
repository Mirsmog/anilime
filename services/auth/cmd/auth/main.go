package main

import (
	"context"
	"net"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/example/anime-platform/services/auth/internal/bootstrap"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	authv1 "github.com/example/anime-platform/gen/auth/v1"

	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/auth/internal/app"
	authconfig "github.com/example/anime-platform/services/auth/internal/config"
	grpcconfig "github.com/example/anime-platform/services/auth/internal/config"
	grpcapi "github.com/example/anime-platform/services/auth/internal/grpc"
	"github.com/example/anime-platform/services/auth/internal/store"
	"github.com/example/anime-platform/services/auth/internal/tokens"
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

	// Init dependencies
	a, err := app.New(context.Background(), log)
	if err != nil {
		log.Error("init app", zap.Error(err))
		run.Exit(1)
	}
	defer a.Close()

	authCfg, err := authconfig.LoadAuth()
	if err != nil {
		log.Error("load auth config", zap.Error(err))
		run.Exit(1)
	}

	// Bootstrap admin (optional)
	if u := os.Getenv("BOOTSTRAP_ADMIN_USERNAME"); u != "" {
		if err := bootstrap.PromoteAdmin(context.Background(), a.DB, u); err != nil {
			log.Error("bootstrap admin", zap.Error(err))
			run.Exit(1)
		}
		log.Info("bootstrap admin ensured", zap.String("username", u))
	}

	grpcCfg := grpcconfig.LoadGRPC()

	lis, err := net.Listen("tcp", grpcCfg.Addr)
	if err != nil {
		log.Error("listen", zap.Error(err))
		run.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcSrv, &grpcapi.AuthService{
		Store:  store.Store{DB: a.DB},
		Tokens: tokens.Service{Secret: authCfg.JWTSecret, AccessTokenTTL: authCfg.AccessTokenTTL, RefreshTokenTTL: authCfg.RefreshTokenTTL},
		Cfg:    authCfg,
	})
	reflection.Register(grpcSrv)

	log.Info("grpc server starting", zap.String("addr", grpcCfg.Addr))

	// graceful shutdown
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

	if err := grpcSrv.Serve(lis); err != nil {
		log.Error("grpc serve", zap.Error(err))
		run.Exit(1)
	}

	run.Exit(0)
}

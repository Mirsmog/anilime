package run

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type Runner struct {
	Logger *zap.Logger
}

func New(log *zap.Logger) *Runner {
	return &Runner{Logger: log}
}

func (r *Runner) WithSignals(start func(ctx context.Context) error) int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		err := start(ctx)
		select {
		case errCh <- err:
		default:
		}
	}()

	select {
	case <-ctx.Done():
		r.Logger.Info("shutdown signal received")
		return 0
	case err := <-errCh:
		if err == nil {
			return 0
		}
		if errors.Is(err, http.ErrServerClosed) {
			return 0
		}
		r.Logger.Error("service exited with error", zap.Error(err))
		return 1
	}
}

func (r *Runner) Graceful(ctx context.Context, shutdown func(context.Context) error) {
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := shutdown(c); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, http.ErrServerClosed) {
		r.Logger.Warn("graceful shutdown failed", zap.Error(err))
	}
}

func Exit(code int) {
	os.Exit(code)
}

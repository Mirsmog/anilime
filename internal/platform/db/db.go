package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open opens a pgxpool using DATABASE_URL.
func Open(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Safe defaults; can be overridden later.
	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Retry ping for a short window to tolerate DB startup.
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			pool.Close()
			return nil, err
		}
		t := 300 * time.Millisecond
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(t):
		}
	}
	return pool, nil
}

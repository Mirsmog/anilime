package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/db"
)

type App struct {
	DB  *pgxpool.Pool
	Log *zap.Logger
}

func New(ctx context.Context, log *zap.Logger) (*App, error) {
	pool, err := db.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &App{DB: pool, Log: log}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}

package idempotency

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStore struct {
	dsn string
	ttl time.Duration
	// pool is lazily initialised on first Check call.
	pool *pgxpool.Pool
}

func newPostgresStore(dsn string, ttl time.Duration) *postgresStore {
	return &postgresStore{dsn: dsn, ttl: ttl}
}

func (s *postgresStore) ensurePool(ctx context.Context) error {
	if s.pool != nil {
		return nil
	}
	pool, err := pgxpool.New(ctx, s.dsn)
	if err != nil {
		return err
	}
	s.pool = pool
	return nil
}

// Check uses INSERT ... ON CONFLICT to atomically deduplicate.
// Table `processed_events` must exist (see billing migrations).
func (s *postgresStore) Check(ctx context.Context, eventID string) (bool, error) {
	if err := s.ensurePool(ctx); err != nil {
		return false, err
	}

	const q = `INSERT INTO processed_events (event_id, created_at)
	           VALUES ($1, now())
	           ON CONFLICT (event_id) DO NOTHING`

	tag, err := s.pool.Exec(ctx, q, eventID)
	if err != nil {
		return false, err
	}
	// RowsAffected == 0 means the row already existed (duplicate).
	return tag.RowsAffected() == 0, nil
}

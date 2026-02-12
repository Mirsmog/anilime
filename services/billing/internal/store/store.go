// Package store provides Postgres persistence for billing data.
package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BillingStore persists payments and subscriptions in Postgres.
type BillingStore struct {
	pool *pgxpool.Pool
}

// New creates a BillingStore. pool may be nil for stub mode.
func New(pool *pgxpool.Pool) *BillingStore {
	return &BillingStore{pool: pool}
}

// SavePayment inserts a payment record inside the given transaction.
func (s *BillingStore) SavePayment(ctx context.Context, tx pgx.Tx, eventID string, rawData json.RawMessage) error {
	const q = `INSERT INTO payments (event_id, raw_data)
	           VALUES ($1, $2)
	           ON CONFLICT (event_id) DO NOTHING`
	_, err := tx.Exec(ctx, q, eventID, rawData)
	return err
}

// SaveSubscription inserts or updates a subscription record inside the given transaction.
func (s *BillingStore) SaveSubscription(ctx context.Context, tx pgx.Tx, eventID string, rawData json.RawMessage) error {
	const q = `INSERT INTO subscriptions (event_id, raw_data)
	           VALUES ($1, $2)
	           ON CONFLICT (event_id) DO UPDATE SET
	             raw_data = EXCLUDED.raw_data,
	             updated_at = now()`
	_, err := tx.Exec(ctx, q, eventID, rawData)
	return err
}

// BeginTx starts a new transaction.
func (s *BillingStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

// Available returns true when the Postgres pool is configured.
func (s *BillingStore) Available() bool {
	return s.pool != nil
}

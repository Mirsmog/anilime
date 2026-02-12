// Package idempotency provides deduplication for Stripe webhook event IDs.
//
// Primary backend: Redis SETNX with TTL (env REDIS_DSN).
// Fallback: Postgres INSERT ... ON CONFLICT (env DATABASE_URL).
// If neither is available, an in-memory store is used (development only).
package idempotency

import (
	"context"
	"errors"
	"time"
)

// Store checks whether an event has already been processed and marks it.
type Store interface {
	// Check returns true if eventID was already processed.
	// If not seen, it atomically marks it as processed.
	Check(ctx context.Context, eventID string) (duplicate bool, err error)
}

// NewStore creates the best available idempotency store:
// Redis > Postgres > in-memory (dev fallback).
// When isProd is true, in-memory fallback is not allowed and the function
// returns nil with an error.
func NewStore(redisDSN, databaseURL string, ttl time.Duration, isProd bool) (Store, error) {
	if redisDSN != "" {
		return newRedisStore(redisDSN, ttl), nil
	}
	if databaseURL != "" {
		return newPostgresStore(databaseURL, ttl), nil
	}
	if isProd {
		return nil, errors.New("production requires REDIS_DSN or DATABASE_URL for idempotency; in-memory store is not allowed")
	}
	return newMemoryStore(), nil
}

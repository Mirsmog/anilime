package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func newRedisStore(dsn string, ttl time.Duration) *redisStore {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		opts = &redis.Options{Addr: dsn}
	}
	return &redisStore{
		client: redis.NewClient(opts),
		ttl:    ttl,
	}
}

func (s *redisStore) Check(ctx context.Context, eventID string) (bool, error) {
	key := "billing:idempotent:" + eventID
	set, err := s.client.SetNX(ctx, key, 1, s.ttl).Result()
	if err != nil {
		return false, err
	}
	// SetNX returns true if the key was SET (i.e. NOT a duplicate).
	return !set, nil
}

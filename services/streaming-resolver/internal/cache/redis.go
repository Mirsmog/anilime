package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	Client *redis.Client
	TTL    time.Duration
}

func NewRedisCache(url string, ttl time.Duration) (*RedisCache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	return &RedisCache{Client: client, TTL: ttl}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	val, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Client.Set(ctx, key, b, c.TTL).Err()
}

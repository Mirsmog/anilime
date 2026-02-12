package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// StripeWebhookSecret is the Stripe webhook signing secret (whsec_...).
	// Required for signature verification. Set via STRIPE_WEBHOOK_SECRET env var.
	StripeWebhookSecret string

	// NATSURL is the NATS server URL for event publishing.
	NATSURL string

	// DatabaseURL is the Postgres connection string for billing data.
	DatabaseURL string

	// RedisDSN is the Redis connection string for idempotency SETNX.
	// If empty, Postgres INSERT ON CONFLICT is used as fallback.
	RedisDSN string

	// IdempotencyTTL controls how long processed event IDs are retained.
	IdempotencyTTL time.Duration
}

func Load() (Config, error) {
	secret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if secret == "" {
		return Config{}, errors.New("STRIPE_WEBHOOK_SECRET is required")
	}

	natsURL := strings.TrimSpace(os.Getenv("NATS_URL"))
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	ttl := 24 * time.Hour
	if v := strings.TrimSpace(os.Getenv("IDEMPOTENCY_TTL_HOURS")); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			ttl = time.Duration(h) * time.Hour
		}
	}

	return Config{
		StripeWebhookSecret: secret,
		NATSURL:             natsURL,
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisDSN:            strings.TrimSpace(os.Getenv("REDIS_DSN")),
		IdempotencyTTL:      ttl,
	}, nil
}

package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the analytics consumer service.
type Config struct {
	LogLevel         string
	NATSURL          string
	PostHogAPIKey    string
	PostHogHost      string // e.g. https://app.posthog.com or self-hosted URL
	FlushInterval    time.Duration
	PostHogBatchSize int // PostHog SDK batch size before flush
	NATSBatchSize    int // NATS fetch batch size
	BatchIntervalMs  int // NATS fetch wait (ms)
}

// Load reads Config from environment variables.
func Load() (Config, error) {
	natsURL := strings.TrimSpace(os.Getenv("NATS_URL"))
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	key := strings.TrimSpace(os.Getenv("POSTHOG_API_KEY"))
	if key == "" {
		return Config{}, errors.New("POSTHOG_API_KEY is required")
	}

	host := strings.TrimSpace(os.Getenv("POSTHOG_HOST"))
	if host == "" {
		host = "https://app.posthog.com"
	}

	logLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "info"
	}

	flushSec := 5
	if v := strings.TrimSpace(os.Getenv("POSTHOG_FLUSH_INTERVAL_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			flushSec = n
		}
	}

	flushAt := 100
	if v := strings.TrimSpace(os.Getenv("POSTHOG_BATCH_SIZE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			flushAt = n
		}
	}

	batchSize := 200
	if v := strings.TrimSpace(os.Getenv("WORKER_BATCH_SIZE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batchSize = n
		}
	}

	batchIntervalMs := 2000
	if v := strings.TrimSpace(os.Getenv("WORKER_BATCH_INTERVAL_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batchIntervalMs = n
		}
	}

	return Config{
		LogLevel:         logLevel,
		NATSURL:          natsURL,
		PostHogAPIKey:    key,
		PostHogHost:      host,
		FlushInterval:    time.Duration(flushSec) * time.Second,
		PostHogBatchSize: flushAt,
		NATSBatchSize:    batchSize,
		BatchIntervalMs:  batchIntervalMs,
	}, nil
}

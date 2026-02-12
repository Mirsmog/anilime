// Package natsconn provides a shared NATS connection factory with
// configurable reconnect behaviour and fail-fast semantics.
package natsconn

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Options configures the NATS connection behaviour.
// Zero values fall back to env vars or built-in defaults.
type Options struct {
	URL           string
	MaxReconnects int           // default from NATS_MAX_RECONNECTS or 5
	ReconnectWait time.Duration // default from NATS_RECONNECT_WAIT or 2s
}

// Connect establishes a NATS connection with the configured retry policy.
// On failure after all retries it returns an error so the caller can fail-fast.
func Connect(opts Options) (*nats.Conn, error) {
	if opts.URL == "" {
		opts.URL = strings.TrimSpace(os.Getenv("NATS_URL"))
		if opts.URL == "" {
			opts.URL = "nats://nats:4222"
		}
	}
	if opts.MaxReconnects == 0 {
		opts.MaxReconnects = envInt("NATS_MAX_RECONNECTS", 5)
	}
	if opts.ReconnectWait == 0 {
		opts.ReconnectWait = envDuration("NATS_RECONNECT_WAIT", 2*time.Second)
	}

	nc, err := nats.Connect(opts.URL,
		nats.MaxReconnects(opts.MaxReconnects),
		nats.ReconnectWait(opts.ReconnectWait),
		nats.RetryOnFailedConnect(false),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect %s (max_reconnects=%d, wait=%s): %w",
			opts.URL, opts.MaxReconnects, opts.ReconnectWait, err)
	}
	return nc, nil
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

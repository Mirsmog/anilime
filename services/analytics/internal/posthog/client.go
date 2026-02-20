// Package posthog wraps the PostHog Go SDK for server-side event capture.
package posthog

import (
	"time"

	ph "github.com/posthog/posthog-go"
	"go.uber.org/zap"
)

// Client wraps posthog-go and exposes a minimal interface for the analytics service.
type Client struct {
	ph  ph.Client
	log *zap.Logger
}

// New creates a PostHog client.
// apiKey is the PostHog project API key; host is the PostHog endpoint (cloud or self-hosted).
func New(apiKey, host string, flushInterval time.Duration, batchSize int, log *zap.Logger) (*Client, error) {
	client, err := ph.NewWithConfig(apiKey, ph.Config{
		Endpoint:  host,
		BatchSize: batchSize,
		Interval:  flushInterval,
		Logger:    &zapLogger{log: log},
	})
	if err != nil {
		return nil, err
	}
	return &Client{ph: client, log: log}, nil
}

// Capture sends a single analytics event to PostHog.
// distinctID is the user_id (or anonymous ID for unauthenticated events).
func (c *Client) Capture(distinctID, event string, props map[string]any) {
	if c == nil || c.ph == nil {
		return
	}
	p := ph.NewProperties()
	for k, v := range props {
		p.Set(k, v)
	}
	if err := c.ph.Enqueue(ph.Capture{
		DistinctId: distinctID,
		Event:      event,
		Properties: p,
	}); err != nil {
		c.log.Warn("posthog: enqueue failed", zap.String("event", event), zap.Error(err))
	}
}

// Identify links a user_id to their known traits (called on registration).
func (c *Client) Identify(userID string, traits map[string]any) {
	if c == nil || c.ph == nil {
		return
	}
	p := ph.NewProperties()
	for k, v := range traits {
		p.Set(k, v)
	}
	if err := c.ph.Enqueue(ph.Identify{
		DistinctId: userID,
		Properties: p,
	}); err != nil {
		c.log.Warn("posthog: identify failed", zap.String("user_id", userID), zap.Error(err))
	}
}

// Close flushes buffered events and shuts down the client.
func (c *Client) Close() error {
	if c == nil || c.ph == nil {
		return nil
	}
	return c.ph.Close()
}

// zapLogger adapts zap to posthog-go's Logger interface.
type zapLogger struct {
	log *zap.Logger
}

func (z *zapLogger) Debugf(format string, args ...any) {
	z.log.Sugar().Debugf(format, args...)
}

func (z *zapLogger) Logf(format string, args ...any) {
	z.log.Sugar().Infof(format, args...)
}

func (z *zapLogger) Warnf(format string, args ...any) {
	z.log.Sugar().Warnf(format, args...)
}

func (z *zapLogger) Errorf(format string, args ...any) {
	z.log.Sugar().Errorf(format, args...)
}

// Package analytics provides a fire-and-forget NATS publisher for analytics events.
// All services that produce business events import this package.
package analytics

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Subject constants for every analytics event type.
const (
	SubjectAuthRegistered     = "analytics.auth.registered"
	SubjectAuthLoggedIn       = "analytics.auth.logged_in"
	SubjectStreamingStarted   = "analytics.streaming.started"
	SubjectCatalogAnimeViewed = "analytics.catalog.anime_viewed"
	SubjectSearchPerformed    = "analytics.search.performed"
)

// Event is the canonical envelope sent to all analytics.* subjects.
type Event struct {
	EventID    string         `json:"event_id"`
	EventName  string         `json:"event_name"`
	UserID     string         `json:"user_id,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
	Properties map[string]any `json:"properties,omitempty"`
}

// Publisher publishes analytics events to NATS JetStream.
// The zero value and a nil pointer are both safe no-op stubs.
type Publisher struct {
	js  nats.JetStreamContext
	log *zap.Logger
}

// New creates a Publisher using an existing JetStream context.
// Pass js=nil to get a no-op stub (useful in tests and services without NATS).
func New(js nats.JetStreamContext, log *zap.Logger) *Publisher {
	return &Publisher{js: js, log: log}
}

// Publish sends an analytics event asynchronously (fire-and-forget).
// Failures are logged as warnings and never surface to the caller.
// The publisher is safe to call with a nil receiver.
func (p *Publisher) Publish(subject, eventName, userID string, props map[string]any) {
	if p == nil || p.js == nil {
		return
	}
	ev := Event{
		EventID:    uuid.NewString(),
		EventName:  eventName,
		UserID:     userID,
		OccurredAt: time.Now().UTC(),
		Properties: props,
	}
	data, err := json.Marshal(ev)
	if err != nil {
		p.log.Warn("analytics: marshal failed", zap.String("event", eventName), zap.Error(err))
		return
	}
	if _, err := p.js.PublishAsync(subject, data); err != nil {
		p.log.Warn("analytics: publish failed", zap.String("subject", subject), zap.Error(err))
	}
}

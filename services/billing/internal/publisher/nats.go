// Package publisher provides NATS JetStream event publishing for billing.
package publisher

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	SubjectPaymentCompleted    = "billing.payment.completed"
	SubjectSubscriptionUpdated = "billing.subscription.updated"
	streamName                 = "BILLING"
)

// Publisher publishes billing events to NATS JetStream.
type Publisher struct {
	js  nats.JetStreamContext
	log *zap.Logger
}

// New connects to NATS and ensures the BILLING stream exists.
// If natsURL is empty, returns a no-op publisher (stub).
func New(natsURL string, log *zap.Logger) (*Publisher, error) {
	if natsURL == "" {
		log.Warn("NATS_URL not set, billing events will not be published (stub mode)")
		return &Publisher{log: log}, nil
	}

	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	// Create stream if it doesn't exist.
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{"billing.>"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		log.Warn("failed to create NATS stream (may already exist)", zap.Error(err))
	}

	log.Info("NATS publisher initialised", zap.String("stream", streamName))
	return &Publisher{js: js, log: log}, nil
}

// BillingEvent is the payload published to NATS.
type BillingEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// Publish sends a billing event to the given subject.
// If JetStream is not configured (stub), it logs and returns nil.
func (p *Publisher) Publish(_ context.Context, subject string, evt BillingEvent) error {
	if p.js == nil {
		p.log.Debug("NATS stub: skipping publish", zap.String("subject", subject), zap.String("event_id", evt.EventID))
		return nil
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	ack, err := p.js.Publish(subject, data)
	if err != nil {
		return err
	}

	p.log.Debug("NATS event published",
		zap.String("subject", subject),
		zap.String("event_id", evt.EventID),
		zap.Uint64("seq", ack.Sequence),
	)
	return nil
}

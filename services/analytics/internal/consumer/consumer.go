// Package consumer manages the JetStream pull consumer for the analytics service.
package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/example/anime-platform/services/analytics/internal/handler"
)

const (
	analyticsStream   = "ANALYTICS"
	analyticsConsumer = "analytics_processor"
)

// Consumer wraps a JetStream pull subscription and dispatches messages.
type Consumer struct {
	sub        *nats.Subscription
	dispatcher *handler.Dispatcher
	batchSize  int
	waitMs     time.Duration
	log        *zap.Logger
}

// New creates the ANALYTICS JetStream stream (with sources from existing streams)
// and returns a Consumer ready to call Run.
func New(nc *nats.Conn, d *handler.Dispatcher, batchSize, batchIntervalMs int, log *zap.Logger) (*Consumer, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	ensureStream(js, log)

	sub, err := js.PullSubscribe(">", analyticsConsumer, nats.BindStream(analyticsStream))
	if err != nil {
		return nil, err
	}

	return &Consumer{
		sub:        sub,
		dispatcher: d,
		batchSize:  batchSize,
		waitMs:     time.Duration(batchIntervalMs) * time.Millisecond,
		log:        log,
	}, nil
}

// Run processes messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := c.sub.Fetch(c.batchSize, nats.MaxWait(c.waitMs))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			c.log.Error("analytics consumer: fetch", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		for _, msg := range msgs {
			c.dispatcher.Dispatch(msg)
			if err := msg.Ack(); err != nil {
				c.log.Warn("analytics consumer: ack", zap.Error(err))
			}
		}
	}
}

// ensureStream creates the ANALYTICS JetStream stream if it doesn't exist.
// It sources events from ACTIVITY, BILLING, and SOCIAL streams in addition
// to its own analytics.> subjects.
func ensureStream(js nats.JetStreamContext, log *zap.Logger) {
	cfg := &nats.StreamConfig{
		Name:      analyticsStream,
		Subjects:  []string{"analytics.>"},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    30 * 24 * time.Hour,
		// Re-source existing operational streams so the analytics consumer
		// is the single read point for all business events.
		Sources: []*nats.StreamSource{
			{Name: "ACTIVITY", FilterSubject: "activity.progress"},
			{Name: "BILLING"},
			{Name: "SOCIAL", FilterSubject: "social.comments.>"},
		},
	}

	_, err := js.AddStream(cfg)
	if err == nil {
		log.Info("analytics: stream created", zap.String("stream", analyticsStream))
		return
	}

	if !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		if _, updateErr := js.UpdateStream(cfg); updateErr != nil {
			log.Warn("analytics: stream update failed (may already be up to date)", zap.Error(updateErr))
		}
	}
}

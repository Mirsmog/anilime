package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type Handlers struct {
	JikanSync   func(ctx context.Context, malID int) error
	HiAnimeSync func(ctx context.Context, malID int) error
}

type Worker struct {
	Log      *zap.Logger
	NATS     *nats.Conn
	JS       nats.JetStreamContext
	Handlers Handlers

	MaxDeliver int
}

func NewWorker(log *zap.Logger, nc *nats.Conn, handlers Handlers) (*Worker, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	w := &Worker{Log: log, NATS: nc, JS: js, Handlers: handlers, MaxDeliver: 5}
	return w, nil
}

func (w *Worker) EnsureStream(ctx context.Context) error {
	info, err := w.JS.StreamInfo("INGESTION_JOBS")
	if err == nil {
		// Ensure subjects cover ingestion.>
		needsUpdate := true
		for _, s := range info.Config.Subjects {
			if s == "ingestion.>" {
				needsUpdate = false
				break
			}
		}
		if needsUpdate {
			cfg := info.Config
			cfg.Subjects = []string{"ingestion.>"}
			_, err := w.JS.UpdateStream(&cfg)
			return err
		}
		return nil
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}
	_, err = w.JS.AddStream(&nats.StreamConfig{
		Name:     "INGESTION_JOBS",
		Subjects: []string{"ingestion.>"},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	return err
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.EnsureStream(ctx); err != nil {
		return err
	}

	// Create pull subscriptions (durable)
	jikanSub, err := w.JS.PullSubscribe("ingestion.jikan.sync", "ingestion_jikan")
	if err != nil {
		return err
	}
	hiaSub, err := w.JS.PullSubscribe("ingestion.hianime.sync", "ingestion_hianime")
	if err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go func() { errCh <- w.consumeLoop(ctx, jikanSub, "ingestion.jikan.sync") }()
	go func() { errCh <- w.consumeLoop(ctx, hiaSub, "ingestion.hianime.sync") }()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (w *Worker) consumeLoop(ctx context.Context, sub *nats.Subscription, subj string) error {
	w.Log.Info("consumer started", zap.String("subject", subj))
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := sub.Fetch(1, nats.MaxWait(2*time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			return err
		}
		for _, m := range msgs {
			_ = w.handleMsg(ctx, m, subj)
		}
	}
}

func (w *Worker) handleMsg(ctx context.Context, m *nats.Msg, subj string) error {
	md, _ := m.Metadata()
	numDelivered := uint64(1)
	if md != nil {
		numDelivered = md.NumDelivered
	}

	if w.MaxDeliver > 0 && int(numDelivered) > w.MaxDeliver {
		_ = w.publishDLQ(subj, m.Data, fmt.Sprintf("max deliveries exceeded: %d", numDelivered))
		_ = m.Ack()
		return nil
	}

	switch subj {
	case "ingestion.jikan.sync":
		var j JikanSyncJob
		if err := json.Unmarshal(m.Data, &j); err != nil {
			w.Log.Warn("bad payload", zap.String("subject", subj), zap.Error(err))
			_ = m.Ack()
			return nil
		}
		if j.MALID <= 0 {
			w.Log.Warn("bad mal_id", zap.Int("mal_id", j.MALID))
			_ = m.Ack()
			return nil
		}
		if err := w.Handlers.JikanSync(ctx, j.MALID); err != nil {
			w.Log.Warn("jikan sync failed", zap.Int("mal_id", j.MALID), zap.Uint64("attempt", numDelivered), zap.Error(err))
			_ = m.NakWithDelay(backoffDelay(numDelivered))
			return err
		}
		_ = m.Ack()
		return nil

	case "ingestion.hianime.sync":
		var j HiAnimeSyncJob
		if err := json.Unmarshal(m.Data, &j); err != nil {
			w.Log.Warn("bad payload", zap.String("subject", subj), zap.Error(err))
			_ = m.Ack()
			return nil
		}
		if j.MALID <= 0 {
			w.Log.Warn("bad mal_id", zap.Int("mal_id", j.MALID))
			_ = m.Ack()
			return nil
		}
		if err := w.Handlers.HiAnimeSync(ctx, j.MALID); err != nil {
			w.Log.Warn("hianime sync failed", zap.Int("mal_id", j.MALID), zap.Uint64("attempt", numDelivered), zap.Error(err))
			_ = m.NakWithDelay(backoffDelay(numDelivered))
			return err
		}
		_ = m.Ack()
		return nil
	default:
		_ = m.Ack()
		return nil
	}
}

func (w *Worker) publishDLQ(subject string, data []byte, reason string) error {
	msg := map[string]any{"subject": subject, "reason": reason, "payload": json.RawMessage(data)}
	b, _ := json.Marshal(msg)
	_, err := w.JS.Publish("ingestion.dlq", b)
	return err
}

package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type Publisher struct {
	Log          *zap.Logger
	DB           *pgxpool.Pool
	JS           nats.JetStreamContext
	BatchSize    int
	PollInterval time.Duration
}

type outboxRow struct {
	ID        string
	EventType string
	Payload   json.RawMessage
}

func NewPublisher(log *zap.Logger, db *pgxpool.Pool, nc *nats.Conn) (*Publisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &Publisher{
		Log:          log,
		DB:           db,
		JS:           js,
		BatchSize:    100,
		PollInterval: 2 * time.Second,
	}, nil
}

func (p *Publisher) EnsureStream(ctx context.Context) error {
	info, err := p.JS.StreamInfo("CATALOG_EVENTS")
	if err == nil {
		needsUpdate := true
		for _, s := range info.Config.Subjects {
			if s == "catalog.>" {
				needsUpdate = false
				break
			}
		}
		if needsUpdate {
			cfg := info.Config
			cfg.Subjects = []string{"catalog.>"}
			_, err := p.JS.UpdateStream(&cfg)
			return err
		}
		return nil
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}
	_, err = p.JS.AddStream(&nats.StreamConfig{
		Name:     "CATALOG_EVENTS",
		Subjects: []string{"catalog.>"},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	return err
}

func (p *Publisher) Run(ctx context.Context) error {
	if err := p.EnsureStream(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(p.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.flushOnce(ctx); err != nil {
				p.Log.Warn("outbox flush failed", zap.Error(err))
			}
		}
	}
}

func (p *Publisher) flushOnce(ctx context.Context) error {
	tx, err := p.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
SELECT id::text, event_type, payload
FROM catalog_outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED
`, p.BatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]outboxRow, 0, p.BatchSize)
	for rows.Next() {
		var item outboxRow
		if err := rows.Scan(&item.ID, &item.EventType, &item.Payload); err != nil {
			return err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if _, err := p.JS.Publish(item.EventType, item.Payload); err != nil {
			return err
		}
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	if _, err := tx.Exec(ctx, `UPDATE catalog_outbox SET published_at = now() WHERE id::text = ANY($1)`, ids); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

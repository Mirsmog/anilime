package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// ProgressEvent is the payload published by BFF for episode progress.
type ProgressEvent struct {
	EventID    string `json:"event_id"`
	UserID     string `json:"user_id"`
	AnimeID    string `json:"anime_id"`
	EpisodeID  string `json:"episode_id"`
	Position   int32  `json:"position"`
	ClientTsMs int64  `json:"client_ts_ms"`
	CreatedAt  string `json:"created_at"`
}

// StartProgressConsumer subscribes to activity.progress and applies idempotent upserts to the DB.
func StartProgressConsumer(ctx context.Context, nc *nats.Conn, pool *pgxpool.Pool) {
	js, err := nc.JetStream()
	if err != nil {
		log.Printf("progress_consumer: jetstream error: %v", err)
		return
	}

	sub, err := js.PullSubscribe("activity.progress", "activity_progress")
	if err != nil {
		log.Printf("progress_consumer: subscribe error: %v", err)
		return
	}

	go func() {
		batchSize := envInt("WORKER_BATCH_SIZE", 100)
		batchInterval := envInt("WORKER_BATCH_INTERVAL_MS", 2000)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgs, err := sub.Fetch(batchSize, nats.MaxWait(time.Duration(batchInterval)*time.Millisecond))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				log.Printf("progress_consumer: fetch error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(msgs) == 0 {
				continue
			}

			tx, err := pool.Begin(ctx)
			if err != nil {
				log.Printf("progress_consumer: db begin: %v", err)
				for _, m := range msgs {
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
				}
				continue
			}

			failed := false
			for _, m := range msgs {
				var ev ProgressEvent
				if err := json.Unmarshal(m.Data, &ev); err != nil {
					log.Printf("progress_consumer: invalid json: %v", err)
					failed = true
					break
				}

				ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "activity.progress", ev.CreatedAt, m.Data)
				if err != nil {
					if strings.Contains(err.Error(), "does not exist") {
						// fallback: apply single message without idempotency in its own tx
						_ = tx.Rollback(ctx)
						if err := applyProgressWithoutIdempotency(ctx, pool, &ev); err != nil {
							log.Printf("progress_consumer: apply without idempotency failed: %v", err)
							failed = true
						}
						if !failed {
							if err := m.Ack(); err != nil {
								log.Printf("progress_consumer: ack error: %v", err)
							}
						}
						// start a new tx for remaining messages
						tx, err = pool.Begin(ctx)
						if err != nil {
							failed = true
							break
						}
						continue
					}
					log.Printf("progress_consumer: insert processed_events error: %v", err)
					failed = true
					break
				}

				if ct.RowsAffected() == 0 {
					// already processed; skip
					continue
				}

				if err := applyProgressUpsert(ctx, tx, &ev); err != nil {
					log.Printf("progress_consumer: upsert failed: %v", err)
					failed = true
					break
				}
			}

			if failed {
				_ = tx.Rollback(ctx)
				for _, m := range msgs {
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
				}
				continue
			}

			if err := tx.Commit(ctx); err != nil {
				log.Printf("progress_consumer: commit failed: %v", err)
				for _, m := range msgs {
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
				}
				continue
			}

			for _, m := range msgs {
				if err := m.Ack(); err != nil {
					log.Printf("progress_consumer: ack error: %v", err)
				}
			}
		}
	}()
}

// applyProgressUpsert runs the idempotent upsert into user_episode_progress using provided tx.
func applyProgressUpsert(ctx context.Context, tx pgx.Tx, ev *ProgressEvent) error {
	// Note: using simple upsert; duration unknown so set to 0 and completed false unless duration known.
	q := `
INSERT INTO user_episode_progress (user_id, episode_id, position_seconds, duration_seconds, completed, client_ts_ms, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, episode_id)
DO UPDATE SET
	position_seconds = EXCLUDED.position_seconds,
	duration_seconds = EXCLUDED.duration_seconds,
	completed = EXCLUDED.completed,
	client_ts_ms = EXCLUDED.client_ts_ms,
	updated_at = EXCLUDED.updated_at
WHERE user_episode_progress.client_ts_ms <= EXCLUDED.client_ts_ms;
`
	pos := int(ev.Position)
	dur := 0
	completed := false
	now := time.Now().UTC()
	_, err := tx.Exec(ctx, q, ev.UserID, ev.EpisodeID, pos, dur, completed, ev.ClientTsMs, now)
	return err
}

// applyProgressWithoutIdempotency applies a single event without using processed_events (fallback path)
func applyProgressWithoutIdempotency(ctx context.Context, pool *pgxpool.Pool, ev *ProgressEvent) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := applyProgressUpsert(ctx, tx, ev); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

package worker

import (
	"context"
	"encoding/json"
	"log"
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
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgs, err := sub.Fetch(10, nats.MaxWait(2*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				log.Printf("progress_consumer: fetch error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, m := range msgs {
				var ev ProgressEvent
				if err := json.Unmarshal(m.Data, &ev); err != nil {
					log.Printf("progress_consumer: invalid json: %v", err)
					if err := m.Ack(); err != nil {
						log.Printf("progress_consumer: ack error: %v", err)
					}
					continue
				}

				// Begin transaction
				tx, err := pool.Begin(ctx)
				if err != nil {
					log.Printf("progress_consumer: db begin: %v", err)
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
					continue
				}

				// Try insert into processed_events for idempotency; ON CONFLICT DO NOTHING
				ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "activity.progress", ev.CreatedAt, m.Data)
				if err != nil {
					// If the processed_events table does not exist, fallback: rollback and continue without idempotency
					if strings.Contains(err.Error(), "does not exist") {
						_ = tx.Rollback(ctx)
						log.Printf("progress_consumer: processed_events table missing, applying without idempotency")
						// apply upsert below without processed_events
						tx2, err2 := pool.Begin(ctx)
						if err2 != nil {
							log.Printf("progress_consumer: db begin2: %v", err2)
							if err := m.Nak(); err != nil {
								log.Printf("progress_consumer: nak error: %v", err)
							}
							continue
						}
						if err := applyProgressUpsert(ctx, tx2, &ev); err != nil {
							_ = tx2.Rollback(ctx)
							log.Printf("progress_consumer: upsert without idempotency failed: %v", err)
							if err := m.Nak(); err != nil {
								log.Printf("progress_consumer: nak error: %v", err)
							}
							continue
						}
						if err := tx2.Commit(ctx); err != nil {
							log.Printf("progress_consumer: commit2 failed: %v", err)
							if err := m.Nak(); err != nil {
								log.Printf("progress_consumer: nak error: %v", err)
							}
							continue
						}
						if err := m.Ack(); err != nil {
							log.Printf("progress_consumer: ack error: %v", err)
						}
						continue
					}
					log.Printf("progress_consumer: insert processed_events error: %v", err)
					_ = tx.Rollback(ctx)
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
					continue
				}

				if ct.RowsAffected() == 0 {
					// already processed
					_ = tx.Rollback(ctx)
					if err := m.Ack(); err != nil {
						log.Printf("progress_consumer: ack error: %v", err)
					}
					continue
				}

				// Apply the progress upsert
				if err := applyProgressUpsert(ctx, tx, &ev); err != nil {
					_ = tx.Rollback(ctx)
					log.Printf("progress_consumer: upsert failed: %v", err)
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
					continue
				}

				if err := tx.Commit(ctx); err != nil {
					log.Printf("progress_consumer: commit failed: %v", err)
					if err := m.Nak(); err != nil {
						log.Printf("progress_consumer: nak error: %v", err)
					}
					continue
				}

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

package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// CreateCommentEvent is the payload for create
type CreateCommentEvent struct {
	EventID   string  `json:"event_id"`
	UserID    string  `json:"user_id"`
	AnimeID   string  `json:"anime_id"`
	ParentID  *string `json:"parent_id,omitempty"`
	Body      string  `json:"body"`
	CreatedAt string  `json:"created_at"`
}

// UpdateCommentEvent is the payload for update
type UpdateCommentEvent struct {
	EventID   string `json:"event_id"`
	UserID    string `json:"user_id"`
	CommentID string `json:"comment_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// DeleteCommentEvent is the payload for delete
type DeleteCommentEvent struct {
	EventID   string `json:"event_id"`
	UserID    string `json:"user_id"`
	CommentID string `json:"comment_id"`
	CreatedAt string `json:"created_at"`
}

// VoteCommentEvent is the payload for vote
type VoteCommentEvent struct {
	EventID   string `json:"event_id"`
	UserID    string `json:"user_id"`
	CommentID string `json:"comment_id"`
	Vote      int32  `json:"vote"`
	CreatedAt string `json:"created_at"`
}

// StartCommentsConsumer subscribes to social.comments.* and processes events.
func StartCommentsConsumer(ctx context.Context, nc *nats.Conn) {
	js, err := nc.JetStream()
	if err != nil {
		log.Printf("comments_consumer: jetstream: %v", err)
		return
	}

	sub, err := js.PullSubscribe("social.comments.*", "social_comments")
	if err != nil {
		log.Printf("comments_consumer: subscribe: %v", err)
		return
	}

	// Create DB pool from DATABASE_URL
	dsn := strings.TrimSpace("")
	if dsn == "" {
		// read from env
		dsn = strings.TrimSpace(getenv("DATABASE_URL"))
	}
	if dsn == "" {
		log.Printf("comments_consumer: DATABASE_URL not set")
		return
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Printf("comments_consumer: pgxpool.New: %v", err)
		return
	}
	defer pool.Close()

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
				log.Printf("comments_consumer: fetch: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(msgs) == 0 {
				continue
			}

			tx, err := pool.Begin(ctx)
			if err != nil {
				log.Printf("comments_consumer: db begin: %v", err)
				for _, m := range msgs {
					if err := m.Nak(); err != nil {
						log.Printf("comments_consumer: nak error: %v", err)
					}
				}
				continue
			}

			failed := false
			for _, m := range msgs {
				subj := m.Subject
				action := strings.TrimPrefix(subj, "social.comments.")
				switch action {
				case "create":
					var ev CreateCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid create event: %v", err)
						failed = true
						break
					}
					ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.create", ev.CreatedAt, m.Data)
					if err != nil {
						if strings.Contains(err.Error(), "does not exist") {
							_ = tx.Rollback(ctx)
							if err := handleCreate(ctx, pool, &ev); err != nil {
								log.Printf("comments_consumer: handleCreate fallback: %v", err)
								failed = true
							}
							if !failed {
								if err := m.Ack(); err != nil {
									log.Printf("comments_consumer: ack error: %v", err)
								}
							}
							tx, err = pool.Begin(ctx)
							if err != nil {
								failed = true
								break
							}
							continue
						}
						log.Printf("comments_consumer: insert processed_events error: %v", err)
						failed = true
						break
					}
					if ct.RowsAffected() == 0 {
						continue
					}
					if _, err := tx.Exec(ctx, `INSERT INTO comments (anime_id, user_id, parent_id, body) VALUES ($1,$2,$3,$4)`, ev.AnimeID, ev.UserID, ev.ParentID, ev.Body); err != nil {
						log.Printf("comments_consumer: insert comment: %v", err)
						failed = true
						break
					}
				case "update":
					var ev UpdateCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid update event: %v", err)
						failed = true
						break
					}
					ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.update", ev.CreatedAt, m.Data)
					if err != nil {
						if strings.Contains(err.Error(), "does not exist") {
							_ = tx.Rollback(ctx)
							if _, err := pool.Exec(ctx, `UPDATE comments SET body=$1, updated_at=now() WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`, ev.Body, ev.CommentID, ev.UserID); err != nil {
								log.Printf("comments_consumer: fallback update: %v", err)
								failed = true
							}
							if !failed {
								if err := m.Ack(); err != nil {
									log.Printf("comments_consumer: ack error: %v", err)
								}
							}
							tx, err = pool.Begin(ctx)
							if err != nil {
								failed = true
								break
							}
							continue
						}
						log.Printf("comments_consumer: insert processed_events error: %v", err)
						failed = true
						break
					}
					if ct.RowsAffected() == 0 {
						continue
					}
					if _, err := tx.Exec(ctx, `UPDATE comments SET body=$1, updated_at=now() WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`, ev.Body, ev.CommentID, ev.UserID); err != nil {
						log.Printf("comments_consumer: update comment: %v", err)
						failed = true
						break
					}
				case "delete":
					var ev DeleteCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid delete event: %v", err)
						failed = true
						break
					}
					ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.delete", ev.CreatedAt, m.Data)
					if err != nil {
						if strings.Contains(err.Error(), "does not exist") {
							_ = tx.Rollback(ctx)
							if _, err := pool.Exec(ctx, `UPDATE comments SET body='[deleted]', deleted_at=now() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, ev.CommentID, ev.UserID); err != nil {
								log.Printf("comments_consumer: fallback delete: %v", err)
								failed = true
							}
							if !failed {
								if err := m.Ack(); err != nil {
									log.Printf("comments_consumer: ack error: %v", err)
								}
							}
							tx, err = pool.Begin(ctx)
							if err != nil {
								failed = true
								break
							}
							continue
						}
						log.Printf("comments_consumer: insert processed_events error: %v", err)
						failed = true
						break
					}
					if ct.RowsAffected() == 0 {
						continue
					}
					if _, err := tx.Exec(ctx, `UPDATE comments SET body='[deleted]', deleted_at=now() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, ev.CommentID, ev.UserID); err != nil {
						log.Printf("comments_consumer: delete comment: %v", err)
						failed = true
						break
					}
				case "vote":
					var ev VoteCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid vote event: %v", err)
						failed = true
						break
					}
					ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.vote", ev.CreatedAt, m.Data)
					if err != nil {
						if strings.Contains(err.Error(), "does not exist") {
							// fallback: perform vote logic without idempotency
							_ = tx.Rollback(ctx)
							var exists bool
							if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)`, ev.CommentID).Scan(&exists); err != nil {
								log.Printf("comments_consumer: fallback exists: %v", err)
								failed = true
								break
							}
							if !exists {
								if err := m.Ack(); err != nil {
									log.Printf("comments_consumer: ack error: %v", err)
								}
								tx, err = pool.Begin(ctx)
								if err != nil {
									failed = true
									break
								}
								continue
							}
							fallbackTx, ftErr := pool.Begin(ctx)
							if ftErr != nil {
								failed = true
								break
							}
							var oldVote int16
							fErr := fallbackTx.QueryRow(ctx, `SELECT vote FROM comment_votes WHERE comment_id = $1 AND user_id = $2`, ev.CommentID, ev.UserID).Scan(&oldVote)
							var delta int16
							switch {
							case fErr == pgx.ErrNoRows:
								delta = int16(ev.Vote)
								_, fErr = fallbackTx.Exec(ctx, `INSERT INTO comment_votes (comment_id, user_id, vote) VALUES ($1,$2,$3)`, ev.CommentID, ev.UserID, ev.Vote)
							case fErr != nil:
								log.Printf("comments_consumer: fallback vote read error: %v", fErr)
								failed = true
							default:
								delta = int16(ev.Vote) - oldVote
								_, fErr = fallbackTx.Exec(ctx, `UPDATE comment_votes SET vote = $1 WHERE comment_id = $2 AND user_id = $3`, ev.Vote, ev.CommentID, ev.UserID)
							}
							if fErr != nil {
								log.Printf("comments_consumer: fallback vote write error: %v", fErr)
								_ = fallbackTx.Rollback(ctx)
								failed = true
								break
							}
							if delta != 0 {
								if _, fErr = fallbackTx.Exec(ctx, `UPDATE comments SET score = score + $1 WHERE id = $2`, delta, ev.CommentID); fErr != nil {
									log.Printf("comments_consumer: fallback update score error: %v", fErr)
									_ = fallbackTx.Rollback(ctx)
									failed = true
									break
								}
							}
							if err := fallbackTx.Commit(ctx); err != nil {
								log.Printf("comments_consumer: commit fallback: %v", err)
								failed = true
								break
							}
							if err := m.Ack(); err != nil {
								log.Printf("comments_consumer: ack error: %v", err)
							}
							tx, err = pool.Begin(ctx)
							if err != nil {
								failed = true
								break
							}
							continue
						}
						log.Printf("comments_consumer: insert processed_events error: %v", err)
						failed = true
						break
					}
					if ct.RowsAffected() == 0 {
						continue
					}
					var exists bool
					if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)`, ev.CommentID).Scan(&exists); err != nil {
						log.Printf("comments_consumer: exists check: %v", err)
						failed = true
						break
					}
					if !exists {
						continue
					}
					var oldVote int16
					err = tx.QueryRow(ctx, `SELECT vote FROM comment_votes WHERE comment_id = $1 AND user_id = $2`, ev.CommentID, ev.UserID).Scan(&oldVote)
					var delta int16
					switch {
					case err == pgx.ErrNoRows:
						delta = int16(ev.Vote)
						_, err = tx.Exec(ctx, `INSERT INTO comment_votes (comment_id, user_id, vote) VALUES ($1,$2,$3)`, ev.CommentID, ev.UserID, ev.Vote)
					case err != nil:
						log.Printf("comments_consumer: vote read error: %v", err)
						failed = true
					default:
						delta = int16(ev.Vote) - oldVote
						_, err = tx.Exec(ctx, `UPDATE comment_votes SET vote = $1 WHERE comment_id = $2 AND user_id = $3`, ev.Vote, ev.CommentID, ev.UserID)
					}
					if err != nil {
						log.Printf("comments_consumer: vote write error: %v", err)
						failed = true
						break
					}
					if delta != 0 {
						if _, err = tx.Exec(ctx, `UPDATE comments SET score = score + $1 WHERE id = $2`, delta, ev.CommentID); err != nil {
							log.Printf("comments_consumer: update score error: %v", err)
							failed = true
							break
						}
					}
				default:
					// unknown action
					log.Printf("comments_consumer: unknown action: %s", action)
					failed = true
				}
			}

			if failed {
				_ = tx.Rollback(ctx)
				for _, m := range msgs {
					if err := m.Nak(); err != nil {
						log.Printf("comments_consumer: nak error: %v", err)
					}
				}
				continue
			}

			if err := tx.Commit(ctx); err != nil {
				log.Printf("comments_consumer: commit failed: %v", err)
				for _, m := range msgs {
					if err := m.Nak(); err != nil {
						log.Printf("comments_consumer: nak error: %v", err)
					}
				}
				continue
			}

			for _, m := range msgs {
				if err := m.Ack(); err != nil {
					log.Printf("comments_consumer: ack error: %v", err)
				}
			}
		}
	}()
}

func getenv(k string) string {
	return strings.TrimSpace(os.Getenv(k))
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

func handleCreate(ctx context.Context, pool *pgxpool.Pool, ev *CreateCommentEvent) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.create", ev.CreatedAt, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			// fallback: create without idempotency
			if _, err := tx.Exec(ctx, `INSERT INTO comments (anime_id, user_id, parent_id, body) VALUES ($1,$2,$3,$4)`, ev.AnimeID, ev.UserID, ev.ParentID, ev.Body); err != nil {
				return err
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		// already processed
		return nil
	}

	if _, err := tx.Exec(ctx, `INSERT INTO comments (anime_id, user_id, parent_id, body) VALUES ($1,$2,$3,$4)`, ev.AnimeID, ev.UserID, ev.ParentID, ev.Body); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

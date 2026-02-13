package worker

import (
	"context"
	"encoding/json"
	"log"
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
				log.Printf("comments_consumer: fetch: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, m := range msgs {
				subj := m.Subject
				action := strings.TrimPrefix(subj, "social.comments.")
				switch action {
				case "create":
					var ev CreateCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid create event: %v", err)
						if err := m.Ack(); err != nil {
							log.Printf("comments_consumer: ack error: %v", err)
						}
						continue
					}
					if err := handleCreate(ctx, pool, &ev); err != nil {
						log.Printf("comments_consumer: handleCreate: %v", err)
						// do not ack to allow retry
						continue
					}
					if err := m.Ack(); err != nil {
						log.Printf("comments_consumer: ack error: %v", err)
					}
				case "update":
					var ev UpdateCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid update event: %v", err)
						if err := m.Ack(); err != nil {
							log.Printf("comments_consumer: ack error: %v", err)
						}
						continue
					}
					if err := handleUpdate(ctx, pool, &ev); err != nil {
						log.Printf("comments_consumer: handleUpdate: %v", err)
						continue
					}
					if err := m.Ack(); err != nil {
						log.Printf("comments_consumer: ack error: %v", err)
					}
				case "delete":
					var ev DeleteCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid delete event: %v", err)
						if err := m.Ack(); err != nil {
							log.Printf("comments_consumer: ack error: %v", err)
						}
						continue
					}
					if err := handleDelete(ctx, pool, &ev); err != nil {
						log.Printf("comments_consumer: handleDelete: %v", err)
						continue
					}
					if err := m.Ack(); err != nil {
						log.Printf("comments_consumer: ack error: %v", err)
					}
				case "vote":
					var ev VoteCommentEvent
					if err := json.Unmarshal(m.Data, &ev); err != nil {
						log.Printf("comments_consumer: invalid vote event: %v", err)
						if err := m.Ack(); err != nil {
							log.Printf("comments_consumer: ack error: %v", err)
						}
						continue
					}
					if err := handleVote(ctx, pool, &ev); err != nil {
						log.Printf("comments_consumer: handleVote: %v", err)
						continue
					}
					if err := m.Ack(); err != nil {
						log.Printf("comments_consumer: ack error: %v", err)
					}
				default:
					if err := m.Ack(); err != nil {
						log.Printf("comments_consumer: ack error: %v", err)
					}
				}
			}
		}
	}()
}

func getenv(k string) string {
	return strings.TrimSpace(os.Getenv(k))
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

func handleUpdate(ctx context.Context, pool *pgxpool.Pool, ev *UpdateCommentEvent) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.update", ev.CreatedAt, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			if _, err := tx.Exec(ctx, `UPDATE comments SET body=$1, updated_at=now() WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`, ev.Body, ev.CommentID, ev.UserID); err != nil {
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
		return nil
	}

	if _, err := tx.Exec(ctx, `UPDATE comments SET body=$1, updated_at=now() WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`, ev.Body, ev.CommentID, ev.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func handleDelete(ctx context.Context, pool *pgxpool.Pool, ev *DeleteCommentEvent) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.delete", ev.CreatedAt, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			if _, err := tx.Exec(ctx, `UPDATE comments SET body='[deleted]', deleted_at=now() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, ev.CommentID, ev.UserID); err != nil {
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
		return nil
	}

	if _, err := tx.Exec(ctx, `UPDATE comments SET body='[deleted]', deleted_at=now() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, ev.CommentID, ev.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func handleVote(ctx context.Context, pool *pgxpool.Pool, ev *VoteCommentEvent) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `INSERT INTO processed_events (event_id, subject, created_at, payload) VALUES ($1,$2,$3,$4) ON CONFLICT (event_id) DO NOTHING`, ev.EventID, "social.comments.vote", ev.CreatedAt, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			// fallback: perform vote logic without idempotency
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)`, ev.CommentID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				if err := tx.Commit(ctx); err != nil {
					return err
				}
				return nil
			}

			var oldVote int16
			err = tx.QueryRow(ctx, `SELECT vote FROM comment_votes WHERE comment_id = $1 AND user_id = $2`, ev.CommentID, ev.UserID).Scan(&oldVote)
			var delta int16
			switch {
			case err == pgx.ErrNoRows:
				delta = int16(ev.Vote)
				_, err = tx.Exec(ctx, `INSERT INTO comment_votes (comment_id, user_id, vote) VALUES ($1,$2,$3)`, ev.CommentID, ev.UserID, ev.Vote)
			case err != nil:
				return err
			default:
				delta = int16(ev.Vote) - oldVote
				_, err = tx.Exec(ctx, `UPDATE comment_votes SET vote = $1 WHERE comment_id = $2 AND user_id = $3`, ev.Vote, ev.CommentID, ev.UserID)
			}
			if err != nil {
				return err
			}
			if delta != 0 {
				_, err = tx.Exec(ctx, `UPDATE comments SET score = score + $1 WHERE id = $2`, delta, ev.CommentID)
				if err != nil {
					return err
				}
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		return nil
	}

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)`, ev.CommentID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return nil
	}

	var oldVote int16
	err = tx.QueryRow(ctx, `SELECT vote FROM comment_votes WHERE comment_id = $1 AND user_id = $2`, ev.CommentID, ev.UserID).Scan(&oldVote)

	var delta int16
	switch {
	case err == pgx.ErrNoRows:
		delta = int16(ev.Vote)
		_, err = tx.Exec(ctx, `INSERT INTO comment_votes (comment_id, user_id, vote) VALUES ($1,$2,$3)`, ev.CommentID, ev.UserID, ev.Vote)
	case err != nil:
		return err
	default:
		delta = int16(ev.Vote) - oldVote
		_, err = tx.Exec(ctx, `UPDATE comment_votes SET vote = $1 WHERE comment_id = $2 AND user_id = $3`, ev.Vote, ev.CommentID, ev.UserID)
	}
	if err != nil {
		return err
	}

	if delta != 0 {
		_, err = tx.Exec(ctx, `UPDATE comments SET score = score + $1 WHERE id = $2`, delta, ev.CommentID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

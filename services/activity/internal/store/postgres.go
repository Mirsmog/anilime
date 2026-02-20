package store

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PostgresProgressRepository is the production Postgres-backed implementation.
type PostgresProgressRepository struct {
	db *pgxpool.Pool
}

func NewPostgresProgressRepository(db *pgxpool.Pool) *PostgresProgressRepository {
	return &PostgresProgressRepository{db: db}
}

func (r *PostgresProgressRepository) Upsert(ctx context.Context, rec ProgressRecord) (ProgressRecord, error) {
	q := `
INSERT INTO user_episode_progress (user_id, episode_id, position_seconds, duration_seconds, completed, client_ts_ms, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, episode_id)
DO UPDATE SET
  position_seconds = EXCLUDED.position_seconds,
  duration_seconds = EXCLUDED.duration_seconds,
  completed        = EXCLUDED.completed,
  client_ts_ms     = EXCLUDED.client_ts_ms,
  updated_at       = EXCLUDED.updated_at
WHERE user_episode_progress.client_ts_ms <= EXCLUDED.client_ts_ms
RETURNING position_seconds, duration_seconds, completed, client_ts_ms, updated_at`

	var out ProgressRecord
	out.UserID = rec.UserID
	out.EpisodeID = rec.EpisodeID

	err := r.db.QueryRow(ctx, q,
		rec.UserID, rec.EpisodeID, rec.PositionSeconds, rec.DurationSeconds,
		rec.Completed, rec.ClientTsMs, time.Now().UTC(),
	).Scan(&out.PositionSeconds, &out.DurationSeconds, &out.Completed, &out.ClientTsMs, &out.UpdatedAt)

	if err != nil {
		// WHERE clause blocked the update; fetch current state instead.
		if errors.Is(err, pgx.ErrNoRows) {
			return r.fetchOne(ctx, rec.UserID, rec.EpisodeID)
		}
		return ProgressRecord{}, status.Error(codes.Internal, "db")
	}
	return out, nil
}

func (r *PostgresProgressRepository) fetchOne(ctx context.Context, userID, episodeID uuid.UUID) (ProgressRecord, error) {
	q := `SELECT position_seconds, duration_seconds, completed, client_ts_ms, updated_at
	      FROM user_episode_progress WHERE user_id=$1 AND episode_id=$2`
	var out ProgressRecord
	out.UserID = userID
	out.EpisodeID = episodeID
	err := r.db.QueryRow(ctx, q, userID, episodeID).
		Scan(&out.PositionSeconds, &out.DurationSeconds, &out.Completed, &out.ClientTsMs, &out.UpdatedAt)
	if err != nil {
		return ProgressRecord{}, status.Error(codes.Internal, "db")
	}
	return out, nil
}

func (r *PostgresProgressRepository) List(ctx context.Context, userID uuid.UUID, limit int, cursor *ProgressCursor) ([]ProgressRecord, error) {
	q := `SELECT episode_id, position_seconds, duration_seconds, completed, client_ts_ms, updated_at
	      FROM user_episode_progress WHERE user_id=$1`
	args := []any{userID}

	if cursor != nil {
		q += " AND (updated_at, episode_id) < (to_timestamp($2 / 1000.0), $3)"
		args = append(args, cursor.UpdatedAt.UnixMilli(), cursor.EpisodeID)
	}
	q += " ORDER BY updated_at DESC, episode_id DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, status.Error(codes.Internal, "db")
	}
	defer rows.Close()

	var out []ProgressRecord
	for rows.Next() {
		var rec ProgressRecord
		rec.UserID = userID
		if err := rows.Scan(&rec.EpisodeID, &rec.PositionSeconds, &rec.DurationSeconds, &rec.Completed, &rec.ClientTsMs, &rec.UpdatedAt); err != nil {
			return nil, status.Error(codes.Internal, "db")
		}
		out = append(out, rec)
	}
	return out, nil
}

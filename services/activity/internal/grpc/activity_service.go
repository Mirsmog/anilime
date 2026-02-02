package grpcapi

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	activityv1 "github.com/example/anime-platform/gen/activity/v1"
)

type ActivityService struct {
	activityv1.UnimplementedActivityServiceServer
	DB *pgxpool.Pool
}

const completedThreshold = 0.90

func (s *ActivityService) UpsertEpisodeProgress(ctx context.Context, req *activityv1.UpsertEpisodeProgressRequest) (*activityv1.UpsertEpisodeProgressResponse, error) {
	userID, err := uuid.Parse(strings.TrimSpace(req.GetUserId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	epID, err := uuid.Parse(strings.TrimSpace(req.GetEpisodeId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid episode_id")
	}

	pos := int(req.GetPositionSeconds())
	dur := int(req.GetDurationSeconds())
	if pos < 0 {
		pos = 0
	}
	if dur < 0 {
		dur = 0
	}
	clientTS := req.GetClientTsMs()

	completed := false
	if dur > 0 {
		completed = float64(pos)/float64(dur) >= completedThreshold
	}

	now := time.Now().UTC()

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
WHERE user_episode_progress.client_ts_ms <= EXCLUDED.client_ts_ms
RETURNING position_seconds, duration_seconds, completed, client_ts_ms, updated_at;
`

	var (
		retPos, retDur int
		retCompleted   bool
		retClientTS    int64
		updatedAt      time.Time
	)

	err = s.DB.QueryRow(ctx, q, userID, epID, pos, dur, completed, clientTS, now).Scan(&retPos, &retDur, &retCompleted, &retClientTS, &updatedAt)
	if err != nil {
		// If WHERE prevented update, RETURNING returns no rows.
		if errors.Is(err, pgx.ErrNoRows) {
			// Fetch current state
			getQ := `SELECT position_seconds, duration_seconds, completed, client_ts_ms, updated_at FROM user_episode_progress WHERE user_id=$1 AND episode_id=$2;`
			if err := s.DB.QueryRow(ctx, getQ, userID, epID).Scan(&retPos, &retDur, &retCompleted, &retClientTS, &updatedAt); err != nil {
				return nil, status.Error(codes.Internal, "db")
			}
		} else {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				_ = pgErr
			}
			return nil, status.Error(codes.Internal, "db")
		}
	}

	resp := &activityv1.UpsertEpisodeProgressResponse{Progress: &activityv1.EpisodeProgress{
		UserId:          userID.String(),
		EpisodeId:       epID.String(),
		PositionSeconds: int32(retPos),
		DurationSeconds: int32(retDur),
		Completed:       retCompleted,
		UpdatedAtMs:     updatedAt.UnixMilli(),
		ClientTsMs:      retClientTS,
	}}
	return resp, nil
}

func (s *ActivityService) GetContinueWatching(ctx context.Context, req *activityv1.GetContinueWatchingRequest) (*activityv1.GetContinueWatchingResponse, error) {
	userID, err := uuid.Parse(strings.TrimSpace(req.GetUserId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	// Cursor format: base64("<tsMs>:<episodeUUID>")
	var (
		cursorTS      int64
		cursorEpisode uuid.UUID
		useCursor     bool
	)
	if c := strings.TrimSpace(req.GetCursor()); c != "" {
		b, err := base64.RawURLEncoding.DecodeString(c)
		if err == nil {
			parts := strings.SplitN(string(b), ":", 2)
			if len(parts) == 2 {
				if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
					if eid, err := uuid.Parse(parts[1]); err == nil {
						cursorTS = ts
						cursorEpisode = eid
						useCursor = true
					}
				}
			}
		}
	}

	qBase := `
SELECT episode_id::text, position_seconds, duration_seconds, completed, client_ts_ms, updated_at
FROM user_episode_progress
WHERE user_id=$1
`
	args := []any{userID}
	if useCursor {
		qBase += " AND (updated_at, episode_id) < (to_timestamp($2 / 1000.0), $3)"
		args = append(args, cursorTS, cursorEpisode)
	}
	qBase += " ORDER BY updated_at DESC, episode_id DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.DB.Query(ctx, qBase, args...)
	if err != nil {
		return nil, status.Error(codes.Internal, "db")
	}
	defer rows.Close()

	resp := &activityv1.GetContinueWatchingResponse{Limit: int32(limit)}
	var lastUpdated time.Time
	var lastEpisode string
	for rows.Next() {
		var (
			epID      string
			pos, dur  int
			completed bool
			clientTS  int64
			updatedAt time.Time
		)
		if err := rows.Scan(&epID, &pos, &dur, &completed, &clientTS, &updatedAt); err != nil {
			return nil, status.Error(codes.Internal, "db")
		}
		resp.Items = append(resp.Items, &activityv1.ContinueItem{Progress: &activityv1.EpisodeProgress{
			UserId:          userID.String(),
			EpisodeId:       epID,
			PositionSeconds: int32(pos),
			DurationSeconds: int32(dur),
			Completed:       completed,
			UpdatedAtMs:     updatedAt.UnixMilli(),
			ClientTsMs:      clientTS,
		}})
		lastUpdated = updatedAt
		lastEpisode = epID
	}

	if len(resp.Items) == limit {
		// generate next cursor from last item
		payload := strconv.FormatInt(lastUpdated.UnixMilli(), 10) + ":" + lastEpisode
		resp.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(payload))
	}
	return resp, nil
}

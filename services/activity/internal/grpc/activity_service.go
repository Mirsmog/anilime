package grpcapi

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	activityv1 "github.com/example/anime-platform/gen/activity/v1"
	"github.com/example/anime-platform/services/activity/internal/store"
)

// completedThreshold is the watch ratio at which an episode is marked completed.
const completedThreshold = 0.90

type ActivityService struct {
	activityv1.UnimplementedActivityServiceServer
	Progress store.ProgressRepository
}

func (s *ActivityService) UpsertEpisodeProgress(ctx context.Context, req *activityv1.UpsertEpisodeProgressRequest) (*activityv1.UpsertEpisodeProgressResponse, error) {
	userID, err := uuid.Parse(strings.TrimSpace(req.GetUserId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	epID, err := uuid.Parse(strings.TrimSpace(req.GetEpisodeId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid episode_id")
	}

	pos := clampMin(int(req.GetPositionSeconds()), 0)
	dur := clampMin(int(req.GetDurationSeconds()), 0)

	rec := store.ProgressRecord{
		UserID:          userID,
		EpisodeID:       epID,
		PositionSeconds: pos,
		DurationSeconds: dur,
		Completed:       dur > 0 && float64(pos)/float64(dur) >= completedThreshold,
		ClientTsMs:      req.GetClientTsMs(),
	}

	out, err := s.Progress.Upsert(ctx, rec)
	if err != nil {
		return nil, err
	}
	return &activityv1.UpsertEpisodeProgressResponse{Progress: toProtoProgress(out)}, nil
}

func (s *ActivityService) GetContinueWatching(ctx context.Context, req *activityv1.GetContinueWatchingRequest) (*activityv1.GetContinueWatchingResponse, error) {
	userID, err := uuid.Parse(strings.TrimSpace(req.GetUserId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	limit := clampLimit(int(req.GetLimit()), 25, 100)
	cursor := decodeCursor(req.GetCursor())

	records, err := s.Progress.List(ctx, userID, limit, cursor)
	if err != nil {
		return nil, err
	}

	resp := &activityv1.GetContinueWatchingResponse{Limit: int32(limit)}
	for _, r := range records {
		resp.Items = append(resp.Items, &activityv1.ContinueItem{Progress: toProtoProgress(r)})
	}
	if len(records) == limit {
		last := records[len(records)-1]
		resp.NextCursor = encodeCursor(last.UpdatedAt.UnixMilli(), last.EpisodeID.String())
	}
	return resp, nil
}

func toProtoProgress(r store.ProgressRecord) *activityv1.EpisodeProgress {
	return &activityv1.EpisodeProgress{
		UserId:          r.UserID.String(),
		EpisodeId:       r.EpisodeID.String(),
		PositionSeconds: int32(r.PositionSeconds),
		DurationSeconds: int32(r.DurationSeconds),
		Completed:       r.Completed,
		UpdatedAtMs:     r.UpdatedAt.UnixMilli(),
		ClientTsMs:      r.ClientTsMs,
	}
}

// encodeCursor encodes updated_at millis and episode UUID as a base64 opaque cursor.
func encodeCursor(tsMs int64, episodeID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(tsMs, 10) + ":" + episodeID))
}

// decodeCursor parses the opaque cursor produced by encodeCursor.
func decodeCursor(raw string) *store.ProgressCursor {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil
	}
	parts := strings.SplitN(string(b), ":", 2)
	if len(parts) != 2 {
		return nil
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil
	}
	eid, err := uuid.Parse(parts[1])
	if err != nil {
		return nil
	}
	return &store.ProgressCursor{
		UpdatedAt: time.UnixMilli(ts).UTC(),
		EpisodeID: eid,
	}
}

func clampLimit(v, def, maxVal int) int {
	if v <= 0 {
		return def
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

func clampMin(v, minVal int) int {
	if v < minVal {
		return minVal
	}
	return v
}

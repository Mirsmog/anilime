package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	activityv1 "github.com/example/anime-platform/gen/activity/v1"
	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type upsertProgressRequest struct {
	EpisodeID       string `json:"episode_id"`
	PositionSeconds int32  `json:"position_seconds"`
	DurationSeconds int32  `json:"duration_seconds"`
	ClientTsMs      int64  `json:"client_ts_ms"`
}

type continueEpisode struct {
	EpisodeID string `json:"episode_id"`
	AnimeID   string `json:"anime_id"`
	Number    int32  `json:"number"`
	Title     string `json:"title"`
	AiredAt   string `json:"aired_at"`
}

type continueItem struct {
	Episode  continueEpisode `json:"episode"`
	Progress struct {
		PositionSeconds int32 `json:"position_seconds"`
		DurationSeconds int32 `json:"duration_seconds"`
		Completed       bool  `json:"completed"`
		UpdatedAtMs     int64 `json:"updated_at_ms"`
		ClientTsMs      int64 `json:"client_ts_ms"`
	} `json:"progress"`
}

type continueResponse struct {
	Items      []continueItem `json:"items"`
	Limit      int32          `json:"limit"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

func UpsertProgress(activity activityv1.ActivityServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		uid, ok := auth.UserIDFromContext(r.Context())
		if !ok || strings.TrimSpace(uid) == "" {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		var req upsertProgressRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "Invalid JSON", rid, nil)
			return
		}
		if req.ClientTsMs == 0 {
			req.ClientTsMs = time.Now().UnixMilli()
		}

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := activity.UpsertEpisodeProgress(ctx, &activityv1.UpsertEpisodeProgressRequest{
			UserId:          uid,
			EpisodeId:       strings.TrimSpace(req.EpisodeID),
			PositionSeconds: req.PositionSeconds,
			DurationSeconds: req.DurationSeconds,
			ClientTsMs:      req.ClientTsMs,
		})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		api.WriteJSON(w, http.StatusOK, resp.GetProgress())
	}
}

func ContinueWatching(activity activityv1.ActivityServiceClient, catalog catalogv1.CatalogServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		uid, ok := auth.UserIDFromContext(r.Context())
		if !ok || strings.TrimSpace(uid) == "" {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		limit := int32(25)
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				if n < 1 {
					n = 1
				}
				if n > 100 {
					n = 100
				}
				limit = int32(n)
			}
		}
		cursor := r.URL.Query().Get("cursor")

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		cw, err := activity.GetContinueWatching(ctx, &activityv1.GetContinueWatchingRequest{UserId: uid, Limit: limit, Cursor: cursor})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		epIDs := make([]string, 0, len(cw.GetItems()))
		for _, it := range cw.GetItems() {
			if it.GetProgress() != nil {
				epIDs = append(epIDs, it.GetProgress().GetEpisodeId())
			}
		}

		meta := map[string]*catalogv1.Episode{}
		if len(epIDs) > 0 {
			resp, err := catalog.GetEpisodesByIDs(ctx, &catalogv1.GetEpisodesByIDsRequest{EpisodeIds: epIDs})
			if err != nil {
				writeGRPCError(w, rid, err)
				return
			}
			for _, e := range resp.GetEpisodes() {
				meta[e.GetId()] = e
			}
		}

		out := continueResponse{Limit: cw.GetLimit(), NextCursor: cw.GetNextCursor()}
		for _, it := range cw.GetItems() {
			p := it.GetProgress()
			if p == nil {
				continue
			}
			e := meta[p.GetEpisodeId()]
			ci := continueItem{}
			if e != nil {
				ci.Episode = continueEpisode{EpisodeID: e.GetId(), AnimeID: e.GetAnimeId(), Number: e.GetNumber(), Title: e.GetTitle(), AiredAt: e.GetAiredAtRfc3339()}
			} else {
				ci.Episode = continueEpisode{EpisodeID: p.GetEpisodeId()}
			}
			ci.Progress.PositionSeconds = p.GetPositionSeconds()
			ci.Progress.DurationSeconds = p.GetDurationSeconds()
			ci.Progress.Completed = p.GetCompleted()
			ci.Progress.UpdatedAtMs = p.GetUpdatedAtMs()
			ci.Progress.ClientTsMs = p.GetClientTsMs()
			out.Items = append(out.Items, ci)
		}

		api.WriteJSON(w, http.StatusOK, out)
	}
}

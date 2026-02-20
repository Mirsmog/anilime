package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type rateReq struct {
	Score int32 `json:"score"`
}

// GetRating returns the aggregate rating for an anime, including the caller's
// own score when authenticated.
func GetRating(client socialv1.SocialServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		animeID := chi.URLParam(r, "anime_id")
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", rid, nil)
			return
		}

		ctx := r.Context()
		if mdCtx, ok := withUserMD(r); ok {
			ctx = mdCtx
		}

		resp, err := client.GetRating(ctx, &socialv1.GetRatingRequest{AnimeId: animeID})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		out := map[string]any{
			"anime_id": animeID,
			"average":  resp.GetAverage(),
			"count":    resp.GetCount(),
		}
		if resp.UserScore != nil {
			out["user_score"] = resp.GetUserScore()
		}

		api.WriteJSON(w, http.StatusOK, out)
	}
}

// RateAnime submits or updates the authenticated user's rating for an anime.
func RateAnime(client socialv1.SocialServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		animeID := chi.URLParam(r, "anime_id")
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", rid, nil)
			return
		}

		var req rateReq
		if !decodeJSON(w, r, rid, &req) {
			return
		}

		ctx, ok := withUserMD(r)
		if !ok {
			api.Unauthorized(w, "AUTH_MISSING", "authentication required", rid)
			return
		}

		resp, err := client.RateAnime(ctx, &socialv1.RateAnimeRequest{
			AnimeId: animeID,
			Score:   req.Score,
		})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		api.WriteJSON(w, http.StatusOK, map[string]any{
			"anime_id": animeID,
			"average":  resp.GetAverage(),
			"count":    resp.GetCount(),
		})
	}
}

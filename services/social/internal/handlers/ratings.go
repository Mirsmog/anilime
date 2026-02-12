package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/services/social/internal/store"
)

type rateRequest struct {
	UserID string `json:"user_id"`
	Score  int    `json:"score"`
}

type getRatingsResponse struct {
	store.RatingSummary
	UserScore *int `json:"user_score,omitempty"`
}

// GetRatings returns the rating summary for an anime.
func GetRatings(s store.RatingStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", "", nil)
			return
		}
		summary, err := s.GetSummary(r.Context(), animeID)
		if err != nil {
			api.Internal(w, "")
			return
		}
		resp := getRatingsResponse{RatingSummary: summary}

		// If user_id query param is provided, include their rating.
		if uid := strings.TrimSpace(r.URL.Query().Get("user_id")); uid != "" {
			if score, ok, err := s.GetUserRating(r.Context(), animeID, uid); err == nil && ok {
				resp.UserScore = &score
			}
		}
		api.WriteJSON(w, http.StatusOK, resp)
	}
}

// PostRating upserts a rating for an anime.
func PostRating(s store.RatingStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", "", nil)
			return
		}

		var req rateRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "Invalid JSON", "", nil)
			return
		}
		if strings.TrimSpace(req.UserID) == "" {
			api.BadRequest(w, "MISSING_USER_ID", "user_id is required", "", nil)
			return
		}
		if req.Score < 1 || req.Score > 10 {
			api.BadRequest(w, "INVALID_SCORE", "score must be between 1 and 10", "", nil)
			return
		}

		if err := s.Upsert(r.Context(), animeID, req.UserID, req.Score); err != nil {
			api.Internal(w, "")
			return
		}
		summary, err := s.GetSummary(r.Context(), animeID)
		if err != nil {
			api.Internal(w, "")
			return
		}
		api.WriteJSON(w, http.StatusOK, summary)
	}
}

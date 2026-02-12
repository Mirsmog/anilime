package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/services/social/internal/store"
)

type createCommentRequest struct {
	Body     string  `json:"body"`
	ParentID *string `json:"parent_id,omitempty"`
}

type updateCommentRequest struct {
	Body string `json:"body"`
}

type voteRequest struct {
	Vote int16 `json:"vote"`
}

type threadResponse struct {
	Comments   []store.CommentTreeNode `json:"comments"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

// CreateComment handles POST /v1/comments/{anime_id}
func CreateComment(cs store.CommentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			api.Unauthorized(w, "UNAUTHORIZED", "authentication required", "")
			return
		}

		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", "", nil)
			return
		}

		var req createCommentRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "invalid JSON", "", nil)
			return
		}
		if strings.TrimSpace(req.Body) == "" {
			api.BadRequest(w, "EMPTY_BODY", "body must not be empty", "", nil)
			return
		}

		c := store.Comment{
			AnimeID:  animeID,
			UserID:   userID,
			ParentID: req.ParentID,
			Body:     req.Body,
		}

		created, err := cs.Create(r.Context(), c)
		if err != nil {
			api.Internal(w, "")
			return
		}
		api.WriteJSON(w, http.StatusCreated, created)
	}
}

// GetThread handles GET /v1/comments/{anime_id}
func GetThread(cs store.CommentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", "", nil)
			return
		}

		sortParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort")))
		if sortParam != store.SortTop {
			sortParam = "new"
		}

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))

		nodes, nextCursor, err := cs.GetThread(r.Context(), animeID, sortParam, limit, cursor)
		if err != nil {
			api.Internal(w, "")
			return
		}

		api.WriteJSON(w, http.StatusOK, threadResponse{
			Comments:   nodes,
			NextCursor: nextCursor,
		})
	}
}

// VoteComment handles POST /v1/comments/{comment_id}/vote
func VoteComment(cs store.CommentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			api.Unauthorized(w, "UNAUTHORIZED", "authentication required", "")
			return
		}

		commentID := strings.TrimSpace(chi.URLParam(r, "comment_id"))
		if commentID == "" {
			api.BadRequest(w, "MISSING_ID", "comment_id is required", "", nil)
			return
		}

		var req voteRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "invalid JSON", "", nil)
			return
		}
		if req.Vote != 1 && req.Vote != -1 {
			api.BadRequest(w, "INVALID_VOTE", "vote must be 1 or -1", "", nil)
			return
		}

		if err := cs.Vote(r.Context(), commentID, userID, req.Vote); err != nil {
			if err == store.ErrNotFoundOrForbidden {
				api.NotFound(w, "NOT_FOUND", "comment not found", "")
				return
			}
			api.Internal(w, "")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateComment handles PUT /v1/comments/{comment_id}
func UpdateComment(cs store.CommentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			api.Unauthorized(w, "UNAUTHORIZED", "authentication required", "")
			return
		}

		commentID := strings.TrimSpace(chi.URLParam(r, "comment_id"))
		if commentID == "" {
			api.BadRequest(w, "MISSING_ID", "comment_id is required", "", nil)
			return
		}

		var req updateCommentRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			api.BadRequest(w, "INVALID_JSON", "invalid JSON", "", nil)
			return
		}
		if strings.TrimSpace(req.Body) == "" {
			api.BadRequest(w, "EMPTY_BODY", "body must not be empty", "", nil)
			return
		}

		if err := cs.UpdateBody(r.Context(), commentID, userID, req.Body); err != nil {
			if err == store.ErrNotFoundOrForbidden {
				api.Forbidden(w, "FORBIDDEN", "not found or not the author", "")
				return
			}
			api.Internal(w, "")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteComment handles DELETE /v1/comments/{comment_id}
func DeleteComment(cs store.CommentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			api.Unauthorized(w, "UNAUTHORIZED", "authentication required", "")
			return
		}

		commentID := strings.TrimSpace(chi.URLParam(r, "comment_id"))
		if commentID == "" {
			api.BadRequest(w, "MISSING_ID", "comment_id is required", "", nil)
			return
		}

		if err := cs.SoftDelete(r.Context(), commentID, userID); err != nil {
			if err == store.ErrNotFoundOrForbidden {
				api.Forbidden(w, "FORBIDDEN", "not found or not the author", "")
				return
			}
			api.Internal(w, "")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

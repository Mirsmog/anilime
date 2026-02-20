package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/metadata"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type createCommentReq struct {
	Body     string  `json:"body"`
	ParentID *string `json:"parent_id,omitempty"`
}

type updateCommentReq struct {
	Body string `json:"body"`
}

type voteReq struct {
	Vote int32 `json:"vote"`
}

func withUserMD(r *http.Request) (context.Context, bool) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || strings.TrimSpace(uid) == "" {
		return nil, false
	}
	md := metadata.New(map[string]string{"user_id": uid})
	return metadata.NewOutgoingContext(r.Context(), md), true
}

// tryPublishAsync publishes an event and writes 202 Accepted.
// Returns true if handled (success or publish error); false if publisher is disabled.
func tryPublishAsync(w http.ResponseWriter, rid, subject string, payload map[string]any, publisher *EventPublisher) bool {
	if publisher == nil || !publisher.Enabled() {
		return false
	}
	eventID, err := publisher.PublishJSON(subject, payload)
	if err != nil {
		api.WriteError(w, http.StatusServiceUnavailable, "EVENT_PUBLISH_FAILED", "failed to publish event", rid, nil)
		return true
	}
	w.Header().Set("X-Event-ID", eventID)
	w.WriteHeader(http.StatusAccepted)
	return true
}

func CreateComment(client socialv1.SocialServiceClient, publisher *EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		ctx, ok := withUserMD(r)
		if !ok {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", rid, nil)
			return
		}

		var req createCommentReq
		if !decodeJSON(w, r, rid, &req) {
			return
		}

		uid, _ := auth.UserIDFromContext(r.Context())
		payload := map[string]any{"user_id": uid, "anime_id": animeID, "body": req.Body}
		if req.ParentID != nil {
			payload["parent_id"] = *req.ParentID
		}
		if tryPublishAsync(w, rid, "social.comments.create", payload, publisher) {
			return
		}

		pbReq := &socialv1.CreateCommentRequest{
			AnimeId: animeID,
			Body:    req.Body,
		}
		if req.ParentID != nil {
			pbReq.ParentId = req.ParentID
		}

		resp, err := client.CreateComment(ctx, pbReq)
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		api.WriteJSON(w, http.StatusCreated, resp.GetComment())
	}
}

func ListComments(client socialv1.SocialServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", rid, nil)
			return
		}

		sortParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort")))
		limit := int32(50)
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
				limit = int32(n)
			}
		}
		cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := client.ListComments(ctx, &socialv1.ListCommentsRequest{
			AnimeId: animeID,
			Sort:    sortParam,
			Limit:   limit,
			Cursor:  cursor,
		})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		api.WriteJSON(w, http.StatusOK, resp)
	}
}

func VoteComment(client socialv1.SocialServiceClient, publisher *EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		ctx, ok := withUserMD(r)
		if !ok {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		commentID := strings.TrimSpace(chi.URLParam(r, "comment_id"))
		if commentID == "" {
			api.BadRequest(w, "MISSING_ID", "comment_id is required", rid, nil)
			return
		}

		var req voteReq
		if !decodeJSON(w, r, rid, &req) {
			return
		}

		uid, _ := auth.UserIDFromContext(r.Context())
		if tryPublishAsync(w, rid, "social.comments.vote", map[string]any{
			"user_id": uid, "comment_id": commentID, "vote": req.Vote,
		}, publisher) {
			return
		}

		_, err := client.VoteComment(ctx, &socialv1.VoteCommentRequest{
			CommentId: commentID,
			Vote:      req.Vote,
		})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func UpdateComment(client socialv1.SocialServiceClient, publisher *EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		ctx, ok := withUserMD(r)
		if !ok {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		commentID := strings.TrimSpace(chi.URLParam(r, "comment_id"))
		if commentID == "" {
			api.BadRequest(w, "MISSING_ID", "comment_id is required", rid, nil)
			return
		}

		var req updateCommentReq
		if !decodeJSON(w, r, rid, &req) {
			return
		}

		uid, _ := auth.UserIDFromContext(r.Context())
		if tryPublishAsync(w, rid, "social.comments.update", map[string]any{
			"user_id": uid, "comment_id": commentID, "body": req.Body,
		}, publisher) {
			return
		}

		_, err := client.UpdateComment(ctx, &socialv1.UpdateCommentRequest{
			CommentId: commentID,
			Body:      req.Body,
		})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func DeleteComment(client socialv1.SocialServiceClient, publisher *EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		ctx, ok := withUserMD(r)
		if !ok {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		commentID := strings.TrimSpace(chi.URLParam(r, "comment_id"))
		if commentID == "" {
			api.BadRequest(w, "MISSING_ID", "comment_id is required", rid, nil)
			return
		}

		uid, _ := auth.UserIDFromContext(r.Context())
		if tryPublishAsync(w, rid, "social.comments.delete", map[string]any{
			"user_id": uid, "comment_id": commentID,
		}, publisher) {
			return
		}

		_, err := client.DeleteComment(ctx, &socialv1.DeleteCommentRequest{
			CommentId: commentID,
		})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/services/social/internal/store"
)

// setupReq builds a request with chi URL params and optional user_id in context.
func setupReq(method, url string, body string, params map[string]string, userID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = auth.WithUserID(ctx, userID)
	}
	return req.WithContext(ctx)
}

func TestCreateComment(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	handler := CreateComment(cs)

	req := setupReq(http.MethodPost, "/v1/comments/anime-1", `{"body":"hello world"}`,
		map[string]string{"anime_id": "anime-1"}, "user-a")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var c store.Comment
	if err := json.NewDecoder(rr.Body).Decode(&c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.Body != "hello world" {
		t.Fatalf("expected body 'hello world', got %q", c.Body)
	}
	if c.UserID != "user-a" {
		t.Fatalf("expected user_id 'user-a', got %q", c.UserID)
	}
}

func TestCreateComment_Unauthorized(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	handler := CreateComment(cs)

	req := setupReq(http.MethodPost, "/v1/comments/anime-1", `{"body":"hello"}`,
		map[string]string{"anime_id": "anime-1"}, "")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestCreateComment_EmptyBody(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	handler := CreateComment(cs)

	req := setupReq(http.MethodPost, "/v1/comments/anime-1", `{"body":""}`,
		map[string]string{"anime_id": "anime-1"}, "user-a")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetThread(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	ctx := context.Background()
	_, _ = cs.Create(ctx, store.Comment{AnimeID: "anime-1", UserID: "user-a", Body: "root"})

	handler := GetThread(cs)
	req := setupReq(http.MethodGet, "/v1/comments/anime-1?sort=new&limit=10", "",
		map[string]string{"anime_id": "anime-1"}, "")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp threadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(resp.Comments))
	}
}

func TestVoteComment(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	ctx := context.Background()
	c, _ := cs.Create(ctx, store.Comment{AnimeID: "anime-1", UserID: "user-a", Body: "voteable"})

	handler := VoteComment(cs)
	req := setupReq(http.MethodPost, "/v1/comments/"+c.ID+"/vote", `{"vote":1}`,
		map[string]string{"comment_id": c.ID}, "user-b")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestVoteComment_InvalidVote(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	ctx := context.Background()
	c, _ := cs.Create(ctx, store.Comment{AnimeID: "anime-1", UserID: "user-a", Body: "voteable"})

	handler := VoteComment(cs)
	req := setupReq(http.MethodPost, "/v1/comments/"+c.ID+"/vote", `{"vote":2}`,
		map[string]string{"comment_id": c.ID}, "user-b")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUpdateComment_AuthorOnly(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	ctx := context.Background()
	c, _ := cs.Create(ctx, store.Comment{AnimeID: "anime-1", UserID: "user-a", Body: "original"})

	handler := UpdateComment(cs)

	// Non-author: forbidden
	req := setupReq(http.MethodPut, "/v1/comments/"+c.ID, `{"body":"hacked"}`,
		map[string]string{"comment_id": c.ID}, "user-b")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-author, got %d", rr.Code)
	}

	// Author: success
	req = setupReq(http.MethodPut, "/v1/comments/"+c.ID, `{"body":"updated"}`,
		map[string]string{"comment_id": c.ID}, "user-a")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for author, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteComment_AuthorOnly(t *testing.T) {
	cs := store.NewInMemoryCommentStore()
	ctx := context.Background()
	c, _ := cs.Create(ctx, store.Comment{AnimeID: "anime-1", UserID: "user-a", Body: "will delete"})

	handler := DeleteComment(cs)

	// Non-author: forbidden
	req := setupReq(http.MethodDelete, "/v1/comments/"+c.ID, "",
		map[string]string{"comment_id": c.ID}, "user-b")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-author, got %d", rr.Code)
	}

	// Author: success
	req = setupReq(http.MethodDelete, "/v1/comments/"+c.ID, "",
		map[string]string{"comment_id": c.ID}, "user-a")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for author, got %d: %s", rr.Code, rr.Body.String())
	}
}

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
	"github.com/example/anime-platform/internal/platform/auth"
)

// ─── stub social client ───────────────────────────────────────────────────────

type stubSocialClient struct {
	socialv1.SocialServiceClient
	getRatingResp *socialv1.GetRatingResponse
	getRatingErr  error
	rateAnimeResp *socialv1.RateAnimeResponse
	rateAnimeErr  error
}

func (s *stubSocialClient) GetRating(_ context.Context, _ *socialv1.GetRatingRequest, _ ...grpc.CallOption) (*socialv1.GetRatingResponse, error) {
	return s.getRatingResp, s.getRatingErr
}

func (s *stubSocialClient) RateAnime(_ context.Context, _ *socialv1.RateAnimeRequest, _ ...grpc.CallOption) (*socialv1.RateAnimeResponse, error) {
	return s.rateAnimeResp, s.rateAnimeErr
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// ratingReq builds a request with the anime_id chi param set.
func ratingReq(method, url, animeID string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("anime_id", animeID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// asAuthUser injects user-1 into the request context.
func asAuthUser(req *http.Request) *http.Request {
	return req.WithContext(auth.WithUserID(req.Context(), "user-1"))
}

// ─── GetRating tests ─────────────────────────────────────────────────────────

func TestGetRating_OK(t *testing.T) {
	score := int32(8)
	stub := &stubSocialClient{
		getRatingResp: &socialv1.GetRatingResponse{Average: 8.5, Count: 10, UserScore: &score},
	}

	req := ratingReq(http.MethodGet, "/v1/anime/anime-1/rating", "anime-1", nil)
	rr := httptest.NewRecorder()
	GetRating(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["average"] != 8.5 {
		t.Fatalf("expected average 8.5, got %v", resp["average"])
	}
	if resp["user_score"] == nil {
		t.Fatal("expected user_score in response")
	}
}

func TestGetRating_MissingAnimeID(t *testing.T) {
	req := ratingReq(http.MethodGet, "/v1/anime//rating", "", nil)
	rr := httptest.NewRecorder()
	GetRating(&stubSocialClient{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetRating_GRPCError(t *testing.T) {
	stub := &stubSocialClient{getRatingErr: status.Error(codes.Internal, "db error")}
	req := ratingReq(http.MethodGet, "/v1/anime/anime-1/rating", "anime-1", nil)
	rr := httptest.NewRecorder()
	GetRating(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestGetRating_NoUserScore_NotPresent(t *testing.T) {
	stub := &stubSocialClient{
		getRatingResp: &socialv1.GetRatingResponse{Average: 7.0, Count: 5},
	}
	req := ratingReq(http.MethodGet, "/v1/anime/anime-1/rating", "anime-1", nil)
	rr := httptest.NewRecorder()
	GetRating(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if _, exists := resp["user_score"]; exists {
		t.Fatal("expected user_score absent when user has not rated")
	}
}

// ─── RateAnime tests ─────────────────────────────────────────────────────────

func TestRateAnime_OK(t *testing.T) {
	stub := &stubSocialClient{
		rateAnimeResp: &socialv1.RateAnimeResponse{Average: 8.0, Count: 1},
	}
	body, _ := json.Marshal(map[string]int{"score": 8})
	req := asAuthUser(ratingReq(http.MethodPost, "/v1/anime/anime-1/rating", "anime-1", body))

	rr := httptest.NewRecorder()
	RateAnime(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["average"] != 8.0 {
		t.Fatalf("expected average 8.0, got %v", resp["average"])
	}
}

func TestRateAnime_MissingAnimeID(t *testing.T) {
	body, _ := json.Marshal(map[string]int{"score": 8})
	req := asAuthUser(ratingReq(http.MethodPost, "/v1/anime//rating", "", body))

	rr := httptest.NewRecorder()
	RateAnime(&stubSocialClient{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRateAnime_Unauthenticated(t *testing.T) {
	body, _ := json.Marshal(map[string]int{"score": 8})
	req := ratingReq(http.MethodPost, "/v1/anime/anime-1/rating", "anime-1", body)
	// No auth injected

	rr := httptest.NewRecorder()
	RateAnime(&stubSocialClient{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRateAnime_InvalidBody(t *testing.T) {
	req := asAuthUser(ratingReq(http.MethodPost, "/v1/anime/anime-1/rating", "anime-1", []byte("not json")))

	rr := httptest.NewRecorder()
	RateAnime(&stubSocialClient{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestRateAnime_GRPCValidationError(t *testing.T) {
	stub := &stubSocialClient{
		rateAnimeErr: status.Error(codes.InvalidArgument, "score must be 1-10"),
	}
	body, _ := json.Marshal(map[string]int{"score": 99})
	req := asAuthUser(ratingReq(http.MethodPost, "/v1/anime/anime-1/rating", "anime-1", body))

	rr := httptest.NewRecorder()
	RateAnime(stub).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

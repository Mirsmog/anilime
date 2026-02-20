package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

type stubCatalogClient struct {
	catalogv1.CatalogServiceClient

	getAnimeByIDsResp        *catalogv1.GetAnimeByIDsResponse
	getAnimeByIDsErr         error
	getEpisodesByIDsResp     *catalogv1.GetEpisodesByIDsResponse
	getEpisodesByIDsErr      error
	getEpisodesByAnimeIDResp *catalogv1.GetEpisodesByAnimeIDResponse
	getEpisodesByAnimeIDErr  error
	getAnimeIDsResp          *catalogv1.GetAnimeIDsResponse
	getAnimeIDsErr           error
}

func (s *stubCatalogClient) GetAnimeByIDs(_ context.Context, _ *catalogv1.GetAnimeByIDsRequest, _ ...grpc.CallOption) (*catalogv1.GetAnimeByIDsResponse, error) {
	return s.getAnimeByIDsResp, s.getAnimeByIDsErr
}

func (s *stubCatalogClient) GetEpisodesByIDs(_ context.Context, _ *catalogv1.GetEpisodesByIDsRequest, _ ...grpc.CallOption) (*catalogv1.GetEpisodesByIDsResponse, error) {
	return s.getEpisodesByIDsResp, s.getEpisodesByIDsErr
}

func (s *stubCatalogClient) GetEpisodesByAnimeID(_ context.Context, _ *catalogv1.GetEpisodesByAnimeIDRequest, _ ...grpc.CallOption) (*catalogv1.GetEpisodesByAnimeIDResponse, error) {
	return s.getEpisodesByAnimeIDResp, s.getEpisodesByAnimeIDErr
}

func (s *stubCatalogClient) GetAnimeIDs(_ context.Context, _ *catalogv1.GetAnimeIDsRequest, _ ...grpc.CallOption) (*catalogv1.GetAnimeIDsResponse, error) {
	return s.getAnimeIDsResp, s.getAnimeIDsErr
}

func chiReq(url string, params map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestGetAnime_OK(t *testing.T) {
	stub := &stubCatalogClient{
		getAnimeByIDsResp: &catalogv1.GetAnimeByIDsResponse{
			Anime: []*catalogv1.Anime{{Id: "a1", Title: "Steins;Gate", Score: 9.1}},
		},
	}
	handler := GetAnime(stub, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime/a1", map[string]string{"anime_id": "a1"}))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp animeResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID != "a1" || resp.Title != "Steins;Gate" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetAnime_NotFound(t *testing.T) {
	stub := &stubCatalogClient{
		getAnimeByIDsResp: &catalogv1.GetAnimeByIDsResponse{Anime: nil},
	}
	handler := GetAnime(stub, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime/missing", map[string]string{"anime_id": "missing"}))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetAnime_MissingID(t *testing.T) {
	stub := &stubCatalogClient{}
	handler := GetAnime(stub, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime/", map[string]string{"anime_id": ""}))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetAnime_GRPCError(t *testing.T) {
	stub := &stubCatalogClient{
		getAnimeByIDsErr: status.Error(codes.Internal, "db error"),
	}
	handler := GetAnime(stub, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime/a1", map[string]string{"anime_id": "a1"}))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestGetEpisode_OK(t *testing.T) {
	stub := &stubCatalogClient{
		getEpisodesByIDsResp: &catalogv1.GetEpisodesByIDsResponse{
			Episodes: []*catalogv1.Episode{{Id: "e1", AnimeId: "a1", Number: 1, Title: "Prologue"}},
		},
	}
	handler := GetEpisode(stub)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/episodes/e1", map[string]string{"episode_id": "e1"}))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp episodeResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID != "e1" || resp.Title != "Prologue" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetEpisode_NotFound(t *testing.T) {
	stub := &stubCatalogClient{
		getEpisodesByIDsResp: &catalogv1.GetEpisodesByIDsResponse{Episodes: nil},
	}
	handler := GetEpisode(stub)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/episodes/missing", map[string]string{"episode_id": "missing"}))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetEpisodesByAnime_OK(t *testing.T) {
	stub := &stubCatalogClient{
		getEpisodesByAnimeIDResp: &catalogv1.GetEpisodesByAnimeIDResponse{
			Episodes: []*catalogv1.Episode{
				{Id: "e1", AnimeId: "a1", Number: 1, Title: "Ep 1"},
				{Id: "e2", AnimeId: "a1", Number: 2, Title: "Ep 2"},
			},
		},
	}
	handler := GetEpisodesByAnime(stub)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime/a1/episodes", map[string]string{"anime_id": "a1"}))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Episodes []episodeResponse `json:"episodes"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(resp.Episodes))
	}
}

func TestGetEpisodesByAnime_MissingID(t *testing.T) {
	stub := &stubCatalogClient{}
	handler := GetEpisodesByAnime(stub)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime//episodes", map[string]string{"anime_id": ""}))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestListAnime_OK(t *testing.T) {
	stub := &stubCatalogClient{
		getAnimeIDsResp: &catalogv1.GetAnimeIDsResponse{AnimeIds: []string{"a1", "a2", "a3"}},
		getAnimeByIDsResp: &catalogv1.GetAnimeByIDsResponse{
			Anime: []*catalogv1.Anime{
				{Id: "a1", Title: "Anime 1"},
				{Id: "a2", Title: "Anime 2"},
			},
		},
	}
	handler := ListAnime(stub, NewTTLCache(0, nil, ""))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime?limit=2&offset=0", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Anime  []animeResponse `json:"anime"`
		Total  int32           `json:"total"`
		Limit  int32           `json:"limit"`
		Offset int32           `json:"offset"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Anime) != 2 {
		t.Fatalf("expected 2 anime, got %d", len(resp.Anime))
	}
}

func TestListAnime_OffsetBeyondTotal(t *testing.T) {
	stub := &stubCatalogClient{
		getAnimeIDsResp: &catalogv1.GetAnimeIDsResponse{AnimeIds: []string{"a1"}},
	}
	handler := ListAnime(stub, NewTTLCache(0, nil, ""))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, chiReq("/v1/anime?offset=100", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Anime []animeResponse `json:"anime"`
		Total int32           `json:"total"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Anime) != 0 {
		t.Fatalf("expected empty anime list, got %d", len(resp.Anime))
	}
}

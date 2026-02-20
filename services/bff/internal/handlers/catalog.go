package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/metadata"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type animeResponse struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english,omitempty"`
	TitleJapanese string   `json:"title_japanese,omitempty"`
	Image         string   `json:"image,omitempty"`
	Description   string   `json:"description,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Score         float32  `json:"score"`
	Status        string   `json:"status,omitempty"`
	Type          string   `json:"type,omitempty"`
	TotalEpisodes int32    `json:"total_episodes"`
}

type episodeResponse struct {
	ID      string `json:"id"`
	AnimeID string `json:"anime_id"`
	Number  int32  `json:"number"`
	Title   string `json:"title"`
	AiredAt string `json:"aired_at,omitempty"`
}

func toAnimeResponse(a *catalogv1.Anime) animeResponse {
	return animeResponse{
		ID:            a.GetId(),
		Title:         a.GetTitle(),
		TitleEnglish:  a.GetTitleEnglish(),
		TitleJapanese: a.GetTitleJapanese(),
		Image:         a.GetImage(),
		Description:   a.GetDescription(),
		Genres:        a.GetGenres(),
		Score:         a.GetScore(),
		Status:        a.GetStatus(),
		Type:          a.GetType(),
		TotalEpisodes: a.GetTotalEpisodes(),
	}
}

func toEpisodeResponse(e *catalogv1.Episode) episodeResponse {
	return episodeResponse{
		ID:      e.GetId(),
		AnimeID: e.GetAnimeId(),
		Number:  e.GetNumber(),
		Title:   e.GetTitle(),
		AiredAt: e.GetAiredAtRfc3339(),
	}
}

// GetAnime handles GET /v1/anime/{anime_id}
func GetAnime(catalog catalogv1.CatalogServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", rid, nil)
			return
		}

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := catalog.GetAnimeByIDs(ctx, &catalogv1.GetAnimeByIDsRequest{AnimeIds: []string{animeID}})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		if len(resp.GetAnime()) == 0 {
			api.NotFound(w, "NOT_FOUND", "anime not found", rid)
			return
		}

		api.WriteJSON(w, http.StatusOK, toAnimeResponse(resp.GetAnime()[0]))
	}
}

// GetEpisodesByAnime handles GET /v1/anime/{anime_id}/episodes
func GetEpisodesByAnime(catalog catalogv1.CatalogServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		animeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if animeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", rid, nil)
			return
		}

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := catalog.GetEpisodesByAnimeID(ctx, &catalogv1.GetEpisodesByAnimeIDRequest{AnimeId: animeID})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		episodes := make([]episodeResponse, 0, len(resp.GetEpisodes()))
		for _, e := range resp.GetEpisodes() {
			episodes = append(episodes, toEpisodeResponse(e))
		}

		api.WriteJSON(w, http.StatusOK, map[string]any{"episodes": episodes})
	}
}

// GetEpisode handles GET /v1/episodes/{episode_id}
func GetEpisode(catalog catalogv1.CatalogServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		episodeID := strings.TrimSpace(chi.URLParam(r, "episode_id"))
		if episodeID == "" {
			api.BadRequest(w, "MISSING_ID", "episode_id is required", rid, nil)
			return
		}

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := catalog.GetEpisodesByIDs(ctx, &catalogv1.GetEpisodesByIDsRequest{EpisodeIds: []string{episodeID}})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		if len(resp.GetEpisodes()) == 0 {
			api.NotFound(w, "NOT_FOUND", "episode not found", rid)
			return
		}

		api.WriteJSON(w, http.StatusOK, toEpisodeResponse(resp.GetEpisodes()[0]))
	}
}

// ListAnime handles GET /v1/anime?limit=N&offset=M
func ListAnime(catalog catalogv1.CatalogServiceClient, cache Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())

		limit := parseInt32(r.URL.Query().Get("limit"), 25, 1, 100)
		offset := parseInt32(r.URL.Query().Get("offset"), 0, 0, 10000)

		// Try cache first
		key := fmt.Sprintf("ListAnime:%d:%d", limit, offset)
		if cached, ok := cache.Get(key); ok {
			api.WriteJSON(w, http.StatusOK, cached)
			return
		}

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		idsResp, err := catalog.GetAnimeIDs(ctx, &catalogv1.GetAnimeIDsRequest{})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		allIDs := idsResp.GetAnimeIds()
		total := int32(len(allIDs))

		if offset >= total {
			resp := map[string]any{"anime": []any{}, "total": total, "limit": limit, "offset": offset}
			cache.Set(key, resp)
			api.WriteJSON(w, http.StatusOK, resp)
			return
		}

		end := offset + limit
		if end > total {
			end = total
		}
		pageIDs := allIDs[offset:end]

		animeResp, err := catalog.GetAnimeByIDs(ctx, &catalogv1.GetAnimeByIDsRequest{AnimeIds: pageIDs})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}

		items := make([]animeResponse, 0, len(animeResp.GetAnime()))
		for _, a := range animeResp.GetAnime() {
			items = append(items, toAnimeResponse(a))
		}

		resp := map[string]any{"anime": items, "total": total, "limit": limit, "offset": offset}
		cache.Set(key, resp)
		api.WriteJSON(w, http.StatusOK, resp)
	}
}

package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"

	searchv1 "github.com/example/anime-platform/gen/search/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type searchResponse struct {
	Hits  []*searchv1.AnimeHit `json:"hits"`
	Total int32                `json:"total"`
}

func Search(search searchv1.SearchServiceClient, cache Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		limit := parseInt32(r.URL.Query().Get("limit"), 25, 1, 100)
		offset := parseInt32(r.URL.Query().Get("offset"), 0, 0, 10000)
		genres := splitList(r.URL.Query().Get("genres"))
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		animeType := strings.TrimSpace(r.URL.Query().Get("type"))
		minScore := parseFloat32(r.URL.Query().Get("min_score"), 0)
		maxScore := parseFloat32(r.URL.Query().Get("max_score"), 0)

		// Cache key based on raw query
		key := "Search:" + r.URL.RawQuery
		if cached, ok := cache.Get(key); ok {
			api.WriteJSON(w, http.StatusOK, cached)
			return
		}

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := search.SearchAnime(ctx, &searchv1.SearchAnimeRequest{Query: q, Limit: limit, Offset: offset, Genres: genres, Status: status, Type: animeType, MinScore: minScore, MaxScore: maxScore})
		if err != nil {
			writeGRPCError(w, rid, err)
			return
		}
		out := searchResponse{Hits: resp.GetHits(), Total: resp.GetTotal()}
		cache.Set(key, out)
		api.WriteJSON(w, http.StatusOK, out)
	}
}

func splitList(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseInt32(v string, def, min, max int32) int32 {
	if strings.TrimSpace(v) == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if i < int(min) {
		return min
	}
	if i > int(max) {
		return max
	}
	return int32(i)
}

func parseFloat32(v string, def float32) float32 {
	if strings.TrimSpace(v) == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return def
	}
	return float32(f)
}

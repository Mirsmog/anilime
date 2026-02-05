package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	searchv1 "github.com/example/anime-platform/gen/search/v1"
	"github.com/example/anime-platform/services/search/internal/meili"
)

type SearchService struct {
	searchv1.UnimplementedSearchServiceServer
	Meili *meili.Client
}

type meiliAnime struct {
	AnimeID       string   `json:"anime_id"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	Image         string   `json:"image"`
	Description   string   `json:"description"`
	Genres        []string `json:"genres"`
	Score         float32  `json:"score"`
	Status        string   `json:"status"`
	Type          string   `json:"type"`
	TotalEpisodes int32    `json:"total_episodes"`
}

func (s *SearchService) SearchAnime(ctx context.Context, req *searchv1.SearchAnimeRequest) (*searchv1.SearchAnimeResponse, error) {
	q := strings.TrimSpace(req.GetQuery())
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}

	filters := buildFilters(req)

	payload := map[string]any{"q": q, "limit": limit, "offset": offset}
	if filters != "" {
		payload["filter"] = filters
	}

	resp, err := s.Meili.Search(ctx, "anime", payload)
	if err != nil {
		return nil, status.Error(codes.Internal, "search backend")
	}

	out := &searchv1.SearchAnimeResponse{Total: int32(resp.EstimatedTotalHits)}
	for _, hit := range resp.Hits {
		var doc meiliAnime
		if err := json.Unmarshal(hit, &doc); err != nil {
			continue
		}
		out.Hits = append(out.Hits, &searchv1.AnimeHit{
			AnimeId:       doc.AnimeID,
			Title:         doc.Title,
			TitleEnglish:  doc.TitleEnglish,
			TitleJapanese: doc.TitleJapanese,
			Image:         doc.Image,
			Description:   doc.Description,
			Genres:        doc.Genres,
			Score:         doc.Score,
			Status:        doc.Status,
			Type:          doc.Type,
			TotalEpisodes: doc.TotalEpisodes,
		})
	}
	return out, nil
}

func buildFilters(req *searchv1.SearchAnimeRequest) string {
	filters := []string{}
	if len(req.GetGenres()) > 0 {
		vals := make([]string, 0, len(req.GetGenres()))
		for _, g := range req.GetGenres() {
			g = strings.TrimSpace(g)
			if g != "" {
				vals = append(vals, "genres = \""+g+"\"")
			}
		}
		if len(vals) > 0 {
			filters = append(filters, "("+strings.Join(vals, " OR ")+")")
		}
	}
	if v := strings.TrimSpace(req.GetStatus()); v != "" {
		filters = append(filters, "status = \""+v+"\"")
	}
	if v := strings.TrimSpace(req.GetType()); v != "" {
		filters = append(filters, "type = \""+v+"\"")
	}
	if req.GetMinScore() > 0 {
		filters = append(filters, "score >= "+formatFloat(req.GetMinScore()))
	}
	if req.GetMaxScore() > 0 {
		filters = append(filters, "score <= "+formatFloat(req.GetMaxScore()))
	}
	return strings.Join(filters, " AND ")
}

func formatFloat(v float32) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

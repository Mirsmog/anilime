package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	searchv1 "github.com/example/anime-platform/gen/search/v1"
)

// JikanFallback searches Jikan when local search has no results,
// returns provisional hits and asynchronously enqueues ingestion.
type JikanFallback interface {
	Search(ctx context.Context, q string, limit int32) ([]*searchv1.AnimeHit, error)
}

type jikanFallbackClient struct {
	baseURL    string
	httpClient *http.Client
	js         nats.JetStreamContext
}

// NewJikanFallback creates a JikanFallback that queries Jikan and triggers ingestion via NATS.
func NewJikanFallback(baseURL string, js nats.JetStreamContext) JikanFallback {
	if baseURL == "" {
		baseURL = "https://api.jikan.moe/v4"
	}
	return &jikanFallbackClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		js:         js,
	}
}

func (c *jikanFallbackClient) Search(ctx context.Context, q string, limit int32) ([]*searchv1.AnimeHit, error) {
	if limit <= 0 {
		limit = 10
	}
	rawURL := fmt.Sprintf("%s/anime?q=%s&limit=%d&order_by=popularity&sort=asc",
		c.baseURL, url.QueryEscape(q), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anime-platform-bff/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jikan fallback: status %d", resp.StatusCode)
	}

	var out jikanListResp
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	hits := make([]*searchv1.AnimeHit, 0, len(out.Data))
	malIDs := make([]int32, 0, len(out.Data))
	for _, a := range out.Data {
		if a.MalID <= 0 {
			continue
		}
		genres := make([]string, 0, len(a.Genres))
		for _, g := range a.Genres {
			if name := strings.TrimSpace(g.Name); name != "" {
				genres = append(genres, name)
			}
		}
		hits = append(hits, &searchv1.AnimeHit{
			AnimeId:       "", // not yet in local catalog
			Title:         strings.TrimSpace(a.Title),
			TitleEnglish:  strings.TrimSpace(a.TitleEnglish),
			TitleJapanese: strings.TrimSpace(a.TitleJapanese),
			Image:         strings.TrimSpace(a.Images.JPG.LargeImageURL),
			Description:   strings.TrimSpace(a.Synopsis),
			Genres:        genres,
			Score:         a.Score,
			Status:        strings.TrimSpace(a.Status),
			Type:          strings.TrimSpace(a.Type),
			TotalEpisodes: a.Episodes,
		})
		malIDs = append(malIDs, a.MalID)
	}

	// Enqueue ingestion asynchronously so results appear in local search next time.
	if c.js != nil && len(malIDs) > 0 {
		go func() {
			for _, id := range malIDs {
				msg, _ := json.Marshal(map[string]any{"mal_id": id})
				_, _ = c.js.Publish("ingestion.jikan.sync", msg)
			}
		}()
	}

	return hits, nil
}

type jikanListResp struct {
	Data []struct {
		MalID         int32   `json:"mal_id"`
		Title         string  `json:"title"`
		TitleEnglish  string  `json:"title_english"`
		TitleJapanese string  `json:"title_japanese"`
		Synopsis      string  `json:"synopsis"`
		Type          string  `json:"type"`
		Status        string  `json:"status"`
		Episodes      int32   `json:"episodes"`
		Score         float32 `json:"score"`
		Genres        []struct {
			Name string `json:"name"`
		} `json:"genres"`
		Images struct {
			JPG struct {
				LargeImageURL string `json:"large_image_url"`
			} `json:"jpg"`
		} `json:"images"`
	} `json:"data"`
}

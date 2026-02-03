package hianime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://void-roan-six.vercel.app/api/v2"
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: &http.Client{Timeout: 10 * time.Second}}
}

type SearchResponse struct {
	Status int `json:"status"`
	Data   struct {
		Animes []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			JName string `json:"jname"`
		} `json:"animes"`
	} `json:"data"`
}

type AnimeInfoResponse struct {
	Status int `json:"status"`
	Data   struct {
		Anime struct {
			Info struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				MalID     int    `json:"malId"`
				AnilistID int    `json:"anilistId"`
			} `json:"info"`
		} `json:"anime"`
	} `json:"data"`
}

type EpisodesResponse struct {
	Status int `json:"status"`
	Data   struct {
		TotalEpisodes int `json:"totalEpisodes"`
		Episodes      []struct {
			Title     string `json:"title"`
			EpisodeID string `json:"episodeId"`
			Number    int32  `json:"number"`
			IsFiller  bool   `json:"isFiller"`
		} `json:"episodes"`
	} `json:"data"`
}

func (c *Client) Search(ctx context.Context, q string, page int) (*SearchResponse, error) {
	u, _ := url.Parse(c.BaseURL + "/hianime/search")
	qq := u.Query()
	qq.Set("q", q)
	if page > 0 {
		qq.Set("page", fmt.Sprintf("%d", page))
	}
	u.RawQuery = qq.Encode()
	return doJSON[SearchResponse](ctx, c.HTTPClient, u.String())
}

func (c *Client) GetAnime(ctx context.Context, slug string) (*AnimeInfoResponse, error) {
	u := c.BaseURL + "/hianime/anime/" + url.PathEscape(slug)
	return doJSON[AnimeInfoResponse](ctx, c.HTTPClient, u)
}

func (c *Client) GetEpisodes(ctx context.Context, slug string) (*EpisodesResponse, error) {
	u := c.BaseURL + "/hianime/anime/" + url.PathEscape(slug) + "/episodes"
	return doJSON[EpisodesResponse](ctx, c.HTTPClient, u)
}

func doJSON[T any](ctx context.Context, hc *http.Client, u string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anime-platform-ingestion/1.0")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hianime: status %d body=%q", resp.StatusCode, string(b[:min(len(b), 200)]))
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("hianime: decode error: %w body=%q", err, string(b[:min(len(b), 200)]))
	}
	return &out, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

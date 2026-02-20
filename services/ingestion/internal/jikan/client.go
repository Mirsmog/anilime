package jikan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.jikan.moe/v4"
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: &http.Client{Timeout: 10 * time.Second}}
}

// AnimeData is the shared data block returned by single and list endpoints.
type AnimeData struct {
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
}

type AnimeResponse struct {
	Data AnimeData `json:"data"`
}

type AnimeListResponse struct {
	Data       []AnimeData `json:"data"`
	Pagination struct {
		HasNextPage bool `json:"has_next_page"`
	} `json:"pagination"`
}

func (c *Client) GetAnime(ctx context.Context, malID int) (*AnimeResponse, error) {
	if malID <= 0 {
		return nil, fmt.Errorf("malID required")
	}

	u, _ := url.Parse(c.BaseURL + "/anime/" + strconv.Itoa(malID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anime-platform-ingestion/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jikan: status %d body=%q", resp.StatusCode, string(b[:min(len(b), 200)]))
	}
	var out AnimeResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("jikan: decode error: %w body=%q", err, string(b[:min(len(b), 200)]))
	}
	return &out, nil
}

// GetTopAnime returns a page of top anime by popularity.
func (c *Client) GetTopAnime(ctx context.Context, page int) (*AnimeListResponse, error) {
	u := fmt.Sprintf("%s/top/anime?page=%d", c.BaseURL, page)
	return c.fetchList(ctx, u)
}

// GetSeasonNow returns anime airing in the current season.
func (c *Client) GetSeasonNow(ctx context.Context, page int) (*AnimeListResponse, error) {
	u := fmt.Sprintf("%s/seasons/now?page=%d", c.BaseURL, page)
	return c.fetchList(ctx, u)
}

// Search queries Jikan for anime by title.
func (c *Client) Search(ctx context.Context, q string, limit int) (*AnimeListResponse, error) {
	u := fmt.Sprintf("%s/anime?q=%s&limit=%d&order_by=popularity&sort=asc",
		c.BaseURL, url.QueryEscape(q), limit)
	return c.fetchList(ctx, u)
}

func (c *Client) fetchList(ctx context.Context, rawURL string) (*AnimeListResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anime-platform-ingestion/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jikan: status %d body=%q", resp.StatusCode, string(b[:min(len(b), 200)]))
	}
	var out AnimeListResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("jikan: decode error: %w body=%q", err, string(b[:min(len(b), 200)]))
	}
	return &out, nil
}
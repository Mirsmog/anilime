package animekai

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.consumet.org/anime/animekai"
	}
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type AnimeInfo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Image       string    `json:"image"`
	Description string    `json:"description"`
	Genres      []string  `json:"genres"`
	SubOrDub    string    `json:"subOrDub"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	OtherName   string    `json:"otherName"`
	Total       int32     `json:"totalEpisodes"`
	Episodes    []Episode `json:"episodes"`
}

type Episode struct {
	ID     string `json:"id"`
	Number int32  `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

func (c *Client) GetInfo(ctx context.Context, id string) (*AnimeInfo, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path += "/info"
	q := u.Query()
	q.Set("id", id)
	u.RawQuery = q.Encode()

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

	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("animekai info: status %d body=%q", resp.StatusCode, string(b[:min(len(b), 200)]))
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(ct), "application/json") {
		// Sometimes providers return HTML error pages.
		return nil, fmt.Errorf("animekai info: unexpected content-type=%q body=%q", ct, string(b[:min(len(b), 200)]))
	}

	var out AnimeInfo
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("animekai info: decode error: %w body=%q", err, string(b[:min(len(b), 200)]))
	}
	return &out, nil
}

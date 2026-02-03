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

type AnimeResponse struct {
	Data struct {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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

type ServersResponse struct {
	Status int `json:"status"`
	Data   struct {
		Sub []struct {
			ServerName string `json:"serverName"`
			ServerID   int    `json:"serverId"`
		} `json:"sub"`
		Dub []struct {
			ServerName string `json:"serverName"`
			ServerID   int    `json:"serverId"`
		} `json:"dub"`
		Raw []struct {
			ServerName string `json:"serverName"`
			ServerID   int    `json:"serverId"`
		} `json:"raw"`
		EpisodeID string `json:"episodeId"`
		EpisodeNo int    `json:"episodeNo"`
	} `json:"data"`
}

type SourcesResponse struct {
	Status int `json:"status"`
	Data   struct {
		Episode struct {
			Sources []struct {
				URL    string `json:"url"`
				Type   string `json:"type"`
				IsM3U8 bool   `json:"isM3U8"`
			} `json:"sources"`
			Tracks []struct {
				Kind    string `json:"kind"`
				File    string `json:"file"`
				Label   string `json:"label"`
				Lang    string `json:"lang"`
				Default bool   `json:"default"`
			} `json:"tracks"`
			Intro struct {
				Start float32 `json:"start"`
				End   float32 `json:"end"`
			} `json:"intro"`
			Outro struct {
				Start float32 `json:"start"`
				End   float32 `json:"end"`
			} `json:"outro"`
			Headers map[string]string `json:"headers"`
		} `json:"episode"`
	} `json:"data"`
}

func (c *Client) GetServers(ctx context.Context, providerEpisodeID string) (*ServersResponse, error) {
	u, _ := url.Parse(c.BaseURL + "/hianime/episode/servers")
	q := u.Query()
	q.Set("animeEpisodeId", providerEpisodeID)
	q.Set("raw", "1")
	u.RawQuery = q.Encode()
	return doJSON[ServersResponse](ctx, c.HTTPClient, u.String())
}

func (c *Client) GetSources(ctx context.Context, providerEpisodeID, serverID, category string) (*SourcesResponse, error) {
	u, _ := url.Parse(c.BaseURL + "/hianime/episode/sources")
	q := u.Query()
	q.Set("animeEpisodeId", providerEpisodeID)
	q.Set("server", serverID)
	q.Set("category", category)
	u.RawQuery = q.Encode()
	return doJSON[SourcesResponse](ctx, c.HTTPClient, u.String())
}

func doJSON[T any](ctx context.Context, hc *http.Client, u string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anime-platform-streaming/1.0")

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

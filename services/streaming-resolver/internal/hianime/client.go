package hianime

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
		Headers map[string]string `json:"headers"`
		Tracks  []struct {
			URL  string `json:"url"`
			Lang string `json:"lang"`
		} `json:"tracks"`
		Intro struct {
			Start float32 `json:"start"`
			End   float32 `json:"end"`
		} `json:"intro"`
		Outro struct {
			Start float32 `json:"start"`
			End   float32 `json:"end"`
		} `json:"outro"`
		Sources []struct {
			URL    string `json:"url"`
			IsM3U8 bool   `json:"isM3U8"`
			Type   string `json:"type"`
		} `json:"sources"`
	} `json:"data"`
}

func (c *Client) GetServers(ctx context.Context, providerEpisodeID string) (*ServersResponse, error) {
	endpoint := c.BaseURL + "/hianime/episode/servers?animeEpisodeId=" + providerEpisodeID + "&raw=1"
	return doJSON[ServersResponse](ctx, c.HTTPClient, endpoint)
}

func (c *Client) GetSources(ctx context.Context, providerEpisodeID, serverID, category string) (*SourcesResponse, error) {
	endpoint := c.BaseURL + "/hianime/episode/sources?animeEpisodeId=" + providerEpisodeID + "&server=" + serverID + "&category=" + category
	return doJSON[SourcesResponse](ctx, c.HTTPClient, endpoint)
}

func doJSON[T any](ctx context.Context, hc *http.Client, u string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("TE", "trailers")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:146.0) Gecko/20100101 Firefox/146.0")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	reader := resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		reader = gz
	}

	b, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hianime: status %d body=%q", resp.StatusCode, string(b[:min(len(b), 200)]))
	}
	if os.Getenv("HIANIME_DEBUG") == "true" {
		fmt.Println("hianime raw", string(b[:min(len(b), 1000)]))
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

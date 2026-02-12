package hianime

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// ClientConfig holds configurable settings for the HiAnime client.
type ClientConfig struct {
	UserAgent      string
	MaxRetries     int
	RetryBaseDelay time.Duration
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Config     ClientConfig
	CB         *gobreaker.CircuitBreaker
	Log        *zap.Logger
}

// Option configures the Client.
type Option func(*Client)

func WithCircuitBreaker(cb *gobreaker.CircuitBreaker) Option {
	return func(c *Client) { c.CB = cb }
}

func WithLogger(log *zap.Logger) Option {
	return func(c *Client) { c.Log = log }
}

func New(baseURL string, cfg ClientConfig, opts ...Option) *Client {
	if baseURL == "" {
		baseURL = "https://void-roan-six.vercel.app/api/v2"
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:146.0) Gecko/20100101 Firefox/146.0"
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay = 500 * time.Millisecond
	}
	c := &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Config:     cfg,
		Log:        zap.NewNop(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
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
	return doWithBreaker[ServersResponse](ctx, c, endpoint)
}

func (c *Client) GetSources(ctx context.Context, providerEpisodeID, serverID, category string) (*SourcesResponse, error) {
	endpoint := c.BaseURL + "/hianime/episode/sources?animeEpisodeId=" + providerEpisodeID + "&server=" + serverID + "&category=" + category
	return doWithBreaker[SourcesResponse](ctx, c, endpoint)
}

func doWithBreaker[T any](ctx context.Context, c *Client, u string) (*T, error) {
	if c.CB == nil {
		return doJSONWithRetry[T](ctx, c, u)
	}
	result, err := c.CB.Execute(func() (interface{}, error) {
		return doJSONWithRetry[T](ctx, c, u)
	})
	if err != nil {
		return nil, err
	}
	return result.(*T), nil
}

func doJSONWithRetry[T any](ctx context.Context, c *Client, u string) (*T, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.Config.RetryBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			c.Log.Debug("retrying request", zap.String("url", u), zap.Int("attempt", attempt), zap.Duration("delay", delay))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		result, err := doJSON[T](ctx, c, u)
		if err == nil {
			return result, nil
		}
		lastErr = err
		c.Log.Warn("request failed", zap.String("url", u), zap.Int("attempt", attempt), zap.Error(err))
	}
	return nil, lastErr
}

func doJSON[T any](ctx context.Context, c *Client, u string) (*T, error) {
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
	req.Header.Set("User-Agent", c.Config.UserAgent)

	resp, err := c.HTTPClient.Do(req)
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

package meili

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type SearchResponse struct {
	Hits               []json.RawMessage `json:"hits"`
	EstimatedTotalHits int               `json:"estimatedTotalHits"`
}

func New(baseURL, apiKey string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *Client) EnsureIndex(ctx context.Context, index string, primaryKey string) error {
	payload := map[string]string{"uid": index, "primaryKey": primaryKey}
	b, _ := json.Marshal(payload)
	_, err := c.do(ctx, http.MethodPost, "/indexes", bytes.NewReader(b))
	if err == nil || strings.Contains(err.Error(), "index already exists") {
		return nil
	}
	return err
}

func (c *Client) UpdateSettings(ctx context.Context, index string, settings map[string]any) error {
	b, _ := json.Marshal(settings)
	_, err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/indexes/%s/settings", index), bytes.NewReader(b))
	return err
}

func (c *Client) AddDocuments(ctx context.Context, index string, docs any) error {
	b, _ := json.Marshal(docs)
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/indexes/%s/documents", index), bytes.NewReader(b))
	return err
}

func (c *Client) Search(ctx context.Context, index string, payload any) (SearchResponse, error) {
	b, _ := json.Marshal(payload)
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/indexes/%s/search", index), bytes.NewReader(b))
	if err != nil {
		return SearchResponse{}, err
	}
	var out SearchResponse
	if err := json.Unmarshal(resp, &out); err != nil {
		return SearchResponse{}, err
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("meili error: %s", string(data))
	}
	return data, nil
}

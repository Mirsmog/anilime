package admin

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

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type BackfillHandler struct {
	JikanBaseURL string
	HTTPClient   *http.Client
	JS           nats.JetStreamContext
}

type publishResult struct {
	Requested int `json:"requested"`
	Published int `json:"published"`
}

func (h BackfillHandler) Register(r chi.Router) {
	r.Post("/backfill/jikan/top", h.handleTop)
	r.Post("/backfill/jikan/season/now", h.handleSeasonNow)
}

func (h BackfillHandler) handleTop(w http.ResponseWriter, r *http.Request) {
	pages := parseIntDefault(r.URL.Query().Get("pages"), 1)
	if pages < 1 {
		pages = 1
	}
	if pages > 200 {
		pages = 200
	}
	res, err := h.enqueueJikanList(r.Context(), func(page int) string {
		return fmt.Sprintf("%s/top/anime?page=%d", strings.TrimRight(h.JikanBaseURL, "/"), page)
	}, pages)
	if err != nil {
		api.WriteError(w, http.StatusBadGateway, "BACKFILL_FAILED", err.Error(), httpserver.RequestIDFromContext(r.Context()), nil)
		return
	}
	api.WriteJSON(w, http.StatusOK, res)
}

func (h BackfillHandler) handleSeasonNow(w http.ResponseWriter, r *http.Request) {
	pages := parseIntDefault(r.URL.Query().Get("pages"), 1)
	if pages < 1 {
		pages = 1
	}
	if pages > 100 {
		pages = 100
	}
	res, err := h.enqueueJikanList(r.Context(), func(page int) string {
		return fmt.Sprintf("%s/seasons/now?page=%d", strings.TrimRight(h.JikanBaseURL, "/"), page)
	}, pages)
	if err != nil {
		api.WriteError(w, http.StatusBadGateway, "BACKFILL_FAILED", err.Error(), httpserver.RequestIDFromContext(r.Context()), nil)
		return
	}
	api.WriteJSON(w, http.StatusOK, res)
}

type jikanListResponse struct {
	Data []struct {
		MalID int `json:"mal_id"`
	} `json:"data"`
}

func (h BackfillHandler) enqueueJikanList(ctx context.Context, urlForPage func(page int) string, pages int) (publishResult, error) {
	if h.HTTPClient == nil {
		h.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}

	dedup := map[int]struct{}{}
	requested := 0
	for p := 1; p <= pages; p++ {
		u := urlForPage(p)
		ids, err := fetchMALIDs(ctx, h.HTTPClient, u)
		if err != nil {
			return publishResult{}, err
		}
		for _, id := range ids {
			requested++
			dedup[id] = struct{}{}
		}
	}

	published := 0
	for id := range dedup {
		b, _ := json.Marshal(map[string]any{"mal_id": id})
		if _, err := h.JS.Publish("ingestion.jikan.sync", b); err != nil {
			return publishResult{}, err
		}
		published++
	}
	return publishResult{Requested: requested, Published: published}, nil
}

func fetchMALIDs(ctx context.Context, hc *http.Client, rawURL string) ([]int, error) {
	_, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anime-platform-bff/1.0")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jikan list status=%d body=%q", resp.StatusCode, string(b[:min(len(b), 200)]))
	}
	var out jikanListResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(out.Data))
	for _, it := range out.Data {
		if it.MalID > 0 {
			ids = append(ids, it.MalID)
		}
	}
	return ids, nil
}

func parseIntDefault(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

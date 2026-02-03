package jobs

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

type HiAnimeTrigger struct {
	Log *zap.Logger
	Job HiAnimeSync
}

type hianimeIngestRequest struct {
	Title string `json:"title"`
}

func (t HiAnimeTrigger) Register(r chi.Router) {
	r.Post("/v1/ingest/hianime/mal/{mal_id}", func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		malStr := strings.TrimSpace(chi.URLParam(r, "mal_id"))
		malID, err := strconv.Atoi(malStr)
		if err != nil || malID <= 0 {
			api.BadRequest(w, "VALIDATION_MAL_ID", "Invalid mal_id", rid, map[string]any{"mal_id": malStr})
			return
		}
		var body hianimeIngestRequest
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body)
		title := strings.TrimSpace(body.Title)
		if title == "" {
			// Best-effort: allow empty title, but it may reduce search quality.
			title = ""
		}

		animeID, slug, eps, err := t.Job.SyncEpisodesByMALID(r.Context(), malID, title)
		if err != nil {
			api.WriteError(w, http.StatusBadGateway, "INGEST_FAILED", err.Error(), rid, nil)
			return
		}
		api.WriteJSON(w, http.StatusOK, map[string]any{"anime_id": animeID, "hianime_slug": slug, "episode_ids": eps})
	})
}

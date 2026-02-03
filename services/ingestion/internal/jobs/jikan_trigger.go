package jobs

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/services/ingestion/internal/jikan"
)

type JikanTrigger struct {
	Log     *zap.Logger
	Jikan   *jikan.Client
	Catalog catalogv1.CatalogServiceClient
}

type jikanIngestResponse struct {
	AnimeID string `json:"anime_id"`
	Title   string `json:"title"`
}

func (t JikanTrigger) Register(r chi.Router) {
	r.Post("/v1/ingest/jikan/mal/{mal_id}", func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		malStr := strings.TrimSpace(chi.URLParam(r, "mal_id"))
		malID, err := strconv.Atoi(malStr)
		if err != nil || malID <= 0 {
			api.BadRequest(w, "VALIDATION_MAL_ID", "Invalid mal_id", rid, map[string]any{"mal_id": malStr})
			return
		}

		resp, err := t.Jikan.GetAnime(r.Context(), malID)
		if err != nil {
			api.WriteError(w, http.StatusBadGateway, "JIKAN_FAILED", err.Error(), rid, nil)
			return
		}
		pb := jikan.ToCatalogProto(resp)
		up, err := t.Catalog.UpsertJikanAnime(r.Context(), &catalogv1.UpsertJikanAnimeRequest{Anime: pb})
		if err != nil {
			api.WriteError(w, http.StatusBadGateway, "CATALOG_FAILED", err.Error(), rid, nil)
			return
		}

		api.WriteJSON(w, http.StatusOK, jikanIngestResponse{AnimeID: up.GetAnimeId(), Title: jikan.BestTitle(resp)})
	})
}

// TitleByMALID helper used by other jobs.
func TitleByMALID(ctx context.Context, c *jikan.Client, malID int) (string, error) {
	resp, err := c.GetAnime(ctx, malID)
	if err != nil {
		return "", err
	}
	return jikan.BestTitle(resp), nil
}

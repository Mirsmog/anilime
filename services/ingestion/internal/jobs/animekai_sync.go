package jobs

import (
	"context"
	"fmt"
	"strings"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/services/ingestion/internal/animekai"
)

type AnimeKaiSync struct {
	AnimeKai *animekai.Client
	Catalog  catalogv1.CatalogServiceClient
}

func (j AnimeKaiSync) SyncAnime(ctx context.Context, providerAnimeID string) (string, []string, error) {
	providerAnimeID = strings.TrimSpace(providerAnimeID)
	if providerAnimeID == "" {
		return "", nil, fmt.Errorf("providerAnimeID required")
	}

	info, err := j.AnimeKai.GetInfo(ctx, providerAnimeID)
	if err != nil {
		return "", nil, err
	}
	pb := animekai.ToCatalogProto(info)
	resp, err := j.Catalog.UpsertAnimeKaiAnime(ctx, &catalogv1.UpsertAnimeKaiAnimeRequest{Anime: pb})
	if err != nil {
		return "", nil, err
	}
	return resp.GetAnimeId(), resp.GetEpisodeIds(), nil
}

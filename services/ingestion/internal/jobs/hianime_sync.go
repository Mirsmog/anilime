package jobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/services/ingestion/internal/hianime"
	"github.com/example/anime-platform/services/ingestion/internal/jikan"
)

type HiAnimeSync struct {
	HiAnime hianime.Provider
	Catalog catalogv1.CatalogServiceClient
	Jikan   jikan.Provider
}

// SyncEpisodesByMALID finds HiAnime slug by search+malId verification and upserts episodes in Catalog.
func (j HiAnimeSync) SyncEpisodesByMALID(ctx context.Context, malID int, queryTitle string) (animeID string, slug string, episodeIDs []string, err error) {
	if malID <= 0 {
		return "", "", nil, fmt.Errorf("malID required")
	}
	queryTitle = strings.TrimSpace(queryTitle)
	if queryTitle == "" {
		if j.Jikan == nil {
			return "", "", nil, fmt.Errorf("queryTitle required")
		}
		jr, err := j.Jikan.GetAnime(ctx, malID)
		if err != nil {
			return "", "", nil, err
		}
		queryTitle = jikan.BestTitle(jr)
		queryTitle = strings.TrimSpace(queryTitle)
		if queryTitle == "" {
			return "", "", nil, fmt.Errorf("queryTitle required")
		}
	}

	res, err := j.Catalog.ResolveAnimeIDByExternalID(ctx, &catalogv1.ResolveAnimeIDByExternalIDRequest{Provider: "mal", ExternalId: strconv.Itoa(malID)})
	if err != nil {
		return "", "", nil, err
	}
	animeID = res.GetAnimeId()

	search, err := j.HiAnime.Search(ctx, queryTitle, 1)
	if err != nil {
		return animeID, "", nil, err
	}
	if search.Status != 200 || len(search.Data.Animes) == 0 {
		return animeID, "", nil, fmt.Errorf("no results for %q", queryTitle)
	}

	// Scan up to 15 candidates; verify malID match via full anime fetch to avoid title collisions.
	for i, a := range search.Data.Animes {
		if i >= 15 {
			break
		}
		cand := strings.TrimSpace(a.ID)
		if cand == "" {
			continue
		}
		ai, err := j.HiAnime.GetAnime(ctx, cand)
		if err != nil {
			continue
		}
		if ai.Data.Anime.Info.MalID == malID {
			slug = cand
			break
		}
	}
	if slug == "" {
		return animeID, "", nil, fmt.Errorf("no hianime slug matched malId=%d", malID)
	}

	eps, err := j.HiAnime.GetEpisodes(ctx, slug)
	if err != nil {
		return animeID, slug, nil, err
	}

	pbEpisodes := make([]*catalogv1.HiAnimeEpisode, 0, len(eps.Data.Episodes))
	for _, e := range eps.Data.Episodes {
		id := strings.TrimSpace(e.EpisodeID)
		if id == "" {
			continue
		}
		pbEpisodes = append(pbEpisodes, &catalogv1.HiAnimeEpisode{ProviderEpisodeId: id, Number: e.Number, Title: strings.TrimSpace(e.Title), IsFiller: e.IsFiller})
	}

	up, err := j.Catalog.UpsertHiAnimeEpisodes(ctx, &catalogv1.UpsertHiAnimeEpisodesRequest{AnimeId: animeID, HianimeSlug: slug, Episodes: pbEpisodes})
	if err != nil {
		return animeID, slug, nil, err
	}
	return animeID, slug, up.GetEpisodeIds(), nil
}

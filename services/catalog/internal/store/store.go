package store

import (
	"context"
	"time"
)

// Anime is the internal catalog representation of an anime title.
type Anime struct {
	ID            string
	Title         string
	TitleEnglish  string
	TitleJapanese string
	URL           string
	Image         string
	Description   string
	Genres        []string
	SubOrDub      string
	Type          string
	Status        string
	OtherName     string
	TotalEpisodes int32
	Score         float32
}

// Episode is the internal catalog representation of a single episode.
type Episode struct {
	ID      string
	AnimeID string
	Number  int32
	Title   string
	URL     string
	AiredAt *time.Time
}

// EpisodeInput carries provider-sourced episode data for upsert operations.
type EpisodeInput struct {
	ProviderEpisodeID string
	Number            int32
	Title             string
	URL               string
	IsFiller          bool
	HasIsFiller       bool
}

// JikanAnimeInput carries MAL/Jikan-sourced anime data.
type JikanAnimeInput struct {
	MalID         int32
	Title         string
	TitleEnglish  string
	TitleJapanese string
	Image         string
	Synopsis      string
	Genres        []string
	Type          string
	Status        string
	TotalEpisodes int32
	Score         float32
}

// CatalogStore defines all persistence operations for the catalog service.
type CatalogStore interface {
	// Anime reads
	GetAnimeByIDs(ctx context.Context, ids []string) ([]Anime, error)
	GetAllAnimeIDs(ctx context.Context) ([]string, error)
	ResolveAnimeIDByExternalID(ctx context.Context, provider, externalID string) (string, error)

	// Anime writes
	AttachExternalAnimeID(ctx context.Context, provider, externalID, animeID string) error
	UpsertJikanAnime(ctx context.Context, a JikanAnimeInput) (animeID string, err error)

	// Episode reads
	GetEpisodesByAnimeID(ctx context.Context, animeID string) ([]Episode, error)
	GetEpisodesByIDs(ctx context.Context, ids []string) ([]Episode, error)
	GetProviderEpisodeID(ctx context.Context, episodeID, provider string) (string, error)

	// Episode writes
	UpsertHiAnimeEpisodes(ctx context.Context, animeID, slug string, episodes []EpisodeInput) (episodeIDs []string, err error)
}

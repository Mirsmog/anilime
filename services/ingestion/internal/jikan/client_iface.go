package jikan

import "context"

// Provider is the port for fetching anime data from the Jikan/MAL API.
type Provider interface {
	GetAnime(ctx context.Context, malID int) (*AnimeResponse, error)
	GetTopAnime(ctx context.Context, page int) (*AnimeListResponse, error)
	GetSeasonNow(ctx context.Context, page int) (*AnimeListResponse, error)
	Search(ctx context.Context, q string, limit int) (*AnimeListResponse, error)
}

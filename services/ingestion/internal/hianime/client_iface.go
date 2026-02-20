package hianime

import "context"

// Provider is the port for fetching anime data from the HiAnime API.
type Provider interface {
	Search(ctx context.Context, q string, page int) (*SearchResponse, error)
	GetAnime(ctx context.Context, slug string) (*AnimeInfoResponse, error)
	GetEpisodes(ctx context.Context, slug string) (*EpisodesResponse, error)
}

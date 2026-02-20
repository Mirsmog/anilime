package jikan

import "context"

// Provider is the port for fetching anime data from the Jikan/MAL API.
type Provider interface {
	GetAnime(ctx context.Context, malID int) (*AnimeResponse, error)
}

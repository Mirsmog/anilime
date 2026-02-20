package animekai

import "context"

// Provider is the port for fetching anime data from the AnimeKai API.
type Provider interface {
	GetInfo(ctx context.Context, id string) (*AnimeInfo, error)
}

package hianime

import "context"

// Provider is the port for fetching playback data from the HiAnime source.
type Provider interface {
	GetServers(ctx context.Context, providerEpisodeID string) (*ServersResponse, error)
	GetSources(ctx context.Context, providerEpisodeID, serverID, category string) (*SourcesResponse, error)
}

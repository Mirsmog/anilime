package grpcapi

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	streamingv1 "github.com/example/anime-platform/gen/streaming/v1"
	"github.com/example/anime-platform/services/streaming-resolver/internal/cache"
	"github.com/example/anime-platform/services/streaming-resolver/internal/hianime"
)

type ResolverService struct {
	streamingv1.UnimplementedStreamingResolverServiceServer
	Catalog catalogv1.CatalogServiceClient
	HiAnime hianime.Provider
	Cache   *cache.RedisCache
	Log     *zap.Logger
}

type cachedPlayback struct {
	Sources           []streamingv1.PlaybackSource    `json:"sources"`
	Tracks            []streamingv1.PlaybackTrack     `json:"tracks"`
	Intro             *streamingv1.PlaybackIntroOutro `json:"intro"`
	Outro             *streamingv1.PlaybackIntroOutro `json:"outro"`
	Headers           map[string]string               `json:"headers"`
	ProviderEpisodeID string                          `json:"provider_episode_id"`
}

func (s *ResolverService) GetPlayback(ctx context.Context, req *streamingv1.GetPlaybackRequest) (*streamingv1.GetPlaybackResponse, error) {
	episodeID := strings.TrimSpace(req.GetEpisodeId())
	if episodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "episode_id required")
	}
	category := strings.ToLower(strings.TrimSpace(req.GetCategory()))
	if category == "" {
		category = "sub"
	}
	server := strings.TrimSpace(req.GetServer())
	if server == "" {
		server = "hd-1"
	}

	cacheKey := fmt.Sprintf("playback:%s:%s:%s", episodeID, category, server)
	if s.Cache != nil {
		var cached cachedPlayback
		if ok, err := s.Cache.Get(ctx, cacheKey, &cached); err == nil && ok {
			return toResponse(&cached), nil
		}
	}

	providerEpisodeID, err := s.resolveProviderEpisode(ctx, episodeID)
	if err != nil {
		return nil, err
	}
	if s.Log != nil {
		s.Log.Debug("resolved provider episode", zap.String("providerEpisodeID", providerEpisodeID))
	}
	servers, err := s.HiAnime.GetServers(ctx, providerEpisodeID)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "provider servers: %v", err)
	}
	if servers.Status != 200 {
		return nil, status.Errorf(codes.Unavailable, "provider servers: status %d", servers.Status)
	}
	server = selectServer(server, servers)
	if server == "" {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	sources, err := s.HiAnime.GetSources(ctx, providerEpisodeID, server, category)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "provider sources: %v", err)
	}
	if sources.Status != 200 {
		return nil, status.Errorf(codes.Unavailable, "provider sources: status %d", sources.Status)
	}

	out := buildCachedPlayback(providerEpisodeID, sources)

	if s.Cache != nil {
		_ = s.Cache.Set(ctx, cacheKey, out)
	}
	return toResponse(out), nil
}

func buildCachedPlayback(providerEpisodeID string, sources *hianime.SourcesResponse) *cachedPlayback {
	out := &cachedPlayback{ProviderEpisodeID: providerEpisodeID, Headers: sources.Data.Headers}
	for _, src := range sources.Data.Sources {
		out.Sources = append(out.Sources, streamingv1.PlaybackSource{Url: src.URL, Quality: src.Type, IsM3U8: src.IsM3U8})
	}
	for _, tr := range sources.Data.Tracks {
		out.Tracks = append(out.Tracks, streamingv1.PlaybackTrack{Kind: "thumbnails", File: tr.URL, Label: tr.Lang, Language: tr.Lang, IsDefault: false})
	}
	if sources.Data.Intro.End > 0 {
		out.Intro = &streamingv1.PlaybackIntroOutro{Start: sources.Data.Intro.Start, End: sources.Data.Intro.End}
	}
	if sources.Data.Outro.End > 0 {
		out.Outro = &streamingv1.PlaybackIntroOutro{Start: sources.Data.Outro.Start, End: sources.Data.Outro.End}
	}
	return out
}

func (s *ResolverService) resolveProviderEpisode(ctx context.Context, episodeID string) (string, error) {
	res, err := s.Catalog.GetProviderEpisodeID(ctx, &catalogv1.GetProviderEpisodeIDRequest{EpisodeId: episodeID, Provider: "hianime"})
	if err != nil {
		return "", status.Error(codes.NotFound, "provider episode not found")
	}
	if res.GetProviderEpisodeId() == "" {
		return "", status.Error(codes.NotFound, "provider episode not found")
	}
	return res.GetProviderEpisodeId(), nil
}

func toResponse(c *cachedPlayback) *streamingv1.GetPlaybackResponse {
	resp := &streamingv1.GetPlaybackResponse{
		ProviderEpisodeId: c.ProviderEpisodeID,
		Headers:           &streamingv1.PlaybackHeaders{Headers: c.Headers},
	}
	for i := range c.Sources {
		resp.Sources = append(resp.Sources, &c.Sources[i])
	}
	for i := range c.Tracks {
		resp.Tracks = append(resp.Tracks, &c.Tracks[i])
	}
	resp.Intro = c.Intro
	resp.Outro = c.Outro
	return resp
}

func selectServer(requested string, servers *hianime.ServersResponse) string {
	if requested = strings.TrimSpace(strings.ToLower(requested)); requested != "" {
		return requested
	}
	switch {
	case len(servers.Data.Sub) > 0:
		return servers.Data.Sub[0].ServerName
	case len(servers.Data.Dub) > 0:
		return servers.Data.Dub[0].ServerName
	case len(servers.Data.Raw) > 0:
		return servers.Data.Raw[0].ServerName
	}
	return ""
}

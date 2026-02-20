package grpcapi

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/services/catalog/internal/store"
)

type CatalogService struct {
	catalogv1.UnimplementedCatalogServiceServer
	Store store.CatalogStore
}

func (s *CatalogService) GetEpisodesByAnimeID(ctx context.Context, req *catalogv1.GetEpisodesByAnimeIDRequest) (*catalogv1.GetEpisodesByAnimeIDResponse, error) {
	animeID := strings.TrimSpace(req.GetAnimeId())
	if animeID == "" {
		return nil, status.Error(codes.InvalidArgument, "anime_id is required")
	}
	eps, err := s.Store.GetEpisodesByAnimeID(ctx, animeID)
	if err != nil {
		return nil, err
	}
	return &catalogv1.GetEpisodesByAnimeIDResponse{Episodes: episodesToProto(eps)}, nil
}

func (s *CatalogService) GetEpisodesByIDs(ctx context.Context, req *catalogv1.GetEpisodesByIDsRequest) (*catalogv1.GetEpisodesByIDsResponse, error) {
	eps, err := s.Store.GetEpisodesByIDs(ctx, req.GetEpisodeIds())
	if err != nil {
		return nil, err
	}
	return &catalogv1.GetEpisodesByIDsResponse{Episodes: episodesToProto(eps)}, nil
}

func (s *CatalogService) GetProviderEpisodeID(ctx context.Context, req *catalogv1.GetProviderEpisodeIDRequest) (*catalogv1.GetProviderEpisodeIDResponse, error) {
	epID := strings.TrimSpace(req.GetEpisodeId())
	provider := strings.TrimSpace(req.GetProvider())
	if epID == "" || provider == "" {
		return nil, status.Error(codes.InvalidArgument, "episode_id and provider required")
	}
	id, err := s.Store.GetProviderEpisodeID(ctx, epID, provider)
	if err != nil {
		return nil, err
	}
	return &catalogv1.GetProviderEpisodeIDResponse{ProviderEpisodeId: id}, nil
}

func (s *CatalogService) GetAnimeIDs(ctx context.Context, _ *catalogv1.GetAnimeIDsRequest) (*catalogv1.GetAnimeIDsResponse, error) {
	ids, err := s.Store.GetAllAnimeIDs(ctx)
	if err != nil {
		return nil, err
	}
	return &catalogv1.GetAnimeIDsResponse{AnimeIds: ids}, nil
}

func (s *CatalogService) GetAnimeByIDs(ctx context.Context, req *catalogv1.GetAnimeByIDsRequest) (*catalogv1.GetAnimeByIDsResponse, error) {
	animes, err := s.Store.GetAnimeByIDs(ctx, req.GetAnimeIds())
	if err != nil {
		return nil, err
	}
	resp := &catalogv1.GetAnimeByIDsResponse{}
	for _, a := range animes {
		resp.Anime = append(resp.Anime, &catalogv1.Anime{
			Id:            a.ID,
			Title:         a.Title,
			TitleEnglish:  a.TitleEnglish,
			TitleJapanese: a.TitleJapanese,
			Image:         a.Image,
			Description:   a.Description,
			Genres:        a.Genres,
			Score:         a.Score,
			Status:        a.Status,
			Type:          a.Type,
			TotalEpisodes: a.TotalEpisodes,
		})
	}
	return resp, nil
}

func (s *CatalogService) AttachExternalAnimeID(ctx context.Context, req *catalogv1.AttachExternalAnimeIDRequest) (*catalogv1.AttachExternalAnimeIDResponse, error) {
	animeID := strings.TrimSpace(req.GetAnimeId())
	provider := strings.TrimSpace(req.GetProvider())
	externalID := strings.TrimSpace(req.GetExternalId())
	if animeID == "" || provider == "" || externalID == "" {
		return nil, status.Error(codes.InvalidArgument, "anime_id, provider and external_id are required")
	}
	if err := s.Store.AttachExternalAnimeID(ctx, provider, externalID, animeID); err != nil {
		return nil, err
	}
	return &catalogv1.AttachExternalAnimeIDResponse{}, nil
}

func (s *CatalogService) ResolveAnimeIDByExternalID(ctx context.Context, req *catalogv1.ResolveAnimeIDByExternalIDRequest) (*catalogv1.ResolveAnimeIDByExternalIDResponse, error) {
	provider := strings.TrimSpace(req.GetProvider())
	externalID := strings.TrimSpace(req.GetExternalId())
	if provider == "" || externalID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider and external_id are required")
	}
	id, err := s.Store.ResolveAnimeIDByExternalID(ctx, provider, externalID)
	if err != nil {
		return nil, err
	}
	return &catalogv1.ResolveAnimeIDByExternalIDResponse{AnimeId: id}, nil
}

func (s *CatalogService) UpsertHiAnimeEpisodes(ctx context.Context, req *catalogv1.UpsertHiAnimeEpisodesRequest) (*catalogv1.UpsertHiAnimeEpisodesResponse, error) {
	animeID := strings.TrimSpace(req.GetAnimeId())
	slug := strings.TrimSpace(req.GetHianimeSlug())
	if animeID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid anime_id")
	}
	if slug == "" {
		return nil, status.Error(codes.InvalidArgument, "hianime_slug is required")
	}

	episodes := make([]store.EpisodeInput, 0, len(req.GetEpisodes()))
	for _, ep := range req.GetEpisodes() {
		if ep == nil {
			continue
		}
		episodes = append(episodes, store.EpisodeInput{
			ProviderEpisodeID: strings.TrimSpace(ep.GetProviderEpisodeId()),
			Number:            ep.GetNumber(),
			Title:             ep.GetTitle(),
			IsFiller:          ep.GetIsFiller(),
			HasIsFiller:       true,
		})
	}

	ids, err := s.Store.UpsertHiAnimeEpisodes(ctx, animeID, slug, episodes)
	if err != nil {
		return nil, err
	}
	return &catalogv1.UpsertHiAnimeEpisodesResponse{EpisodeIds: ids}, nil
}

func (s *CatalogService) UpsertJikanAnime(ctx context.Context, req *catalogv1.UpsertJikanAnimeRequest) (*catalogv1.UpsertJikanAnimeResponse, error) {
	anime := req.GetAnime()
	if anime == nil {
		return nil, status.Error(codes.InvalidArgument, "anime is required")
	}
	if anime.GetMalId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "mal_id is required")
	}

	animeID, err := s.Store.UpsertJikanAnime(ctx, store.JikanAnimeInput{
		MalID:         anime.GetMalId(),
		Title:         anime.GetTitle(),
		TitleEnglish:  anime.GetTitleEnglish(),
		TitleJapanese: anime.GetTitleJapanese(),
		Image:         anime.GetImage(),
		Synopsis:      anime.GetSynopsis(),
		Genres:        anime.GetGenres(),
		Type:          anime.GetType(),
		Status:        anime.GetStatus(),
		TotalEpisodes: anime.GetEpisodes(),
		Score:         anime.GetScore(),
	})
	if err != nil {
		return nil, err
	}
	return &catalogv1.UpsertJikanAnimeResponse{AnimeId: animeID}, nil
}

func (s *CatalogService) UpsertAnimeKaiAnime(ctx context.Context, req *catalogv1.UpsertAnimeKaiAnimeRequest) (*catalogv1.UpsertAnimeKaiAnimeResponse, error) {
	anime := req.GetAnime()
	if anime == nil {
		return nil, status.Error(codes.InvalidArgument, "anime is required")
	}
	provAnimeID := strings.TrimSpace(anime.GetProviderAnimeId())
	if provAnimeID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_anime_id is required")
	}

	episodes := make([]store.EpisodeInput, 0, len(anime.GetEpisodes()))
	for _, ep := range anime.GetEpisodes() {
		if ep == nil {
			continue
		}
		episodes = append(episodes, store.EpisodeInput{
			ProviderEpisodeID: strings.TrimSpace(ep.GetProviderEpisodeId()),
			Number:            ep.GetNumber(),
			Title:             ep.GetTitle(),
			URL:               ep.GetUrl(),
		})
	}

	animeID, epIDs, err := s.Store.UpsertAnimeKaiAnime(ctx, store.AnimeKaiAnimeInput{
		ProviderAnimeID: provAnimeID,
		Title:           anime.GetTitle(),
		URL:             anime.GetUrl(),
		Image:           anime.GetImage(),
		Description:     anime.GetDescription(),
		Genres:          anime.GetGenres(),
		SubOrDub:        anime.GetSubOrDub(),
		Type:            anime.GetType(),
		Status:          anime.GetStatus(),
		OtherName:       anime.GetOtherName(),
		TotalEpisodes:   anime.GetTotalEpisodes(),
		Episodes:        episodes,
	})
	if err != nil {
		return nil, err
	}
	return &catalogv1.UpsertAnimeKaiAnimeResponse{AnimeId: animeID, EpisodeIds: epIDs}, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func episodesToProto(eps []store.Episode) []*catalogv1.Episode {
	out := make([]*catalogv1.Episode, 0, len(eps))
	for _, ep := range eps {
		pb := &catalogv1.Episode{Id: ep.ID, AnimeId: ep.AnimeID, Number: ep.Number, Title: ep.Title}
		if ep.AiredAt != nil {
			pb.AiredAtRfc3339 = ep.AiredAt.UTC().Format(time.RFC3339)
		}
		out = append(out, pb)
	}
	return out
}

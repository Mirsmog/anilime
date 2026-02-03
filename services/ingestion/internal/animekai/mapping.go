package animekai

import (
	"strings"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

func ToCatalogProto(info *AnimeInfo) *catalogv1.AnimeKaiAnime {
	if info == nil {
		return nil
	}
	pb := &catalogv1.AnimeKaiAnime{
		ProviderAnimeId: strings.TrimSpace(info.ID),
		Title:           strings.TrimSpace(info.Title),
		Url:             strings.TrimSpace(info.URL),
		Image:           strings.TrimSpace(info.Image),
		Description:     strings.TrimSpace(info.Description),
		Genres:          info.Genres,
		SubOrDub:        strings.TrimSpace(info.SubOrDub),
		Type:            strings.TrimSpace(info.Type),
		Status:          strings.TrimSpace(info.Status),
		OtherName:       strings.TrimSpace(info.OtherName),
		TotalEpisodes:   info.Total,
	}
	if len(info.Episodes) > 0 {
		pb.Episodes = make([]*catalogv1.AnimeKaiEpisode, 0, len(info.Episodes))
		for _, ep := range info.Episodes {
			id := strings.TrimSpace(ep.ID)
			if id == "" {
				continue
			}
			pb.Episodes = append(pb.Episodes, &catalogv1.AnimeKaiEpisode{
				ProviderEpisodeId: id,
				Number:            ep.Number,
				Title:             strings.TrimSpace(ep.Title),
				Url:               strings.TrimSpace(ep.URL),
			})
		}
	}
	return pb
}

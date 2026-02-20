package jikan

import (
	"strings"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

func ToCatalogProto(resp *AnimeResponse) *catalogv1.JikanAnime {
	if resp == nil {
		return nil
	}
	return AnimeDataToProto(resp.Data)
}

// AnimeDataToProto converts a single AnimeData (from list or single endpoints) to proto.
func AnimeDataToProto(data AnimeData) *catalogv1.JikanAnime {
	genres := make([]string, 0, len(data.Genres))
	for _, g := range data.Genres {
		name := strings.TrimSpace(g.Name)
		if name != "" {
			genres = append(genres, name)
		}
	}

	return &catalogv1.JikanAnime{
		MalId:         data.MalID,
		Title:         strings.TrimSpace(data.Title),
		TitleEnglish:  strings.TrimSpace(data.TitleEnglish),
		TitleJapanese: strings.TrimSpace(data.TitleJapanese),
		Synopsis:      strings.TrimSpace(data.Synopsis),
		Genres:        genres,
		Status:        strings.TrimSpace(data.Status),
		Type:          strings.TrimSpace(data.Type),
		Episodes:      data.Episodes,
		Image:         strings.TrimSpace(data.Images.JPG.LargeImageURL),
		Score:         data.Score,
	}
}

func BestTitle(resp *AnimeResponse) string {
	if resp == nil {
		return ""
	}
	if t := strings.TrimSpace(resp.Data.TitleEnglish); t != "" {
		return t
	}
	if t := strings.TrimSpace(resp.Data.Title); t != "" {
		return t
	}
	return strings.TrimSpace(resp.Data.TitleJapanese)
}

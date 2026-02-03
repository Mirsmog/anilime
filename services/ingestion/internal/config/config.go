package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	CatalogGRPCAddr string
	AnimeKaiBaseURL string
	HiAnimeBaseURL  string
	JikanBaseURL    string
}

func Load() (Config, error) {
	addr := strings.TrimSpace(os.Getenv("CATALOG_GRPC_ADDR"))
	if addr == "" {
		return Config{}, errors.New("CATALOG_GRPC_ADDR is required")
	}
	base := strings.TrimSpace(os.Getenv("ANIMEKAI_BASE_URL"))
	if base == "" {
		base = "https://api.consumet.org/anime/animekai"
	}
	hia := strings.TrimSpace(os.Getenv("HIANIME_BASE_URL"))
	if hia == "" {
		hia = "https://void-roan-six.vercel.app/api/v2"
	}
	jikanURL := strings.TrimSpace(os.Getenv("JIKAN_BASE_URL"))
	if jikanURL == "" {
		jikanURL = "https://api.jikan.moe/v4"
	}
	return Config{CatalogGRPCAddr: addr, AnimeKaiBaseURL: base, HiAnimeBaseURL: hia, JikanBaseURL: jikanURL}, nil
}

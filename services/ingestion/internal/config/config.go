package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	CatalogGRPCAddr string
	AnimeKaiBaseURL string
	HiAnimeBaseURL  string
	JikanBaseURL    string
	NATSURL         string
	JikanRPS        int
	HiAnimeRPS      int
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

	natsURL := strings.TrimSpace(os.Getenv("NATS_URL"))
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	jikanRPS := 1
	if v := strings.TrimSpace(os.Getenv("JIKAN_RPS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			jikanRPS = n
		}
	}
	hiaRPS := 1
	if v := strings.TrimSpace(os.Getenv("HIANIME_RPS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hiaRPS = n
		}
	}

	return Config{CatalogGRPCAddr: addr, AnimeKaiBaseURL: base, HiAnimeBaseURL: hia, JikanBaseURL: jikanURL, NATSURL: natsURL, JikanRPS: jikanRPS, HiAnimeRPS: hiaRPS}, nil
}

package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServiceName     string
	LogLevel        string
	GRPCAddr        string
	CatalogGRPCAddr string
	NATSURL         string
	MeiliURL        string
	MeiliAPIKey     string
	ReindexInterval time.Duration
	ReindexOnce     bool
}

func Load() (Config, error) {
	serviceName := strings.TrimSpace(os.Getenv("SERVICE_NAME"))
	if serviceName == "" {
		return Config{}, errors.New("SERVICE_NAME is required")
	}
	logLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "info"
	}
	grpcAddr := strings.TrimSpace(os.Getenv("GRPC_ADDR"))
	if grpcAddr == "" {
		grpcAddr = ":9094"
	}
	catalogAddr := strings.TrimSpace(os.Getenv("CATALOG_GRPC_ADDR"))
	if catalogAddr == "" {
		return Config{}, errors.New("CATALOG_GRPC_ADDR is required")
	}
	meiliURL := strings.TrimSpace(os.Getenv("MEILI_URL"))
	if meiliURL == "" {
		meiliURL = "http://meilisearch:7700"
	}
	meiliKey := strings.TrimSpace(os.Getenv("MEILI_API_KEY"))
	natsURL := strings.TrimSpace(os.Getenv("NATS_URL"))
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	interval := 24 * time.Hour
	if v := strings.TrimSpace(os.Getenv("REINDEX_INTERVAL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	reindexOnce := false
	if v := strings.TrimSpace(os.Getenv("REINDEX_ONCE")); v == "true" || v == "1" {
		reindexOnce = true
	}

	return Config{
		ServiceName:     serviceName,
		LogLevel:        logLevel,
		GRPCAddr:        grpcAddr,
		CatalogGRPCAddr: catalogAddr,
		NATSURL:         natsURL,
		MeiliURL:        meiliURL,
		MeiliAPIKey:     meiliKey,
		ReindexInterval: interval,
		ReindexOnce:     reindexOnce,
	}, nil
}

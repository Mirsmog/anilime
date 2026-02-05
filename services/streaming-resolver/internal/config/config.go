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
	RedisURL        string
	HiAnimeBaseURL  string
	CacheTTL        time.Duration
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
		grpcAddr = ":9095"
	}
	catalogAddr := strings.TrimSpace(os.Getenv("CATALOG_GRPC_ADDR"))
	if catalogAddr == "" {
		return Config{}, errors.New("CATALOG_GRPC_ADDR is required")
	}
	redisURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if redisURL == "" {
		redisURL = "redis://redis:6379/0"
	}
	baseURL := strings.TrimSpace(os.Getenv("HIANIME_BASE_URL"))
	cacheTTL := 20 * time.Minute
	if v := strings.TrimSpace(os.Getenv("CACHE_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cacheTTL = d
		}
	}
	return Config{
		ServiceName:     serviceName,
		LogLevel:        logLevel,
		GRPCAddr:        grpcAddr,
		CatalogGRPCAddr: catalogAddr,
		RedisURL:        redisURL,
		HiAnimeBaseURL:  baseURL,
		CacheTTL:        cacheTTL,
	}, nil
}

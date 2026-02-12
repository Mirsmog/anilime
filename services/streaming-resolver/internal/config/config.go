package config

import (
	"errors"
	"os"
	"strconv"
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
	// HTTP client headers for upstream requests (configurable via env).
	HiAnimeUserAgent string
	// Retry and circuit-breaker settings.
	MaxRetries        int
	RetryBaseDelay    time.Duration
	CBMaxRequests     uint32
	CBInterval        time.Duration
	CBTimeout         time.Duration
	CBFailureThreshold uint32
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
	userAgent := strings.TrimSpace(os.Getenv("HIANIME_USER_AGENT"))
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:146.0) Gecko/20100101 Firefox/146.0"
	}
	maxRetries := envInt("HIANIME_MAX_RETRIES", 3)
	retryBaseDelay := envDuration("HIANIME_RETRY_BASE_DELAY", 500*time.Millisecond)
	cbMaxRequests := uint32(envInt("CB_MAX_REQUESTS", 5))
	cbInterval := envDuration("CB_INTERVAL", 60*time.Second)
	cbTimeout := envDuration("CB_TIMEOUT", 30*time.Second)
	cbFailureThreshold := uint32(envInt("CB_FAILURE_THRESHOLD", 5))

	return Config{
		ServiceName:        serviceName,
		LogLevel:           logLevel,
		GRPCAddr:           grpcAddr,
		CatalogGRPCAddr:    catalogAddr,
		RedisURL:           redisURL,
		HiAnimeBaseURL:     baseURL,
		CacheTTL:           cacheTTL,
		HiAnimeUserAgent:   userAgent,
		MaxRetries:         maxRetries,
		RetryBaseDelay:     retryBaseDelay,
		CBMaxRequests:      cbMaxRequests,
		CBInterval:         cbInterval,
		CBTimeout:          cbTimeout,
		CBFailureThreshold: cbFailureThreshold,
	}, nil
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

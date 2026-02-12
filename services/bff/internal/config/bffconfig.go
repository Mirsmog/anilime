package config

import (
	"errors"
	"os"
	"strings"
)

type BFFConfig struct {
	JWTSecret             []byte
	AuthGRPCAddr          string
	CatalogGRPCAddr       string
	ActivityGRPCAddr      string
	SearchGRPCAddr        string
	StreamingGRPCAddr     string
	SocialGRPCAddr        string
	HLSProxyBaseURL       string
	HLSProxySigningSecret string
	NATSURL               string
	JikanBaseURL          string
}

func LoadBFF() (BFFConfig, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return BFFConfig{}, errors.New("JWT_SECRET is required")
	}
	authAddr := strings.TrimSpace(os.Getenv("AUTH_GRPC_ADDR"))
	if authAddr == "" {
		return BFFConfig{}, errors.New("AUTH_GRPC_ADDR is required")
	}
	catalogAddr := strings.TrimSpace(os.Getenv("CATALOG_GRPC_ADDR"))
	if catalogAddr == "" {
		return BFFConfig{}, errors.New("CATALOG_GRPC_ADDR is required")
	}
	activityAddr := strings.TrimSpace(os.Getenv("ACTIVITY_GRPC_ADDR"))
	if activityAddr == "" {
		return BFFConfig{}, errors.New("ACTIVITY_GRPC_ADDR is required")
	}
	searchAddr := strings.TrimSpace(os.Getenv("SEARCH_GRPC_ADDR"))
	if searchAddr == "" {
		return BFFConfig{}, errors.New("SEARCH_GRPC_ADDR is required")
	}
	streamingAddr := strings.TrimSpace(os.Getenv("STREAMING_GRPC_ADDR"))
	if streamingAddr == "" {
		return BFFConfig{}, errors.New("STREAMING_GRPC_ADDR is required")
	}
	socialAddr := strings.TrimSpace(os.Getenv("SOCIAL_GRPC_ADDR"))
	if socialAddr == "" {
		return BFFConfig{}, errors.New("SOCIAL_GRPC_ADDR is required")
	}
	hlsBase := strings.TrimSpace(os.Getenv("HLS_PROXY_BASE_URL"))
	if hlsBase == "" {
		return BFFConfig{}, errors.New("HLS_PROXY_BASE_URL is required")
	}
	hlsSecret := strings.TrimSpace(os.Getenv("HLS_SIGNING_SECRET"))
	if hlsSecret == "" {
		return BFFConfig{}, errors.New("HLS_SIGNING_SECRET is required")
	}

	natsURL := strings.TrimSpace(os.Getenv("NATS_URL"))
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	jikanURL := strings.TrimSpace(os.Getenv("JIKAN_BASE_URL"))
	if jikanURL == "" {
		jikanURL = "https://api.jikan.moe/v4"
	}

	return BFFConfig{JWTSecret: []byte(secret), AuthGRPCAddr: authAddr, CatalogGRPCAddr: catalogAddr, ActivityGRPCAddr: activityAddr, SearchGRPCAddr: searchAddr, StreamingGRPCAddr: streamingAddr, SocialGRPCAddr: socialAddr, HLSProxyBaseURL: hlsBase, HLSProxySigningSecret: hlsSecret, NATSURL: natsURL, JikanBaseURL: jikanURL}, nil
}

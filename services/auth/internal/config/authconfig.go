package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type AuthConfig struct {
	JWTSecret              []byte
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	BootstrapAdminUsername string
}

func LoadAuth() (AuthConfig, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return AuthConfig{}, errors.New("JWT_SECRET is required")
	}

	accessTTL := parseDurationWithDefault(os.Getenv("ACCESS_TOKEN_TTL"), 15*time.Minute)
	refreshTTL := parseDurationWithDefault(os.Getenv("REFRESH_TOKEN_TTL"), 30*24*time.Hour)

	bootstrap := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_USERNAME"))
	return AuthConfig{JWTSecret: []byte(secret), AccessTokenTTL: accessTTL, RefreshTokenTTL: refreshTTL, BootstrapAdminUsername: bootstrap}, nil
}

func parseDurationWithDefault(v string, def time.Duration) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

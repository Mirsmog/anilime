package config

import (
	"errors"
	"os"
	"strings"
)

type BFFConfig struct {
	JWTSecret    []byte
	AuthGRPCAddr string
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
	return BFFConfig{JWTSecret: []byte(secret), AuthGRPCAddr: authAddr}, nil
}

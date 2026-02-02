package config

import (
	"errors"
	"os"
	"strings"
)

type HTTPConfig struct {
	Addr string
}

type AppConfig struct {
	ServiceName string
	LogLevel    string
	HTTP        HTTPConfig
}

func Load() (AppConfig, error) {
	cfg := AppConfig{
		ServiceName: strings.TrimSpace(os.Getenv("SERVICE_NAME")),
		LogLevel:    strings.TrimSpace(os.Getenv("LOG_LEVEL")),
		HTTP: HTTPConfig{
			Addr: strings.TrimSpace(os.Getenv("HTTP_ADDR")),
		},
	}
	if cfg.ServiceName == "" {
		return AppConfig{}, errors.New("SERVICE_NAME is required")
	}
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	return cfg, nil
}

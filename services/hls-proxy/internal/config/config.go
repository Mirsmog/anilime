package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	ServiceName   string
	LogLevel      string
	HTTPAddr      string
	SigningSecret string
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
	httpAddr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = ":8084"
	}
	secret := strings.TrimSpace(os.Getenv("HLS_SIGNING_SECRET"))
	if secret == "" {
		return Config{}, errors.New("HLS_SIGNING_SECRET is required")
	}
	return Config{ServiceName: serviceName, LogLevel: logLevel, HTTPAddr: httpAddr, SigningSecret: secret}, nil
}

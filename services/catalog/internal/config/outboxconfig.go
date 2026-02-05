package config

import (
	"os"
	"strings"
)

type OutboxConfig struct {
	NATSURL string
}

func LoadOutbox() OutboxConfig {
	url := strings.TrimSpace(os.Getenv("NATS_URL"))
	if url == "" {
		url = "nats://nats:4222"
	}
	return OutboxConfig{NATSURL: url}
}

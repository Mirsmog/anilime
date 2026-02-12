package config

import (
	"os"
	"strings"
)

type Config struct {
	// StripeWebhookSecret is the Stripe webhook signing secret (whsec_...).
	// Required for signature verification. Set via STRIPE_WEBHOOK_SECRET env var.
	StripeWebhookSecret string
	// NATSURL is the NATS server URL for event publishing.
	NATSURL string
}

func Load() Config {
	natsURL := strings.TrimSpace(os.Getenv("NATS_URL"))
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	return Config{
		StripeWebhookSecret: strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		NATSURL:             natsURL,
	}
}

package handlers

import (
	"os"
	"strings"

	"github.com/nats-io/nats.go"
)

// JS is the JetStream context used by handlers to publish events.
var JS nats.JetStreamContext

// AsyncWrites controls whether handlers should perform async publishes to JetStream.
// Controlled via BFF_ASYNC_WRITES env var (default true).
var AsyncWrites = true

func SetJetStream(js nats.JetStreamContext) {
	JS = js
	v := strings.TrimSpace(os.Getenv("BFF_ASYNC_WRITES"))
	if v == "" {
		AsyncWrites = true
		return
	}
	v = strings.ToLower(v)
	if v == "0" || v == "false" || v == "no" {
		AsyncWrites = false
	} else {
		AsyncWrites = true
	}
}

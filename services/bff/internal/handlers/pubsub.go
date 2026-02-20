package handlers

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

var ErrAsyncPublishDisabled = errors.New("async publish is disabled")

type EventPublisher struct {
	js          nats.JetStreamContext
	asyncWrites bool
}

func NewEventPublisher(js nats.JetStreamContext) *EventPublisher {
	return &EventPublisher{
		js:          js,
		asyncWrites: readAsyncWritesFromEnv(),
	}
}

func readAsyncWritesFromEnv() bool {
	v := strings.TrimSpace(os.Getenv("BFF_ASYNC_WRITES"))
	if v == "" {
		return true
	}
	v = strings.ToLower(v)
	return v != "0" && v != "false" && v != "no"
}

func (p *EventPublisher) Enabled() bool {
	return p != nil && p.js != nil && p.asyncWrites
}

func (p *EventPublisher) PublishJSON(subject string, payload map[string]any) (string, error) {
	if !p.Enabled() {
		return "", ErrAsyncPublishDisabled
	}

	eventID := uuid.NewString()
	payload["event_id"] = eventID
	if _, ok := payload["created_at"]; !ok {
		payload["created_at"] = time.Now().UTC().Format(time.RFC3339)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if _, err := p.js.Publish(subject, body); err != nil {
		return "", err
	}
	return eventID, nil
}

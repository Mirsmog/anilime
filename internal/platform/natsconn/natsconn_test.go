package natsconn

import (
	"testing"
	"time"
)

func TestEnvInt_Default(t *testing.T) {
	v := envInt("NATSCONN_TEST_NONEXISTENT", 42)
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestEnvInt_Set(t *testing.T) {
	t.Setenv("NATSCONN_TEST_INT", "7")
	v := envInt("NATSCONN_TEST_INT", 42)
	if v != 7 {
		t.Fatalf("expected 7, got %d", v)
	}
}

func TestEnvDuration_Default(t *testing.T) {
	v := envDuration("NATSCONN_TEST_NONEXISTENT", 5*time.Second)
	if v != 5*time.Second {
		t.Fatalf("expected 5s, got %s", v)
	}
}

func TestEnvDuration_Set(t *testing.T) {
	t.Setenv("NATSCONN_TEST_DUR", "3s")
	v := envDuration("NATSCONN_TEST_DUR", 5*time.Second)
	if v != 3*time.Second {
		t.Fatalf("expected 3s, got %s", v)
	}
}

func TestConnect_InvalidURL(t *testing.T) {
	_, err := Connect(Options{
		URL:           "nats://127.0.0.1:19999",
		MaxReconnects: 0,
		ReconnectWait: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error connecting to invalid NATS URL")
	}
}

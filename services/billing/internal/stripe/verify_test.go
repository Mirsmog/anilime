package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func makeSignature(t *testing.T, payload []byte, secret string, ts int64) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d", ts)
	mac.Write([]byte("."))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", ts, sig)
}

const testSecret = "whsec_test_secret"

func TestConstructEvent_ValidSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed"}`)
	ts := time.Now().Unix()
	header := makeSignature(t, payload, testSecret, ts)

	err := ConstructEvent(payload, header, testSecret)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestConstructEvent_InvalidSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed"}`)
	ts := time.Now().Unix()
	header := fmt.Sprintf("t=%d,v1=invalidsignature", ts)

	err := ConstructEvent(payload, header, testSecret)
	if err != ErrNoValidSignature {
		t.Fatalf("expected ErrNoValidSignature, got: %v", err)
	}
}

func TestConstructEvent_ExpiredTimestamp(t *testing.T) {
	payload := []byte(`{"id":"evt_123"}`)
	ts := time.Now().Add(-10 * time.Minute).Unix()
	header := makeSignature(t, payload, testSecret, ts)

	err := ConstructEvent(payload, header, testSecret)
	if err != ErrTimestampExpired {
		t.Fatalf("expected ErrTimestampExpired, got: %v", err)
	}
}

func TestConstructEvent_EmptyHeader(t *testing.T) {
	err := ConstructEvent([]byte("{}"), "", "secret")
	if err != ErrInvalidHeader {
		t.Fatalf("expected ErrInvalidHeader, got: %v", err)
	}
}

func TestConstructEvent_MissingTimestamp(t *testing.T) {
	err := ConstructEvent([]byte("{}"), "v1=abc123", "secret")
	if err != ErrInvalidHeader {
		t.Fatalf("expected ErrInvalidHeader, got: %v", err)
	}
}

func TestConstructEvent_WrongSecret(t *testing.T) {
	secret := "whsec_correct"
	payload := []byte(`{"id":"evt_456"}`)
	ts := time.Now().Unix()
	header := makeSignature(t, payload, secret, ts)

	err := ConstructEvent(payload, header, "whsec_wrong")
	if err != ErrNoValidSignature {
		t.Fatalf("expected ErrNoValidSignature, got: %v", err)
	}
}

func TestConstructEvent_TamperedPayload(t *testing.T) {
	secret := "whsec_test"
	original := []byte(`{"id":"evt_123"}`)
	ts := time.Now().Unix()
	header := makeSignature(t, original, secret, ts)

	tampered := []byte(`{"id":"evt_999"}`)
	err := ConstructEvent(tampered, header, secret)
	if err != ErrNoValidSignature {
		t.Fatalf("expected ErrNoValidSignature, got: %v", err)
	}
}

func TestConstructEventWithTolerance_ZeroTolerance(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"id":"evt_123"}`)
	ts := time.Now().Add(-1 * time.Hour).Unix()
	header := makeSignature(t, payload, secret, ts)

	// Zero tolerance = skip timestamp check
	err := ConstructEventWithTolerance(payload, header, secret, 0)
	if err != nil {
		t.Fatalf("expected no error with zero tolerance, got: %v", err)
	}
}

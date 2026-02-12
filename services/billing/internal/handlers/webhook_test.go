package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/example/anime-platform/services/billing/internal/idempotency"
	stripeutil "github.com/example/anime-platform/services/billing/internal/stripe"
)

const testSecret = "whsec_test_secret"

func makeTestSignature(payload []byte, secret string) string {
	ts := time.Now().Unix()
	sig := stripeutil.ComputeSignatureForTest(ts, payload, secret)
	return fmt.Sprintf("t=%d,v1=%s", ts, sig)
}

func newTestHandler() *WebhookHandler {
	log, _ := zap.NewDevelopment()
	idem := idempotency.NewStore("", "", 0) // in-memory
	return NewWebhookHandler(testSecret, log, idem)
}

func TestWebhook_ValidSignature(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"id":   "evt_test_1",
		"type": "checkout.session.completed",
		"data": map[string]any{"object": map[string]any{}},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", makeTestSignature(body, testSecret))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhook_InvalidSignature(t *testing.T) {
	h := newTestHandler()
	body := []byte(`{"id":"evt_test_2","type":"checkout.session.completed"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", fmt.Sprintf("t=%d,v1=invalidsig", time.Now().Unix()))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWebhook_MissingSignature(t *testing.T) {
	h := newTestHandler()
	body := []byte(`{"id":"evt_test_3","type":"checkout.session.completed"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewReader(body))
	// No Stripe-Signature header
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWebhook_Idempotency_DuplicateEvent(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{
		"id":   "evt_dup",
		"type": "invoice.paid",
		"data": map[string]any{"object": map[string]any{}},
	})
	sig := makeTestSignature(body, testSecret)

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewReader(body))
	req1.Header.Set("Stripe-Signature", sig)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", w1.Code)
	}

	// Second request with same event ID (duplicate)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewReader(body))
	req2.Header.Set("Stripe-Signature", sig)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("duplicate call: expected 200, got %d", w2.Code)
	}
}

func TestWebhook_MissingEventID(t *testing.T) {
	h := newTestHandler()
	body := []byte(`{"type":"checkout.session.completed"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", makeTestSignature(body, testSecret))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/services/billing/internal/stripe"
)

const maxBodyBytes = 65536

// stripeEvent represents a minimal Stripe event payload.
type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// WebhookHandler handles Stripe webhook POST requests.
type WebhookHandler struct {
	Secret string
	Log    *zap.Logger
	// processedEvents stores event IDs for idempotency.
	// TODO: replace with persistent store (Redis/DB) for production.
	mu              sync.Mutex
	processedEvents map[string]struct{}
}

func NewWebhookHandler(secret string, log *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		Secret:          secret,
		Log:             log,
		processedEvents: make(map[string]struct{}),
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		api.BadRequest(w, "READ_ERROR", "cannot read body", "", nil)
		return
	}

	// Verify signature if secret is configured
	if h.Secret != "" {
		sigHeader := r.Header.Get("Stripe-Signature")
		if err := stripe.ConstructEvent(body, sigHeader, h.Secret); err != nil {
			h.Log.Warn("stripe signature verification failed", zap.Error(err))
			api.BadRequest(w, "INVALID_SIGNATURE", "webhook signature verification failed", "", nil)
			return
		}
	} else {
		h.Log.Warn("STRIPE_WEBHOOK_SECRET not set, skipping signature verification")
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		api.BadRequest(w, "INVALID_JSON", "cannot parse event", "", nil)
		return
	}

	if event.ID == "" {
		api.BadRequest(w, "MISSING_EVENT_ID", "event id is required", "", nil)
		return
	}

	// Idempotency check
	h.mu.Lock()
	if _, seen := h.processedEvents[event.ID]; seen {
		h.mu.Unlock()
		h.Log.Debug("duplicate event, skipping", zap.String("event_id", event.ID))
		w.WriteHeader(http.StatusOK)
		return
	}
	h.processedEvents[event.ID] = struct{}{}
	h.mu.Unlock()

	// Handle specific event types
	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(event)
	case "invoice.paid":
		h.handleInvoicePaid(event)
	default:
		h.Log.Debug("unhandled event type", zap.String("type", event.Type), zap.String("event_id", event.ID))
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleCheckoutCompleted(event stripeEvent) {
	h.Log.Info("checkout.session.completed",
		zap.String("event_id", event.ID),
	)
	// TODO: publish NATS event billing.checkout.completed
	// TODO: update user subscription status in DB
}

func (h *WebhookHandler) handleInvoicePaid(event stripeEvent) {
	h.Log.Info("invoice.paid",
		zap.String("event_id", event.ID),
	)
	// TODO: publish NATS event billing.invoice.paid
	// TODO: extend user subscription period in DB
}

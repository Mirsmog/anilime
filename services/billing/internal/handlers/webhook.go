package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/services/billing/internal/idempotency"
	"github.com/example/anime-platform/services/billing/internal/publisher"
	billingstore "github.com/example/anime-platform/services/billing/internal/store"
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
	secret     string
	log        *zap.Logger
	idempotent idempotency.Store
	store      *billingstore.BillingStore
	pub        *publisher.Publisher
}

func NewWebhookHandler(
	secret string,
	log *zap.Logger,
	idem idempotency.Store,
	st *billingstore.BillingStore,
	pub *publisher.Publisher,
) *WebhookHandler {
	return &WebhookHandler{
		secret:     secret,
		log:        log,
		idempotent: idem,
		store:      st,
		pub:        pub,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		api.BadRequest(w, "READ_ERROR", "cannot read body", "", nil)
		return
	}

	// Signature is always verified — STRIPE_WEBHOOK_SECRET is required at startup.
	sigHeader := r.Header.Get("Stripe-Signature")
	if err := stripe.ConstructEvent(body, sigHeader, h.secret); err != nil {
		h.log.Warn("stripe signature verification failed", zap.Error(err))
		api.BadRequest(w, "INVALID_SIGNATURE", "webhook signature verification failed", "", nil)
		return
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

	// Idempotency check via Redis SETNX / Postgres fallback / in-memory.
	dup, err := h.idempotent.Check(r.Context(), event.ID)
	if err != nil {
		h.log.Error("idempotency check failed", zap.Error(err))
		api.Internal(w, "")
		return
	}
	if dup {
		h.log.Debug("duplicate event, skipping", zap.String("event_id", event.ID))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle specific event types.
	switch event.Type {
	case "checkout.session.completed":
		if err := h.handleCheckoutCompleted(r.Context(), event); err != nil {
			h.log.Error("handle checkout failed", zap.Error(err))
			api.Internal(w, "")
			return
		}
	case "invoice.paid":
		if err := h.handleInvoicePaid(r.Context(), event); err != nil {
			h.log.Error("handle invoice failed", zap.Error(err))
			api.Internal(w, "")
			return
		}
	default:
		h.log.Debug("unhandled event type", zap.String("type", event.Type), zap.String("event_id", event.ID))
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleCheckoutCompleted(ctx context.Context, event stripeEvent) error {
	h.log.Info("checkout.session.completed", zap.String("event_id", event.ID))

	// Persist payment transactionally before publishing.
	if h.store.Available() {
		tx, err := h.store.BeginTx(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := h.store.SavePayment(ctx, tx, event.ID, event.Data.Object); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	// Publish NATS event after successful persistence.
	return h.pub.Publish(ctx, publisher.SubjectPaymentCompleted, publisher.BillingEvent{
		EventID:   event.ID,
		EventType: event.Type,
		Data:      event.Data.Object,
	})
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, event stripeEvent) error {
	h.log.Info("invoice.paid", zap.String("event_id", event.ID))

	// Persist subscription transactionally before publishing.
	if h.store.Available() {
		tx, err := h.store.BeginTx(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := h.store.SaveSubscription(ctx, tx, event.ID, event.Data.Object); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	// Publish NATS event after successful persistence.
	return h.pub.Publish(ctx, publisher.SubjectSubscriptionUpdated, publisher.BillingEvent{
		EventID:   event.ID,
		EventType: event.Type,
		Data:      event.Data.Object,
	})
}

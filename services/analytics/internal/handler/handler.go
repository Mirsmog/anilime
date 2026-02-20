// Package handler routes raw NATS messages to PostHog captures.
// Each function corresponds to one analytics.* subject or a re-sourced existing subject.
package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/example/anime-platform/services/analytics/internal/posthog"
)

// Dispatcher routes incoming NATS messages to the correct PostHog capture call.
type Dispatcher struct {
	ph  *posthog.Client
	log *zap.Logger
}

// New creates a Dispatcher.
func New(ph *posthog.Client, log *zap.Logger) *Dispatcher {
	return &Dispatcher{ph: ph, log: log}
}

// Dispatch routes msg to the correct handler based on its subject.
// Returns false if the subject is unknown (message should still be Ack'd to avoid replay).
func (d *Dispatcher) Dispatch(msg *nats.Msg) {
	subj := msg.Subject
	switch {
	case subj == "analytics.auth.registered":
		d.handleAuthRegistered(msg)
	case subj == "analytics.auth.logged_in":
		d.handleAuthLoggedIn(msg)
	case subj == "analytics.streaming.started":
		d.handleStreamingStarted(msg)
	case subj == "analytics.catalog.anime_viewed":
		d.handleAnimeViewed(msg)
	case subj == "analytics.search.performed":
		d.handleSearchPerformed(msg)
	case subj == "activity.progress":
		d.handleActivityProgress(msg)
	case strings.HasPrefix(subj, "social.comments."):
		d.handleSocialComment(msg)
	case subj == "billing.payment.completed":
		d.handleBillingPayment(msg)
	case subj == "billing.subscription.updated":
		d.handleBillingSubscription(msg)
	default:
		d.log.Debug("analytics: unhandled subject", zap.String("subject", subj))
	}
}

// ── auth events ──────────────────────────────────────────────────────────────

func (d *Dispatcher) handleAuthRegistered(msg *nats.Msg) {
	var ev struct {
		UserID     string    `json:"user_id"`
		Username   string    `json:"username"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	d.ph.Identify(ev.UserID, map[string]any{
		"username":   ev.Username,
		"created_at": ev.OccurredAt,
	})
	d.ph.Capture(ev.UserID, "user_registered", map[string]any{
		"username": ev.Username,
	})
}

func (d *Dispatcher) handleAuthLoggedIn(msg *nats.Msg) {
	var ev struct {
		UserID     string    `json:"user_id"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	d.ph.Capture(ev.UserID, "user_logged_in", nil)
}

// ── streaming events ─────────────────────────────────────────────────────────

func (d *Dispatcher) handleStreamingStarted(msg *nats.Msg) {
	var ev struct {
		UserID     string    `json:"user_id"`
		EpisodeID  string    `json:"episode_id"`
		AnimeID    string    `json:"anime_id"`
		Category   string    `json:"category"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	d.ph.Capture(ev.UserID, "playback_started", map[string]any{
		"episode_id": ev.EpisodeID,
		"anime_id":   ev.AnimeID,
		"category":   ev.Category,
	})
}

// ── catalog events ────────────────────────────────────────────────────────────

func (d *Dispatcher) handleAnimeViewed(msg *nats.Msg) {
	var ev struct {
		UserID     string    `json:"user_id"`
		AnimeID    string    `json:"anime_id"`
		Title      string    `json:"title"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	distinctID := ev.UserID
	if distinctID == "" {
		distinctID = "anonymous"
	}
	d.ph.Capture(distinctID, "anime_viewed", map[string]any{
		"anime_id": ev.AnimeID,
		"title":    ev.Title,
	})
}

// ── search events ─────────────────────────────────────────────────────────────

func (d *Dispatcher) handleSearchPerformed(msg *nats.Msg) {
	var ev struct {
		UserID       string    `json:"user_id"`
		Query        string    `json:"query"`
		ResultsCount int       `json:"results_count"`
		Filters      any       `json:"filters,omitempty"`
		OccurredAt   time.Time `json:"occurred_at"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	distinctID := ev.UserID
	if distinctID == "" {
		distinctID = "anonymous"
	}
	props := map[string]any{
		"query":         ev.Query,
		"results_count": ev.ResultsCount,
		"has_results":   ev.ResultsCount > 0,
	}
	if ev.Filters != nil {
		props["filters"] = ev.Filters
	}
	d.ph.Capture(distinctID, "search_performed", props)
}

// ── activity events ───────────────────────────────────────────────────────────

func (d *Dispatcher) handleActivityProgress(msg *nats.Msg) {
	var ev struct {
		UserID          string `json:"user_id"`
		EpisodeID       string `json:"episode_id"`
		AnimeID         string `json:"anime_id"`
		PositionSeconds int32  `json:"position_seconds"`
		DurationSeconds int32  `json:"duration_seconds"`
		Completed       bool   `json:"completed"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	if !ev.Completed {
		return // only track completion events to avoid spamming PostHog with every progress tick
	}
	d.ph.Capture(ev.UserID, "episode_completed", map[string]any{
		"episode_id":       ev.EpisodeID,
		"anime_id":         ev.AnimeID,
		"duration_seconds": ev.DurationSeconds,
	})
}

// ── social events ─────────────────────────────────────────────────────────────

func (d *Dispatcher) handleSocialComment(msg *nats.Msg) {
	action := strings.TrimPrefix(msg.Subject, "social.comments.")
	var ev struct {
		UserID  string `json:"user_id"`
		AnimeID string `json:"anime_id,omitempty"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	if action != "create" {
		return // only track comment creation for now
	}
	d.ph.Capture(ev.UserID, "comment_created", map[string]any{
		"anime_id": ev.AnimeID,
	})
}

// ── billing events ────────────────────────────────────────────────────────────

func (d *Dispatcher) handleBillingPayment(msg *nats.Msg) {
	var ev struct {
		EventID   string          `json:"event_id"`
		EventType string          `json:"event_type"`
		Data      json.RawMessage `json:"data"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	// Extract customer metadata from Stripe object.
	var stripeObj struct {
		CustomerDetails struct {
			Email string `json:"email"`
		} `json:"customer_details"`
		AmountTotal int64  `json:"amount_total"`
		Currency    string `json:"currency"`
		Metadata    struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(ev.Data, &stripeObj); err != nil {
		d.log.Warn("analytics: parse billing payment", zap.Error(err))
		return
	}
	distinctID := stripeObj.Metadata.UserID
	if distinctID == "" {
		distinctID = stripeObj.CustomerDetails.Email
	}
	if distinctID == "" {
		return
	}
	d.ph.Capture(distinctID, "subscription_started", map[string]any{
		"amount_cents": stripeObj.AmountTotal,
		"currency":     stripeObj.Currency,
	})
}

func (d *Dispatcher) handleBillingSubscription(msg *nats.Msg) {
	var ev struct {
		EventID string          `json:"event_id"`
		Data    json.RawMessage `json:"data"`
	}
	if !unmarshal(d.log, msg, &ev) {
		return
	}
	var stripeObj struct {
		Status   string `json:"status"`
		Metadata struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
		Plan struct {
			Amount   int64  `json:"amount"`
			Currency string `json:"currency"`
			Interval string `json:"interval"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(ev.Data, &stripeObj); err != nil {
		d.log.Warn("analytics: parse billing subscription", zap.Error(err))
		return
	}
	if stripeObj.Metadata.UserID == "" || stripeObj.Status != "active" {
		return
	}
	d.ph.Capture(stripeObj.Metadata.UserID, "subscription_renewed", map[string]any{
		"amount_cents": stripeObj.Plan.Amount,
		"currency":     stripeObj.Plan.Currency,
		"interval":     stripeObj.Plan.Interval,
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func unmarshal(log *zap.Logger, msg *nats.Msg, dst any) bool {
	if err := json.Unmarshal(msg.Data, dst); err != nil {
		log.Error("analytics: unmarshal message",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return false
	}
	return true
}

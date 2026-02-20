package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProgressRecord is the internal representation of episode watch progress.
type ProgressRecord struct {
	UserID          uuid.UUID
	EpisodeID       uuid.UUID
	PositionSeconds int
	DurationSeconds int
	Completed       bool
	ClientTsMs      int64
	UpdatedAt       time.Time
}

// ProgressCursor is the decoded form of the opaque pagination cursor.
type ProgressCursor struct {
	UpdatedAt time.Time
	EpisodeID uuid.UUID
}

// ProgressRepository defines persistence operations for watch progress.
type ProgressRepository interface {
	// Upsert inserts or updates progress, ignoring stale writes (client_ts_ms <= existing).
	// Returns the current (possibly unchanged) record.
	Upsert(ctx context.Context, r ProgressRecord) (ProgressRecord, error)
	// List returns up to limit records ordered by updated_at DESC.
	// cursor, if non-nil, acts as an exclusive lower bound for keyset pagination.
	List(ctx context.Context, userID uuid.UUID, limit int, cursor *ProgressCursor) ([]ProgressRecord, error)
}

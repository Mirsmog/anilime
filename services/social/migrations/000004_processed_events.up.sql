CREATE TABLE IF NOT EXISTS processed_events (
    event_id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    payload JSONB
);

CREATE INDEX IF NOT EXISTS idx_processed_events_created ON processed_events(created_at);

CREATE TABLE IF NOT EXISTS catalog_outbox (
  id UUID PRIMARY KEY,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS catalog_outbox_unpublished_idx ON catalog_outbox (created_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS processed_events (
  event_id TEXT PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS payments (
  id          BIGSERIAL    PRIMARY KEY,
  event_id    TEXT         NOT NULL UNIQUE,
  stripe_session_id TEXT   NOT NULL DEFAULT '',
  status      TEXT         NOT NULL DEFAULT 'completed',
  raw_data    JSONB        NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subscriptions (
  id          BIGSERIAL    PRIMARY KEY,
  event_id    TEXT         NOT NULL UNIQUE,
  stripe_invoice_id TEXT   NOT NULL DEFAULT '',
  status      TEXT         NOT NULL DEFAULT 'active',
  raw_data    JSONB        NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_episode_progress (
  user_id UUID NOT NULL,
  episode_id UUID NOT NULL,

  position_seconds INT NOT NULL DEFAULT 0,
  duration_seconds INT NOT NULL DEFAULT 0,
  completed BOOLEAN NOT NULL DEFAULT FALSE,

  client_ts_ms BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  PRIMARY KEY (user_id, episode_id)
);

CREATE INDEX IF NOT EXISTS user_episode_progress_updated_at_idx
  ON user_episode_progress (user_id, updated_at DESC);

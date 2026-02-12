CREATE TABLE IF NOT EXISTS ratings (
  user_id   TEXT        NOT NULL,
  anime_id  TEXT        NOT NULL,
  score     INT         NOT NULL CHECK (score >= 1 AND score <= 10),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, anime_id)
);

CREATE INDEX IF NOT EXISTS ratings_anime_id_idx ON ratings (anime_id);

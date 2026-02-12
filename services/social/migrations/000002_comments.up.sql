CREATE TABLE IF NOT EXISTS comments (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  anime_id   TEXT        NOT NULL,
  user_id    TEXT        NOT NULL,
  parent_id  UUID        REFERENCES comments(id),
  body       TEXT        NOT NULL,
  score      INT         NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS comments_anime_created_idx ON comments (anime_id, created_at);
CREATE INDEX IF NOT EXISTS comments_anime_parent_idx  ON comments (anime_id, parent_id);
CREATE INDEX IF NOT EXISTS comments_parent_idx        ON comments (parent_id);
CREATE INDEX IF NOT EXISTS comments_user_created_idx  ON comments (user_id, created_at);

CREATE TABLE IF NOT EXISTS comment_votes (
  comment_id UUID     NOT NULL REFERENCES comments(id),
  user_id    TEXT     NOT NULL,
  vote       SMALLINT NOT NULL CHECK (vote IN (-1, 1)),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (comment_id, user_id)
);

CREATE TABLE IF NOT EXISTS episodes (
  id UUID PRIMARY KEY,
  anime_id UUID NOT NULL,
  number INT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  aired_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS episodes_anime_id_idx ON episodes (anime_id);
CREATE UNIQUE INDEX IF NOT EXISTS episodes_anime_id_number_uidx ON episodes (anime_id, number);

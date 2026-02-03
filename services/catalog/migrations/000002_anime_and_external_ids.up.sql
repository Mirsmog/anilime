-- anime (provider-agnostic internal entity)
CREATE TABLE IF NOT EXISTS anime (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  image TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  genres JSONB NOT NULL DEFAULT '[]',
  sub_or_dub TEXT NOT NULL DEFAULT 'unknown',
  type TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  other_name TEXT NOT NULL DEFAULT '',
  total_episodes INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- map provider anime ids -> internal anime id
CREATE TABLE IF NOT EXISTS external_anime_ids (
  provider TEXT NOT NULL,
  provider_anime_id TEXT NOT NULL,
  anime_id UUID NOT NULL REFERENCES anime(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (provider, provider_anime_id)
);

CREATE INDEX IF NOT EXISTS external_anime_ids_anime_id_idx ON external_anime_ids (anime_id);

-- extend episodes with url and updated_at
ALTER TABLE episodes ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '';
ALTER TABLE episodes ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- map provider episode ids -> internal episode id
CREATE TABLE IF NOT EXISTS external_episode_ids (
  provider TEXT NOT NULL,
  provider_episode_id TEXT NOT NULL,
  episode_id UUID NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (provider, provider_episode_id)
);

CREATE INDEX IF NOT EXISTS external_episode_ids_episode_id_idx ON external_episode_ids (episode_id);

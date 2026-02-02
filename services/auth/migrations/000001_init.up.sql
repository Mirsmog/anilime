-- users
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL,
  username TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_uidx ON users (lower(email));
CREATE UNIQUE INDEX IF NOT EXISTS users_username_uidx ON users (lower(username));

-- refresh sessions (refresh token rotation)
CREATE TABLE IF NOT EXISTS refresh_sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ NULL,
  replaced_by_session_id UUID NULL,
  user_agent TEXT NULL,
  ip INET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_sessions_token_hash_uidx ON refresh_sessions (token_hash);
CREATE INDEX IF NOT EXISTS refresh_sessions_user_id_idx ON refresh_sessions (user_id);
CREATE INDEX IF NOT EXISTS refresh_sessions_expires_at_idx ON refresh_sessions (expires_at);

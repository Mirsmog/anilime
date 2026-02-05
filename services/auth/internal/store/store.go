package store

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/anime-platform/services/auth/internal/domain"
)

var (
	ErrConflict     = errors.New("conflict")
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

type Store struct {
	DB *pgxpool.Pool
}

type CreateUserParams struct {
	Email        string
	Username     string
	PasswordHash string
}

func (s Store) CreateUser(ctx context.Context, p CreateUserParams) (domain.User, error) {
	id := uuid.New()
	var u domain.User
	q := `
INSERT INTO users (id, email, username, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id::text, email, username, role, created_at;
`
	err := s.DB.QueryRow(ctx, q, id, p.Email, p.Username, p.PasswordHash).Scan(&u.ID, &u.Email, &u.Username, &u.Role, &u.CreatedAt)
	if err != nil {
		// unique violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return domain.User{}, ErrConflict
			}
		}
		return domain.User{}, err
	}
	return u, nil
}

type UserRow struct {
	User         domain.User
	PasswordHash string
}

func (s Store) FindUserByLogin(ctx context.Context, login string) (UserRow, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return UserRow{}, ErrNotFound
	}

	q := `
SELECT id::text, email, username, role, password_hash, created_at
FROM users
WHERE lower(email) = lower($1) OR lower(username) = lower($1)
LIMIT 1;
`
	var row UserRow
	err := s.DB.QueryRow(ctx, q, login).Scan(&row.User.ID, &row.User.Email, &row.User.Username, &row.User.Role, &row.PasswordHash, &row.User.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, err
	}
	return row, nil
}

type CreateRefreshSessionParams struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UserAgent string
	IP        net.IP
	Now       time.Time
}

func (s Store) CreateRefreshSession(ctx context.Context, p CreateRefreshSessionParams) error {
	q := `
INSERT INTO refresh_sessions (id, user_id, token_hash, expires_at, created_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7);
`
	_, err := s.DB.Exec(ctx, q, p.SessionID, p.UserID, p.TokenHash, p.ExpiresAt, p.Now, nullableString(p.UserAgent), nullableInet(p.IP))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrConflict
			}
		}
		return err
	}
	return nil
}

type RefreshSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (s Store) GetRefreshSessionByHash(ctx context.Context, tokenHash string) (RefreshSession, error) {
	q := `
SELECT id, user_id, token_hash, expires_at, revoked_at
FROM refresh_sessions
WHERE token_hash = $1
LIMIT 1;
`
	var rs RefreshSession
	err := s.DB.QueryRow(ctx, q, tokenHash).Scan(&rs.ID, &rs.UserID, &rs.TokenHash, &rs.ExpiresAt, &rs.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshSession{}, ErrNotFound
		}
		return RefreshSession{}, err
	}
	return rs, nil
}

func (s Store) RevokeRefreshSession(ctx context.Context, sessionID uuid.UUID, now time.Time) error {
	q := `UPDATE refresh_sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL;`
	_, err := s.DB.Exec(ctx, q, sessionID, now)
	return err
}

func (s Store) ReplaceRefreshSession(ctx context.Context, oldID, newID uuid.UUID, now time.Time) error {
	q := `UPDATE refresh_sessions SET revoked_at = $3, replaced_by_session_id = $2 WHERE id = $1 AND revoked_at IS NULL;`
	_, err := s.DB.Exec(ctx, q, oldID, newID, now)
	return err
}

func nullableString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func nullableInet(ip net.IP) any {
	if ip == nil {
		return nil
	}
	return ip
}

package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/anime-platform/services/auth/internal/domain"
)

func (s Store) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	q := `SELECT id, email, username, role, created_at FROM users WHERE id = $1::uuid LIMIT 1;`
	var u domain.User
	err := s.DB.QueryRow(ctx, q, userID).Scan(&u.ID, &u.Email, &u.Username, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}
	return u, nil
}

var _ *pgxpool.Pool

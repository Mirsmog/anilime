package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PromoteAdmin sets role=admin for a user with the given username (case-insensitive).
func PromoteAdmin(ctx context.Context, db *pgxpool.Pool, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}

	q := `UPDATE users SET role='admin' WHERE lower(username)=lower($1);`
	_, err := db.Exec(ctx, q, username)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// users table might not exist before migrations in local dev.
		if pgErr.Code == "42P01" {
			return nil
		}
	}
	return err
}

package store

import (
	"context"

	"github.com/google/uuid"
)

func (s PostgresStore) SetUserRoleByID(ctx context.Context, userID uuid.UUID, role string) error {
	q := `UPDATE users SET role=$2 WHERE id=$1;`
	_, err := s.DB.Exec(ctx, q, userID, role)
	return err
}

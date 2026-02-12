package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRatingStore persists ratings in Postgres.
type PostgresRatingStore struct {
	pool *pgxpool.Pool
}

// NewPostgresRatingStore creates a store backed by Postgres.
func NewPostgresRatingStore(pool *pgxpool.Pool) *PostgresRatingStore {
	return &PostgresRatingStore{pool: pool}
}

func (s *PostgresRatingStore) Upsert(ctx context.Context, animeID, userID string, score int) error {
	const q = `INSERT INTO ratings (user_id, anime_id, score)
	           VALUES ($1, $2, $3)
	           ON CONFLICT (user_id, anime_id) DO UPDATE SET
	             score = EXCLUDED.score,
	             updated_at = now()`
	_, err := s.pool.Exec(ctx, q, userID, animeID, score)
	return err
}

func (s *PostgresRatingStore) GetSummary(ctx context.Context, animeID string) (RatingSummary, error) {
	const q = `SELECT COALESCE(AVG(score), 0), COUNT(*)
	           FROM ratings WHERE anime_id = $1`
	var avg float64
	var total int
	if err := s.pool.QueryRow(ctx, q, animeID).Scan(&avg, &total); err != nil {
		return RatingSummary{AnimeID: animeID}, err
	}
	return RatingSummary{
		AnimeID:      animeID,
		AverageScore: avg,
		TotalRatings: total,
	}, nil
}

func (s *PostgresRatingStore) GetUserRating(ctx context.Context, animeID, userID string) (int, bool, error) {
	const q = `SELECT score FROM ratings WHERE anime_id = $1 AND user_id = $2`
	var score int
	err := s.pool.QueryRow(ctx, q, animeID, userID).Scan(&score)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return 0, false, nil
		}
		return 0, false, err
	}
	return score, true, nil
}

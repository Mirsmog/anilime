// Package store provides rating storage backends.
// In production, use PostgresRatingStore (NewPostgresRatingStore).
// For development without a database, use NewInMemoryRatingStore.
package store

import (
	"context"
	"sync"
)

type Rating struct {
	UserID  string `json:"user_id"`
	AnimeID string `json:"anime_id"`
	Score   int    `json:"score"` // 1-10
}

type RatingSummary struct {
	AnimeID      string  `json:"anime_id"`
	AverageScore float64 `json:"average_score"`
	TotalRatings int     `json:"total_ratings"`
}

// RatingStore defines the contract for rating persistence.
type RatingStore interface {
	Upsert(ctx context.Context, animeID, userID string, score int) error
	GetSummary(ctx context.Context, animeID string) (RatingSummary, error)
	GetUserRating(ctx context.Context, animeID, userID string) (int, bool, error)
}

// InMemoryRatingStore is a development-only in-memory implementation.
type InMemoryRatingStore struct {
	mu      sync.RWMutex
	ratings map[string]map[string]int // anime_id -> user_id -> score
}

func NewInMemoryRatingStore() *InMemoryRatingStore {
	return &InMemoryRatingStore{ratings: make(map[string]map[string]int)}
}

func (s *InMemoryRatingStore) Upsert(_ context.Context, animeID, userID string, score int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ratings[animeID] == nil {
		s.ratings[animeID] = make(map[string]int)
	}
	s.ratings[animeID][userID] = score
	return nil
}

func (s *InMemoryRatingStore) GetSummary(_ context.Context, animeID string) (RatingSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := s.ratings[animeID]
	if len(users) == 0 {
		return RatingSummary{AnimeID: animeID}, nil
	}
	total := 0
	for _, score := range users {
		total += score
	}
	return RatingSummary{
		AnimeID:      animeID,
		AverageScore: float64(total) / float64(len(users)),
		TotalRatings: len(users),
	}, nil
}

func (s *InMemoryRatingStore) GetUserRating(_ context.Context, animeID, userID string) (int, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := s.ratings[animeID]
	if users == nil {
		return 0, false, nil
	}
	score, ok := users[userID]
	return score, ok, nil
}

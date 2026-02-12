// Package store provides an in-memory ratings store.
// TODO: replace with persistent storage (PostgreSQL) for production.
package store

import (
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

type RatingStore struct {
	mu      sync.RWMutex
	ratings map[string]map[string]int // anime_id -> user_id -> score
}

func NewRatingStore() *RatingStore {
	return &RatingStore{ratings: make(map[string]map[string]int)}
}

func (s *RatingStore) Upsert(animeID, userID string, score int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ratings[animeID] == nil {
		s.ratings[animeID] = make(map[string]int)
	}
	s.ratings[animeID][userID] = score
}

func (s *RatingStore) GetSummary(animeID string) RatingSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := s.ratings[animeID]
	if len(users) == 0 {
		return RatingSummary{AnimeID: animeID}
	}
	total := 0
	for _, score := range users {
		total += score
	}
	return RatingSummary{
		AnimeID:      animeID,
		AverageScore: float64(total) / float64(len(users)),
		TotalRatings: len(users),
	}
}

func (s *RatingStore) GetUserRating(animeID, userID string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := s.ratings[animeID]
	if users == nil {
		return 0, false
	}
	score, ok := users[userID]
	return score, ok
}

package store

import (
	"testing"
)

func TestRatingStore_UpsertAndSummary(t *testing.T) {
	s := NewRatingStore()

	// Empty initially
	summary := s.GetSummary("anime-1")
	if summary.TotalRatings != 0 {
		t.Fatalf("expected 0 ratings, got %d", summary.TotalRatings)
	}

	// Add ratings
	s.Upsert("anime-1", "user-a", 8)
	s.Upsert("anime-1", "user-b", 6)

	summary = s.GetSummary("anime-1")
	if summary.TotalRatings != 2 {
		t.Fatalf("expected 2 ratings, got %d", summary.TotalRatings)
	}
	expectedAvg := 7.0
	if summary.AverageScore != expectedAvg {
		t.Fatalf("expected average %.1f, got %.1f", expectedAvg, summary.AverageScore)
	}

	// Upsert overwrites
	s.Upsert("anime-1", "user-a", 10)
	summary = s.GetSummary("anime-1")
	if summary.TotalRatings != 2 {
		t.Fatalf("expected 2 ratings after upsert, got %d", summary.TotalRatings)
	}
	expectedAvg = 8.0
	if summary.AverageScore != expectedAvg {
		t.Fatalf("expected average %.1f after upsert, got %.1f", expectedAvg, summary.AverageScore)
	}
}

func TestRatingStore_GetUserRating(t *testing.T) {
	s := NewRatingStore()

	_, ok := s.GetUserRating("anime-1", "user-a")
	if ok {
		t.Fatal("expected no rating for non-existent user")
	}

	s.Upsert("anime-1", "user-a", 7)
	score, ok := s.GetUserRating("anime-1", "user-a")
	if !ok {
		t.Fatal("expected rating to exist")
	}
	if score != 7 {
		t.Fatalf("expected score 7, got %d", score)
	}
}

func TestRatingStore_MultipleAnime(t *testing.T) {
	s := NewRatingStore()
	s.Upsert("anime-1", "user-a", 9)
	s.Upsert("anime-2", "user-a", 3)

	s1 := s.GetSummary("anime-1")
	s2 := s.GetSummary("anime-2")
	if s1.AverageScore != 9 || s2.AverageScore != 3 {
		t.Fatalf("expected independent anime scores: got %.1f and %.1f", s1.AverageScore, s2.AverageScore)
	}
}

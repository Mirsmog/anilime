package store

import (
	"context"
	"testing"
)

func TestInMemoryRatingStore_UpsertAndSummary(t *testing.T) {
	s := NewInMemoryRatingStore()
	ctx := context.Background()

	// Empty initially
	summary, err := s.GetSummary(ctx, "anime-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalRatings != 0 {
		t.Fatalf("expected 0 ratings, got %d", summary.TotalRatings)
	}

	// Add ratings
	if err := s.Upsert(ctx, "anime-1", "user-a", 8); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := s.Upsert(ctx, "anime-1", "user-b", 6); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	summary, _ = s.GetSummary(ctx, "anime-1")
	if summary.TotalRatings != 2 {
		t.Fatalf("expected 2 ratings, got %d", summary.TotalRatings)
	}
	expectedAvg := 7.0
	if summary.AverageScore != expectedAvg {
		t.Fatalf("expected average %.1f, got %.1f", expectedAvg, summary.AverageScore)
	}

	// Upsert overwrites
	_ = s.Upsert(ctx, "anime-1", "user-a", 10)
	summary, _ = s.GetSummary(ctx, "anime-1")
	if summary.TotalRatings != 2 {
		t.Fatalf("expected 2 ratings after upsert, got %d", summary.TotalRatings)
	}
	expectedAvg = 8.0
	if summary.AverageScore != expectedAvg {
		t.Fatalf("expected average %.1f after upsert, got %.1f", expectedAvg, summary.AverageScore)
	}
}

func TestInMemoryRatingStore_GetUserRating(t *testing.T) {
	s := NewInMemoryRatingStore()
	ctx := context.Background()

	_, ok, err := s.GetUserRating(ctx, "anime-1", "user-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected no rating for non-existent user")
	}

	_ = s.Upsert(ctx, "anime-1", "user-a", 7)
	score, ok, err := s.GetUserRating(ctx, "anime-1", "user-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected rating to exist")
	}
	if score != 7 {
		t.Fatalf("expected score 7, got %d", score)
	}
}

func TestInMemoryRatingStore_MultipleAnime(t *testing.T) {
	s := NewInMemoryRatingStore()
	ctx := context.Background()
	_ = s.Upsert(ctx, "anime-1", "user-a", 9)
	_ = s.Upsert(ctx, "anime-2", "user-a", 3)

	s1, _ := s.GetSummary(ctx, "anime-1")
	s2, _ := s.GetSummary(ctx, "anime-2")
	if s1.AverageScore != 9 || s2.AverageScore != 3 {
		t.Fatalf("expected independent anime scores: got %.1f and %.1f", s1.AverageScore, s2.AverageScore)
	}
}

// TestRatingStoreInterface ensures both implementations satisfy the interface.
func TestRatingStoreInterface(t *testing.T) {
	var _ RatingStore = (*InMemoryRatingStore)(nil)
	var _ RatingStore = (*PostgresRatingStore)(nil)
}

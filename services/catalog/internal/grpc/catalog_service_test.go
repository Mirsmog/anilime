package grpcapi

import (
	"context"
	"testing"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/services/catalog/internal/store"
)

// stubStore is a no-op CatalogStore for unit tests that don't hit the DB.
type stubStore struct{ store.CatalogStore }

func (stubStore) GetAnimeByIDs(_ context.Context, ids []string) ([]store.Anime, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return nil, nil
}

func TestGetAnimeByIDs_EmptyRequest(t *testing.T) {
	svc := &CatalogService{Store: stubStore{}}

	resp, err := svc.GetAnimeByIDs(context.Background(), &catalogv1.GetAnimeByIDsRequest{})
	if err != nil {
		t.Fatalf("expected no error for empty request, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.GetAnime()) != 0 {
		t.Fatalf("expected empty anime list, got %d items", len(resp.GetAnime()))
	}
}

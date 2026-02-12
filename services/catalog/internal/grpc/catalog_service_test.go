package grpcapi

import (
	"context"
	"testing"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

func TestGetAnimeByIDs_EmptyRequest(t *testing.T) {
	svc := &CatalogService{} // DB is nil — мы не обращаемся к БД при пустом запросе

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

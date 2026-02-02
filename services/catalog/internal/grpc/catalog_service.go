package grpcapi

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

type CatalogService struct {
	catalogv1.UnimplementedCatalogServiceServer
	DB *pgxpool.Pool
}

func (s *CatalogService) GetEpisodesByIDs(ctx context.Context, req *catalogv1.GetEpisodesByIDsRequest) (*catalogv1.GetEpisodesByIDsResponse, error) {
	ids := req.GetEpisodeIds()
	if len(ids) == 0 {
		return &catalogv1.GetEpisodesByIDsResponse{Episodes: nil}, nil
	}

	q := `
SELECT id::text, anime_id::text, number, title, aired_at
FROM episodes
WHERE id::text = ANY($1)
`

	rows, err := s.DB.Query(ctx, q, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()

	resp := &catalogv1.GetEpisodesByIDsResponse{}
	for rows.Next() {
		var (
			id, animeID, title string
			number             int32
			airedAt            *time.Time
		)
		if err := rows.Scan(&id, &animeID, &number, &title, &airedAt); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		pb := &catalogv1.Episode{Id: id, AnimeId: animeID, Number: number, Title: title}
		if airedAt != nil {
			pb.AiredAtRfc3339 = airedAt.UTC().Format(time.RFC3339)
		}
		resp.Episodes = append(resp.Episodes, pb)
	}
	return resp, nil
}

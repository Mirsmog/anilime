package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

type CatalogClient struct {
	Conn   *grpc.ClientConn
	Client catalogv1.CatalogServiceClient
}

func NewCatalogClient(addr string) (*CatalogClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &CatalogClient{Conn: conn, Client: catalogv1.NewCatalogServiceClient(conn)}, nil
}

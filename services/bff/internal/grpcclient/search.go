package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	searchv1 "github.com/example/anime-platform/gen/search/v1"
)

type SearchClient struct {
	Conn   *grpc.ClientConn
	Client searchv1.SearchServiceClient
}

func NewSearchClient(addr string) (*SearchClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &SearchClient{Conn: conn, Client: searchv1.NewSearchServiceClient(conn)}, nil
}

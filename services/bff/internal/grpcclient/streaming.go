package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	streamingv1 "github.com/example/anime-platform/gen/streaming/v1"
)

type StreamingResolverClient struct {
	Conn   *grpc.ClientConn
	Client streamingv1.StreamingResolverServiceClient
}

func NewStreamingResolverClient(addr string) (*StreamingResolverClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &StreamingResolverClient{Conn: conn, Client: streamingv1.NewStreamingResolverServiceClient(conn)}, nil
}

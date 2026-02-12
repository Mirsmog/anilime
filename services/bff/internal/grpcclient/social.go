package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
)

type SocialClient struct {
	Conn   *grpc.ClientConn
	Client socialv1.SocialServiceClient
}

func NewSocialClient(addr string) (*SocialClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &SocialClient{Conn: conn, Client: socialv1.NewSocialServiceClient(conn)}, nil
}

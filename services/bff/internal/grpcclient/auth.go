package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/example/anime-platform/gen/auth/v1"
)

type AuthClient struct {
	Conn   *grpc.ClientConn
	Client authv1.AuthServiceClient
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AuthClient{Conn: conn, Client: authv1.NewAuthServiceClient(conn)}, nil
}

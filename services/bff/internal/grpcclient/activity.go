package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	activityv1 "github.com/example/anime-platform/gen/activity/v1"
)

type ActivityClient struct {
	Conn   *grpc.ClientConn
	Client activityv1.ActivityServiceClient
}

func NewActivityClient(addr string) (*ActivityClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ActivityClient{Conn: conn, Client: activityv1.NewActivityServiceClient(conn)}, nil
}

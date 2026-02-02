package config

import (
	"os"
	"strings"
)

type GRPCConfig struct {
	Addr string
}

func LoadGRPC() GRPCConfig {
	addr := strings.TrimSpace(os.Getenv("GRPC_ADDR"))
	if addr == "" {
		addr = ":9092"
	}
	return GRPCConfig{Addr: addr}
}

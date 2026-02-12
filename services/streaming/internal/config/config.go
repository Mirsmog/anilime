package config

import (
	"os"
	"strings"
)

type Config struct {
	StreamingResolverAddr string
}

func Load() Config {
	addr := strings.TrimSpace(os.Getenv("STREAMING_RESOLVER_GRPC_ADDR"))
	if addr == "" {
		addr = "streaming-resolver:9095"
	}
	return Config{StreamingResolverAddr: addr}
}

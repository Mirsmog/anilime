package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	HTTP *http.Server
}

type Options struct {
	Addr        string
	ServiceName string
	Logger      *zap.Logger
	Router      chi.Router
}

func New(opts Options) *Server {
	if opts.Router == nil {
		r := chi.NewRouter()
		opts.Router = r
	}

	srv := &http.Server{
		Addr:              opts.Addr,
		Handler:           opts.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return &Server{HTTP: srv}
}

func (s *Server) Start(log *zap.Logger) error {
	log.Info("http server starting", zap.String("addr", s.HTTP.Addr))
	return s.HTTP.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.HTTP.Shutdown(ctx)
}

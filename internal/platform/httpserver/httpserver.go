package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
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

	// Minimal CORS. Tighten for production.
	opts.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health endpoints
	opts.Router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	opts.Router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

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

package httpserver

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
)

// RouterConfig holds optional configuration for SetupRouter.
type RouterConfig struct {
	// ReadyFunc is called on GET /readyz. Return a non-nil error to signal
	// service unavailability (HTTP 503). If nil, /readyz always returns 200.
	ReadyFunc func() error
	// Logger is used by the panic recovery middleware.
	Logger *zap.Logger
}

// SetupRouter attaches base middlewares and common endpoints.
// Must be called before registering any routes.
// Accepts an optional RouterConfig as the second argument.
func SetupRouter(r chi.Router, cfgs ...RouterConfig) {
	var cfg RouterConfig
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	r.Use(RequestIDMiddleware("X-Request-Id"))
	r.Use(panicRecovery(cfg.Logger))

	allowedOrigins := parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id", "X-Event-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if cfg.ReadyFunc != nil {
			if err := cfg.ReadyFunc(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable", "error": err.Error()})
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
}

// parseCORSOrigins splits a comma-separated list of origins.
// Falls back to the same-site wildcard pattern if the env var is not set.
func parseCORSOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		// Default: allow all in development; set CORS_ALLOWED_ORIGINS in production.
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// panicRecovery returns a middleware that recovers from panics,
// logs the stack trace, and writes a 500 Internal Server Error.
func panicRecovery(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					stack := debug.Stack()
					if log != nil {
						log.Error("panic recovered",
							zap.String("method", r.Method),
							zap.String("path", r.URL.Path),
							zap.Any("panic", v),
							zap.ByteString("stack", stack),
						)
					}
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

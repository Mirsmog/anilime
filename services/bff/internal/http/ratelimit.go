package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/httpserver"
)

// RateLimiter implements a simple per-IP token bucket rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   int
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a rate limiter with the given rate (req/s) and burst size.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(rl.burst), last: now}
		rl.buckets[key] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware returns an HTTP middleware that rate-limits requests by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		if !rl.allow(ip) {
			rid := httpserver.RequestIDFromContext(r.Context())
			api.RateLimited(w, "RATE_LIMITED", "Too many requests", rid, nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

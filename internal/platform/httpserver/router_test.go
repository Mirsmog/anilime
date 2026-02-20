package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestRouter(cfg ...RouterConfig) chi.Router {
	r := chi.NewRouter()
	SetupRouter(r, cfg...)
	return r
}

func TestHealthz(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", rr.Body.String())
	}
}

func TestReadyz_NoReadyFunc(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestReadyz_ReadyFuncOK(t *testing.T) {
	r := newTestRouter(RouterConfig{ReadyFunc: func() error { return nil }})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestReadyz_ReadyFuncError(t *testing.T) {
	r := newTestRouter(RouterConfig{ReadyFunc: func() error { return errors.New("db down") }})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
	body := rr.Body.String()
	if body == "" {
		t.Fatal("expected non-empty error body")
	}
}

func TestPanicRecovery(t *testing.T) {
	r := newTestRouter()
	r.Get("/boom", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rr := httptest.NewRecorder()

	// Should not propagate the panic
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on panic, got %d", rr.Code)
	}
}

func TestCORS_DefaultWildcard(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	r := newTestRouter()
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS header to be set")
	}
}

func TestParseCORSOrigins_Empty(t *testing.T) {
	origins := parseCORSOrigins("")
	if len(origins) != 1 || origins[0] != "*" {
		t.Fatalf("expected ['*'], got %v", origins)
	}
}

func TestParseCORSOrigins_Single(t *testing.T) {
	origins := parseCORSOrigins("https://anilime.io")
	if len(origins) != 1 || origins[0] != "https://anilime.io" {
		t.Fatalf("expected single origin, got %v", origins)
	}
}

func TestParseCORSOrigins_Multiple(t *testing.T) {
	origins := parseCORSOrigins("https://anilime.io , https://www.anilime.io")
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d: %v", len(origins), origins)
	}
	if origins[0] != "https://anilime.io" || origins[1] != "https://www.anilime.io" {
		t.Fatalf("unexpected origins: %v", origins)
	}
}

func TestRequestIDInjected(t *testing.T) {
	r := newTestRouter()
	var capturedID string
	r.Get("/id", func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if capturedID == "" {
		t.Fatal("expected request ID to be injected into context")
	}
	if rr.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id response header")
	}
}

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret-key-32-bytes-long!!!")

func makeToken(subject, role string, exp time.Time) string {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Role: role,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := tok.SignedString(testSecret)
	return signed
}

func newVerifier() JWTVerifier { return JWTVerifier{Secret: testSecret} }

// withRole injects role into context using the unexported key (same package).
func withRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxKeyRole{}, role)
}

// ─── JWTVerifier tests ──────────────────────────────────────────────────────

func TestJWTVerifier_ValidToken(t *testing.T) {
	tok := makeToken("user-1", "user", time.Now().Add(time.Hour))
	claims, err := newVerifier().Parse(tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("expected subject 'user-1', got %q", claims.Subject)
	}
	if claims.Role != "user" {
		t.Fatalf("expected role 'user', got %q", claims.Role)
	}
}

func TestJWTVerifier_ExpiredToken(t *testing.T) {
	tok := makeToken("user-1", "user", time.Now().Add(-time.Hour))
	_, err := newVerifier().Parse(tok)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTVerifier_WrongSecret(t *testing.T) {
	tok := makeToken("user-1", "user", time.Now().Add(time.Hour))
	_, err := JWTVerifier{Secret: []byte("wrong-secret")}.Parse(tok)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestJWTVerifier_MalformedToken(t *testing.T) {
	_, err := newVerifier().Parse("not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestJWTVerifier_TamperedPayload(t *testing.T) {
	tok := makeToken("user-1", "admin", time.Now().Add(time.Hour))
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatal("expected 3 JWT parts")
	}
	tampered := parts[0] + ".dGFtcGVyZWQ." + parts[2]
	_, err := newVerifier().Parse(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

// ─── RequireUser middleware tests ────────────────────────────────────────────

func callRequireUser(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	RequireUser(newVerifier())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(uid))
	})).ServeHTTP(rr, req)
	return rr
}

func TestRequireUser_ValidBearer(t *testing.T) {
	tok := makeToken("user-42", "user", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	rr := callRequireUser(req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "user-42" {
		t.Fatalf("expected 'user-42' in body, got %q", rr.Body.String())
	}
}

func TestRequireUser_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := callRequireUser(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireUser_NonBearerScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := callRequireUser(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireUser_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := callRequireUser(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireUser_ExpiredToken(t *testing.T) {
	tok := makeToken("user-1", "user", time.Now().Add(-time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := callRequireUser(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireUser_InjectsRoleIntoContext(t *testing.T) {
	tok := makeToken("user-99", "admin", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	var capturedRole string
	rr := httptest.NewRecorder()
	RequireUser(newVerifier())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRole, _ = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if capturedRole != "admin" {
		t.Fatalf("expected role 'admin', got %q", capturedRole)
	}
}

// ─── RequireAdmin middleware tests ───────────────────────────────────────────

func callRequireAdmin(ctx context.Context) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)
	return rr
}

func TestRequireAdmin_WithAdminRole(t *testing.T) {
	ctx := withRole(context.Background(), "admin")
	rr := callRequireAdmin(ctx)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin role, got %d", rr.Code)
	}
}

func TestRequireAdmin_WithUserRole(t *testing.T) {
	ctx := withRole(context.Background(), "user")
	rr := callRequireAdmin(ctx)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user role, got %d", rr.Code)
	}
}

func TestRequireAdmin_NoRole(t *testing.T) {
	rr := callRequireAdmin(context.Background())
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with no role, got %d", rr.Code)
	}
}

func TestRequireAdmin_CaseInsensitive(t *testing.T) {
	ctx := withRole(context.Background(), "ADMIN")
	rr := callRequireAdmin(ctx)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for ADMIN (case insensitive), got %d", rr.Code)
	}
}

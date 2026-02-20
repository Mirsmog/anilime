package tokens

import (
	"strings"
	"testing"
	"time"
)

func newService() Service {
	return Service{
		Secret:          []byte("test-jwt-secret-32-bytes-padded!"),
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
}

// ─── NewAccessToken tests ────────────────────────────────────────────────────

func TestNewAccessToken_HappyPath(t *testing.T) {
	svc := newService()
	now := time.Now().UTC()

	tok, exp, err := svc.NewAccessToken("user-1", "admin", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token")
	}
	if !exp.After(now) {
		t.Fatalf("expected expiry after now, got %v", exp)
	}

	// Roundtrip
	claims, err := svc.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("expected subject 'user-1', got %q", claims.Subject)
	}
	if claims.Role != "admin" {
		t.Fatalf("expected role 'admin', got %q", claims.Role)
	}
}

func TestNewAccessToken_MissingSecret(t *testing.T) {
	svc := Service{Secret: nil, AccessTokenTTL: time.Hour}
	_, _, err := svc.NewAccessToken("user-1", "user", time.Now())
	if err == nil {
		t.Fatal("expected error when secret is empty")
	}
}

func TestNewAccessToken_ZeroTime_UsesNow(t *testing.T) {
	svc := newService()
	before := time.Now().Add(-time.Second)
	tok, exp, err := svc.NewAccessToken("user-1", "user", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exp.After(before) {
		t.Fatalf("expected expiry after 'before', got %v", exp)
	}
	// Should parse fine
	_, err = svc.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
}

// ─── ParseAccessToken tests ──────────────────────────────────────────────────

func TestParseAccessToken_Expired(t *testing.T) {
	svc := Service{
		Secret:         []byte("test-jwt-secret-32-bytes-padded!"),
		AccessTokenTTL: -time.Hour, // already expired at creation
	}
	tok, _, err := svc.NewAccessToken("user-1", "user", time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}
	_, err = svc.ParseAccessToken(tok)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	svc1 := newService()
	svc2 := Service{Secret: []byte("different-secret-32-bytes-padded"), AccessTokenTTL: time.Hour}

	tok, _, err := svc1.NewAccessToken("user-1", "user", time.Now())
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}
	_, err = svc2.ParseAccessToken(tok)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseAccessToken_Malformed(t *testing.T) {
	svc := newService()
	_, err := svc.ParseAccessToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestParseAccessToken_TamperedPayload(t *testing.T) {
	svc := newService()
	tok, _, err := svc.NewAccessToken("user-1", "user", time.Now())
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}

	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatal("expected 3 parts")
	}
	tampered := parts[0] + ".dGFtcGVyZWQ." + parts[2]
	_, err = svc.ParseAccessToken(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

// ─── NewRefreshToken tests ───────────────────────────────────────────────────

func TestNewRefreshToken_Unique(t *testing.T) {
	raw1, hash1, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	raw2, hash2, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if raw1 == raw2 {
		t.Fatal("expected unique raw tokens")
	}
	if hash1 == hash2 {
		t.Fatal("expected unique hashes")
	}
}

func TestNewRefreshToken_HashLength(t *testing.T) {
	_, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	// SHA-256 hex = 64 chars
	if len(hash) != 64 {
		t.Fatalf("expected SHA-256 hex hash (64 chars), got %d: %s", len(hash), hash)
	}
}

func TestNewRefreshToken_NonEmpty(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("expected non-empty raw and hash")
	}
}

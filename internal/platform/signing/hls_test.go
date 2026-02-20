package signing

import (
	"net/url"
	"testing"
	"time"
)

func newSigner() *Signer { return New("test-signing-secret-32-bytes-ok!") }

const testStreamURL = "https://cdn.example.com/stream/ep1/index.m3u8"

func TestSign_Verify_HappyPath(t *testing.T) {
	s := newSigner()
	exp := time.Now().Add(time.Hour)

	signed := s.Sign(testStreamURL, "user-1", exp)
	if !s.Verify(testStreamURL, "user-1", signed.Exp, signed.Sig) {
		t.Fatal("expected Verify to return true for valid signature")
	}
}

func TestVerify_Expired(t *testing.T) {
	s := newSigner()
	exp := time.Now().Add(-time.Hour)

	signed := s.Sign(testStreamURL, "user-1", exp)
	if s.Verify(testStreamURL, "user-1", signed.Exp, signed.Sig) {
		t.Fatal("expected Verify to return false for expired signature")
	}
}

func TestVerify_TamperedURL(t *testing.T) {
	s := newSigner()
	exp := time.Now().Add(time.Hour)
	signed := s.Sign("https://cdn.example.com/ep1", "user-1", exp)

	if s.Verify("https://cdn.example.com/ep2", "user-1", signed.Exp, signed.Sig) {
		t.Fatal("expected Verify to fail for tampered URL")
	}
}

func TestVerify_TamperedUserID(t *testing.T) {
	s := newSigner()
	exp := time.Now().Add(time.Hour)
	signed := s.Sign("https://cdn.example.com/ep1", "user-1", exp)

	if s.Verify("https://cdn.example.com/ep1", "user-2", signed.Exp, signed.Sig) {
		t.Fatal("expected Verify to fail for different user")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	s1 := newSigner()
	s2 := New("different-secret-32-bytes-padded!!")
	exp := time.Now().Add(time.Hour)

	signed := s1.Sign("https://cdn.example.com/ep1", "user-1", exp)
	if s2.Verify("https://cdn.example.com/ep1", "user-1", signed.Exp, signed.Sig) {
		t.Fatal("expected Verify to fail with different secret")
	}
}

func TestBuildSignedURL_ExtractSigned_Roundtrip(t *testing.T) {
	s := newSigner()
	rawURL := "https://cdn.example.com/stream/ep1/index.m3u8"
	exp := time.Now().Add(time.Hour)
	signed := s.Sign(rawURL, "user-42", exp)

	proxyURL, err := BuildSignedURL("https://proxy.example.com/hls", signed)
	if err != nil {
		t.Fatalf("BuildSignedURL: %v", err)
	}

	u, _ := url.Parse(proxyURL)
	extractedURL, extractedUID, extractedExp, extractedSig, err := ExtractSigned(u.Query())
	if err != nil {
		t.Fatalf("ExtractSigned: %v", err)
	}

	if extractedURL != rawURL {
		t.Fatalf("expected URL %q, got %q", rawURL, extractedURL)
	}
	if extractedUID != "user-42" {
		t.Fatalf("expected uid 'user-42', got %q", extractedUID)
	}
	if extractedExp != signed.Exp {
		t.Fatalf("expected exp %d, got %d", signed.Exp, extractedExp)
	}
	if !s.Verify(extractedURL, extractedUID, extractedExp, extractedSig) {
		t.Fatal("extracted signature should verify successfully")
	}
}

func TestExtractSigned_MissingParams(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{"missing url", url.Values{"uid": {"u"}, "exp": {"1"}, "sig": {"s"}}},
		{"missing uid", url.Values{"url": {"u"}, "exp": {"1"}, "sig": {"s"}}},
		{"missing exp", url.Values{"url": {"u"}, "uid": {"u"}, "sig": {"s"}}},
		{"missing sig", url.Values{"url": {"u"}, "uid": {"u"}, "exp": {"1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := ExtractSigned(tt.values)
			if err == nil {
				t.Fatal("expected error for missing param")
			}
		})
	}
}

func TestSignWithHeaders_ExtractHeaders(t *testing.T) {
	s := newSigner()
	rawURL := "https://cdn.example.com/ep1"
	exp := time.Now().Add(time.Hour)
	hdrs := map[string]string{"Referer": "https://anilime.io", "X-Token": "abc123"}

	signed := s.SignWithHeaders(rawURL, "user-1", exp, hdrs)

	proxyURL, err := BuildSignedURL("https://proxy.example.com/hls", signed)
	if err != nil {
		t.Fatalf("BuildSignedURL: %v", err)
	}

	u, _ := url.Parse(proxyURL)
	extracted := ExtractHeaders(u.Query())
	if extracted["Referer"] != "https://anilime.io" {
		t.Fatalf("expected Referer header, got %q", extracted["Referer"])
	}
	if extracted["X-Token"] != "abc123" {
		t.Fatalf("expected X-Token header, got %q", extracted["X-Token"])
	}
}

func TestExtractHeaders_NoHeader(t *testing.T) {
	vals := url.Values{"url": {"u"}}
	hdrs := ExtractHeaders(vals)
	if hdrs != nil {
		t.Fatalf("expected nil for missing hdr param, got %v", hdrs)
	}
}

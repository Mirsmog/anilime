package rewriter

import (
	"strings"
	"testing"
)

const (
	testProxyBase = "https://proxy.example.com/hls"
	testBaseURL   = "https://cdn.example.com/stream/episode1/index.m3u8"
	testSecret    = "supersecret"
	testUID       = "user-42"
	testExp       = "9999999999"
	testSegment   = "seg0.ts"
)

func defaultParams() SigningParams {
	return SigningParams{Secret: testSecret, UID: testUID, Exp: testExp}
}

// ─── RewriteM3U8 ─────────────────────────────────────────────────────────────

func TestRewriteM3U8_CommentsAndEmptyLinesPassThrough(t *testing.T) {
	body := "#EXTM3U\n#EXT-X-VERSION:3\n\n#EXT-X-ENDLIST"
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	if got != body {
		t.Fatalf("expected comments/empty lines unchanged\nwant: %q\ngot:  %q", body, got)
	}
}

func TestRewriteM3U8_RelativeSegmentResolvedAndProxied(t *testing.T) {
	body := "#EXTM3U\nseg0.ts"
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[1], testProxyBase+"?url=") {
		t.Fatalf("segment line should start with proxy URL: %q", lines[1])
	}
	if !strings.Contains(lines[1], "cdn.example.com") {
		t.Fatalf("resolved base URL should appear in proxy param: %q", lines[1])
	}
}

func TestRewriteM3U8_AbsoluteSegmentPassedAsIs(t *testing.T) {
	body := "#EXTM3U\nhttps://other.cdn.net/ep1/seg0.ts"
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[1], "other.cdn.net") {
		t.Fatalf("absolute URL should be forwarded to proxy: %q", lines[1])
	}
	if !strings.HasPrefix(lines[1], testProxyBase) {
		t.Fatalf("segment line should still go through proxy: %q", lines[1])
	}
}

func TestRewriteM3U8_URITagRewritten(t *testing.T) {
	body := `#EXT-X-I-FRAME-STREAM-INF:BANDWIDTH=100,URI="iframe.m3u8"`
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	if !strings.Contains(got, testProxyBase) {
		t.Fatalf("URI= tag value should be rewritten to proxy URL: %q", got)
	}
	if strings.Contains(got, `URI="iframe.m3u8"`) {
		t.Fatal("original URI value should be replaced")
	}
}

func TestRewriteM3U8_SignatureIncludesUIDAndExp(t *testing.T) {
	body := testSegment
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	if !strings.Contains(got, "uid="+testUID) {
		t.Fatalf("proxy URL should contain uid param: %q", got)
	}
	if !strings.Contains(got, "exp="+testExp) {
		t.Fatalf("proxy URL should contain exp param: %q", got)
	}
	if !strings.Contains(got, "sig=") {
		t.Fatalf("proxy URL should contain sig param: %q", got)
	}
}

func TestRewriteM3U8_DifferentSecretsProduceDifferentSigs(t *testing.T) {
	body := testSegment
	params1 := SigningParams{Secret: "secret-a", UID: testUID, Exp: testExp}
	params2 := SigningParams{Secret: "secret-b", UID: testUID, Exp: testExp}
	out1 := RewriteM3U8(body, testBaseURL, testProxyBase, params1)
	out2 := RewriteM3U8(body, testBaseURL, testProxyBase, params2)
	if out1 == out2 {
		t.Fatal("different secrets should produce different signatures")
	}
}

func TestRewriteM3U8_HeadersParamIncluded(t *testing.T) {
	body := testSegment
	params := SigningParams{Secret: testSecret, UID: testUID, Exp: testExp, Hdr: "dGVzdA"}
	got := RewriteM3U8(body, testBaseURL, testProxyBase, params)
	if !strings.Contains(got, "hdr=dGVzdA") {
		t.Fatalf("hdr param should appear when set: %q", got)
	}
}

func TestRewriteM3U8_NoHeadersParamOmitted(t *testing.T) {
	body := testSegment
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	if strings.Contains(got, "hdr=") {
		t.Fatalf("hdr param should be absent when not set: %q", got)
	}
}

func TestRewriteM3U8_MultiSegmentPlaylist(t *testing.T) {
	body := "#EXTM3U\n#EXT-X-VERSION:3\nseg0.ts\nseg1.ts\nseg2.ts\n#EXT-X-ENDLIST"
	got := RewriteM3U8(body, testBaseURL, testProxyBase, defaultParams())
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}
	for _, i := range []int{2, 3, 4} {
		if !strings.HasPrefix(lines[i], testProxyBase) {
			t.Fatalf("line %d should be a proxied URL: %q", i, lines[i])
		}
	}
}

// ─── resolveURL ──────────────────────────────────────────────────────────────

func TestResolveURL_AbsoluteURLUnchanged(t *testing.T) {
	got := resolveURL(testBaseURL, "https://cdn.net/seg.ts")
	if got != "https://cdn.net/seg.ts" {
		t.Fatalf("absolute URL should pass through unchanged: %q", got)
	}
}

func TestResolveURL_RelativePath(t *testing.T) {
	got := resolveURL("https://cdn.example.com/stream/ep1/index.m3u8", testSegment)
	want := "https://cdn.example.com/stream/ep1/seg0.ts"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestResolveURL_AbsolutePath(t *testing.T) {
	got := resolveURL("https://cdn.example.com/stream/ep1/index.m3u8", "/hls/seg0.ts")
	want := "https://cdn.example.com/hls/seg0.ts"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

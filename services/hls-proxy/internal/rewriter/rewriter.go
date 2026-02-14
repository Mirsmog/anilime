package rewriter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"
)

type SigningParams struct {
	Secret string
	UID    string
	Exp    string
	Hdr    string // base64 encoded headers
}

func RewriteM3U8(body string, baseURL string, proxyBase string, params SigningParams) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			// Check for URI= in tags (like EXT-X-I-FRAME-STREAM-INF)
			if strings.Contains(trim, "URI=\"") {
				line = rewriteURITag(line, baseURL, proxyBase, params)
			}
			out = append(out, line)
			continue
		}

		resolved := resolveURL(baseURL, trim)
		out = append(out, buildProxyURL(proxyBase, resolved, params))
	}
	return strings.Join(out, "\n")
}

func buildProxyURL(proxyBase, targetURL string, params SigningParams) string {
	// Generate new signature for this specific URL
	sig := signURL(targetURL, params.UID, params.Exp, params.Secret)
	
	result := fmt.Sprintf("%s?url=%s&exp=%s&uid=%s&sig=%s",
		proxyBase,
		url.QueryEscape(targetURL),
		params.Exp,
		params.UID,
		sig,
	)
	if params.Hdr != "" {
		result += "&hdr=" + params.Hdr
	}
	return result
}

func signURL(rawURL, uid, exp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(rawURL))
	mac.Write([]byte("|"))
	mac.Write([]byte(uid))
	mac.Write([]byte("|"))
	mac.Write([]byte(exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func rewriteURITag(line, baseURL, proxyBase string, params SigningParams) string {
	// Find URI="..." and rewrite it
	start := strings.Index(line, "URI=\"")
	if start == -1 {
		return line
	}
	start += 5 // len("URI=\"")
	end := strings.Index(line[start:], "\"")
	if end == -1 {
		return line
	}
	uri := line[start : start+end]
	resolved := resolveURL(baseURL, uri)
	newURI := buildProxyURL(proxyBase, resolved, params)
	return line[:start] + newURI + line[start+end:]
}

func resolveURL(baseURL, ref string) string {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ref
	}
	if strings.HasPrefix(ref, "/") {
		base.Path = ref
		base.RawQuery = ""
		return base.String()
	}
	base.Path = path.Join(path.Dir(base.Path), ref)
	base.RawQuery = ""
	return base.String()
}

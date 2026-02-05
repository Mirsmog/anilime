package rewriter

import (
	"net/url"
	"path"
	"strings"
)

func RewriteM3U8(body string, baseURL string, proxyBase string) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			out = append(out, line)
			continue
		}

		resolved := resolveURL(baseURL, trim)
		out = append(out, proxyBase+urlEncode(resolved))
	}
	return strings.Join(out, "\n")
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

func urlEncode(raw string) string {
	return "?url=" + url.QueryEscape(raw)
}

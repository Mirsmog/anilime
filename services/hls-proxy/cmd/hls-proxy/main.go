package main

import (
	"context"
	"io"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/internal/platform/signing"
	"github.com/example/anime-platform/services/hls-proxy/internal/config"
	"github.com/example/anime-platform/services/hls-proxy/internal/rewriter"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log, err := logging.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	s := signing.New(cfg.SigningSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("/hls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")
		w.Header().Set("Access-Control-Max-Age", "3600")
		
		// Handle preflight FIRST
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		rawURL, uid, exp, sig, err := signing.ExtractSigned(r.URL.Query())
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !s.Verify(rawURL, uid, exp, sig) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Extract custom headers from signed URL
		customHeaders := signing.ExtractHeaders(r.URL.Query())

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		
		// Apply custom headers from provider first, then fallback headers
		if len(customHeaders) > 0 {
			for k, v := range customHeaders {
				req.Header.Set(k, v)
			}
		} else {
			applyHiAnimeHeaders(req)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "upstream", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		isM3U8 := strings.Contains(contentType, "application/vnd.apple.mpegurl") || 
			strings.Contains(contentType, "application/x-mpegurl") || 
			strings.Contains(contentType, "audio/mpegurl") || 
			strings.Contains(contentType, "application/x-mpegURL") || 
			strings.Contains(contentType, "audio/x-mpegurl") ||
			strings.HasSuffix(strings.ToLower(rawURL), ".m3u8") ||
			strings.HasSuffix(strings.ToLower(rawURL), ".m3u")
		
		if isM3U8 {
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "upstream", http.StatusBadGateway)
				return
			}
			
			// Build proxy base URL with scheme
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
				scheme = fwdProto
			}
			proxyBase := scheme + "://" + r.Host + "/hls"
			
			// Pass signing params to rewriter for re-signing URLs
			signingParams := rewriter.SigningParams{
				Secret: cfg.SigningSecret,
				UID:    uid,
				Exp:    r.URL.Query().Get("exp"),
				Hdr:    r.URL.Query().Get("hdr"),
			}
			
			body := rewriter.RewriteM3U8(string(data), rawURL, proxyBase, signingParams)
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write([]byte(body))
			return
		}

		// Copy headers from upstream but skip CORS headers (we set our own)
		for k, vals := range resp.Header {
			kLower := strings.ToLower(k)
			if strings.HasPrefix(kLower, "access-control-") {
				continue // Skip upstream CORS headers
			}
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("http server starting", zap.String("addr", cfg.HTTPAddr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("http serve", zap.Error(err))
		run.Exit(1)
	}
}

func applyHiAnimeHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", "https://megacloud.blog")
	req.Header.Set("Referer", "https://hianime.to/")
	req.Header.Set("Sec-Ch-Ua", "\"Chromium\";v=\"134\", \"Not:A-Brand\";v=\"24\", \"Brave\";v=\"134\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "\"Windows\"")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Sec-Gpc", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
}

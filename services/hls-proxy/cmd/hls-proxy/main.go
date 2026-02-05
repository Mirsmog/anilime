package main

import (
	"context"
	"io"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/internal/platform/signing"
	"github.com/example/anime-platform/services/hls-proxy/internal/config"
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
		rawURL, uid, exp, sig, err := signing.ExtractSigned(r.URL.Query())
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !s.Verify(rawURL, uid, exp, sig) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		req.Header.Set("User-Agent", "anime-platform-proxy/1.0")
		req.Header.Set("Referer", "https://hianime.to/")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "upstream", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, vals := range resp.Header {
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

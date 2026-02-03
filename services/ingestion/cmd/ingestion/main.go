package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/run"
	"github.com/example/anime-platform/services/ingestion/internal/animekai"
	inkcfg "github.com/example/anime-platform/services/ingestion/internal/config"
	"github.com/example/anime-platform/services/ingestion/internal/grpcclient"
	"github.com/example/anime-platform/services/ingestion/internal/hianime"
	"github.com/example/anime-platform/services/ingestion/internal/jikan"
	"github.com/example/anime-platform/services/ingestion/internal/jobs"
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

	ink, err := inkcfg.Load()
	if err != nil {
		log.Error("load ingestion config", zap.Error(err))
		run.Exit(1)
	}

	catc, err := grpcclient.NewCatalogClient(ink.CatalogGRPCAddr)
	if err != nil {
		log.Error("init catalog grpc client", zap.Error(err))
		run.Exit(1)
	}
	defer catc.Conn.Close()

	r := chi.NewRouter()
	httpserver.SetupRouter(r)

	ak := animekai.New(ink.AnimeKaiBaseURL)
	job := jobs.AnimeKaiSync{AnimeKai: ak, Catalog: catc.Client}
	_ = job

	jc := jikan.New(ink.JikanBaseURL)
	jobs.JikanTrigger{Log: log, Jikan: jc, Catalog: catc.Client}.Register(r)

	hc := hianime.New(ink.HiAnimeBaseURL)
	hijob := jobs.HiAnimeSync{HiAnime: hc, Catalog: catc.Client, Jikan: jc}
	jobs.HiAnimeTrigger{Log: log, Job: hijob}.Register(r)

	r.Get("/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	r.Post("/v1/ingest/animekai/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		resAnimeID, episodeIDs, err := job.SyncAnime(r.Context(), id)
		if err != nil {
			api.WriteError(w, http.StatusBadGateway, "INGEST_FAILED", err.Error(), httpserver.RequestIDFromContext(r.Context()), nil)
			return
		}
		api.WriteJSON(w, http.StatusOK, map[string]any{"anime_id": resAnimeID, "episode_ids": episodeIDs})
	})

	srv := httpserver.New(httpserver.Options{Addr: cfg.HTTP.Addr, ServiceName: cfg.ServiceName, Logger: log, Router: r})

	runner := run.New(log)
	code := runner.WithSignals(func(ctx context.Context) error {
		go func() {
			<-ctx.Done()
			_ = srv.Shutdown(context.Background())
		}()
		return srv.Start(log)
	})

	log.Info("exit", zap.Int("code", code))
	run.Exit(code)
}

package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"encoding/json"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/internal/platform/config"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/logging"
	"github.com/example/anime-platform/internal/platform/natsconn"
	"github.com/example/anime-platform/internal/platform/run"
	inkcfg "github.com/example/anime-platform/services/ingestion/internal/config"
	"github.com/example/anime-platform/services/ingestion/internal/grpcclient"
	"github.com/example/anime-platform/services/ingestion/internal/hianime"
	"github.com/example/anime-platform/services/ingestion/internal/jikan"
	"github.com/example/anime-platform/services/ingestion/internal/jobs"
	"github.com/example/anime-platform/services/ingestion/internal/queue"
	"github.com/example/anime-platform/services/ingestion/internal/ratelimit"
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

	jc := jikan.New(ink.JikanBaseURL)
	hc := hianime.New(ink.HiAnimeBaseURL)
	hijob := jobs.HiAnimeSync{HiAnime: hc, Catalog: catc.Client, Jikan: jc}

	// Optional HTTP triggers for local debugging. Prefer NATS jobs in production.
	if strings.TrimSpace(os.Getenv("ENABLE_HTTP_TRIGGERS")) == "true" {
		jobs.JikanTrigger{Log: log, Jikan: jc, Catalog: catc.Client}.Register(r)
		jobs.HiAnimeTrigger{Log: log, Job: hijob}.Register(r)
	}

	// Start JetStream worker
	nc, err := natsconn.Connect(natsconn.Options{URL: ink.NATSURL})
	if err != nil {
		log.Error("nats connect", zap.Error(err))
		run.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Error("jetstream", zap.Error(err))
		run.Exit(1)
	}

	jikanLimiter := ratelimit.NewRPS(ink.JikanRPS)
	defer jikanLimiter.Stop()
	hiaLimiter := ratelimit.NewRPS(ink.HiAnimeRPS)
	defer hiaLimiter.Stop()

	wrk, err := queue.NewWorker(log, nc, queue.Handlers{
		JikanSync: func(ctx context.Context, malID int) error {
			if err := jikanLimiter.Wait(ctx); err != nil {
				return err
			}
			resp, err := jc.GetAnime(ctx, malID)
			if err != nil {
				return err
			}
			pb := jikan.ToCatalogProto(resp)
			if _, err := catc.Client.UpsertJikanAnime(ctx, &catalogv1.UpsertJikanAnimeRequest{Anime: pb}); err != nil {
				return err
			}
			b, _ := json.Marshal(queue.HiAnimeSyncJob{MALID: malID})
			_, err = js.Publish("ingestion.hianime.sync", b)
			return err
		},
		HiAnimeSync: func(ctx context.Context, malID int) error {
			if err := hiaLimiter.Wait(ctx); err != nil {
				return err
			}
			_, _, _, err := hijob.SyncEpisodesByMALID(ctx, malID, "")
			return err
		},
	})
	if err != nil {
		log.Error("worker init", zap.Error(err))
		run.Exit(1)
	}
	if err := wrk.EnsureStream(context.Background()); err != nil {
		log.Error("ensure stream", zap.Error(err))
		run.Exit(1)
	}
	go func() {
		if err := wrk.Run(context.Background()); err != nil {
			log.Error("worker stopped", zap.Error(err))
		}
	}()

	r.Get("/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
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

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/nats-io/nats.go"

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

	// Bulk import on startup: if catalog is empty, enqueue top-500 anime.
	go func() {
		time.Sleep(5 * time.Second) // wait for catalog service to be ready
		ids, err := catc.Client.GetAnimeIDs(context.Background(), &catalogv1.GetAnimeIDsRequest{})
		if err != nil {
			log.Warn("bulk import: could not check catalog size", zap.Error(err))
			return
		}
		if len(ids.GetAnimeIds()) > 0 {
			log.Info("bulk import: catalog not empty, skipping", zap.Int("count", len(ids.GetAnimeIds())))
			return
		}
		log.Info("bulk import: catalog is empty, enqueueing top-500 anime")
		published := publishJikanPages(context.Background(), log, jc, js, "top", 20)
		log.Info("bulk import: done", zap.Int("published", published))
	}()

	// Cron: sync current season every 24h, refresh top-100 every 7 days.
	go func() {
		seasonTicker := time.NewTicker(24 * time.Hour)
		topTicker := time.NewTicker(7 * 24 * time.Hour)
		defer seasonTicker.Stop()
		defer topTicker.Stop()
		for {
			select {
			case <-seasonTicker.C:
				log.Info("cron: syncing current season")
				n := publishJikanPages(context.Background(), log, jc, js, "season", 2)
				log.Info("cron: season sync done", zap.Int("published", n))
			case <-topTicker.C:
				log.Info("cron: refreshing top-100")
				n := publishJikanPages(context.Background(), log, jc, js, "top", 4)
				log.Info("cron: top refresh done", zap.Int("published", n))
			}
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

// publishJikanPages fetches pages from Jikan (top or season) and publishes mal_ids to NATS.
// kind: "top" → /top/anime, "season" → /seasons/now.
func publishJikanPages(ctx context.Context, log *zap.Logger, jc *jikan.Client, js natsJS, kind string, pages int) int {
	dedup := make(map[int32]struct{}, pages*25)
	for p := 1; p <= pages; p++ {
		var list *jikan.AnimeListResponse
		var err error
		if kind == "season" {
			list, err = jc.GetSeasonNow(ctx, p)
		} else {
			list, err = jc.GetTopAnime(ctx, p)
		}
		if err != nil {
			log.Warn("publishJikanPages: fetch error", zap.String("kind", kind), zap.Int("page", p), zap.Error(err))
			break
		}
		for _, a := range list.Data {
			if a.MalID > 0 {
				dedup[a.MalID] = struct{}{}
			}
		}
		if !list.Pagination.HasNextPage {
			break
		}
		// brief pause to respect Jikan rate limit (3 req/s free tier)
		time.Sleep(400 * time.Millisecond)
	}

	published := 0
	for malID := range dedup {
		b, _ := json.Marshal(queue.JikanSyncJob{MALID: int(malID)})
		if _, err := js.Publish("ingestion.jikan.sync", b); err != nil {
			log.Warn("publishJikanPages: nats publish error", zap.Int32("mal_id", malID), zap.Error(err))
			continue
		}
		published++
	}
	return published
}

// natsJS is the subset of nats.JetStreamContext used by publishJikanPages.
type natsJS interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
}

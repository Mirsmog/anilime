package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
	"github.com/example/anime-platform/services/search/internal/meili"
)

const (
	catalogSubject = "catalog.anime.upserted"
	indexName      = "anime"
)

type Config struct {
	CatalogClient catalogv1.CatalogServiceClient
	Meili         *meili.Client
	Log           *zap.Logger
	NATS          *nats.Conn
	ReindexEvery  time.Duration
}

type EventPayload struct {
	AnimeID string `json:"anime_id"`
}

type AnimeDoc struct {
	AnimeID       string   `json:"anime_id"`
	Title         string   `json:"title"`
	TitleEnglish  string   `json:"title_english"`
	TitleJapanese string   `json:"title_japanese"`
	Image         string   `json:"image"`
	Description   string   `json:"description"`
	Genres        []string `json:"genres"`
	Score         float32  `json:"score"`
	Status        string   `json:"status"`
	Type          string   `json:"type"`
	TotalEpisodes int32    `json:"total_episodes"`
}

func (c *Config) EnsureIndex(ctx context.Context) error {
	if err := c.Meili.EnsureIndex(ctx, indexName, "anime_id"); err != nil {
		return err
	}
	settings := map[string]any{
		"searchableAttributes": []string{"title", "title_english", "title_japanese", "description"},
		"filterableAttributes": []string{"genres", "status", "type", "score", "total_episodes"},
		"sortableAttributes":   []string{"score"},
	}
	return c.Meili.UpdateSettings(ctx, indexName, settings)
}

func (c *Config) Run(ctx context.Context) error {
	if err := c.EnsureIndex(ctx); err != nil {
		return err
	}
	js, err := c.NATS.JetStream()
	if err != nil {
		return err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "CATALOG_EVENTS",
		Subjects: []string{"catalog.>"},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return err
	}

	sub, err := js.PullSubscribe(catalogSubject, "search_indexer")
	if err != nil {
		return err
	}

	log := c.Log
	if c.ReindexEvery > 0 {
		go c.reindexLoop(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := sub.Fetch(1, nats.MaxWait(2*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			return err
		}
		for _, m := range msgs {
			if err := c.handleMsg(ctx, m); err != nil {
				log.Warn("index event failed", zap.Error(err))
				_ = m.Nak()
				continue
			}
			_ = m.Ack()
		}
	}
}

func (c *Config) handleMsg(ctx context.Context, msg *nats.Msg) error {
	var payload EventPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		return err
	}
	payload.AnimeID = strings.TrimSpace(payload.AnimeID)
	if payload.AnimeID == "" {
		return fmt.Errorf("missing anime_id")
	}
	return c.indexAnime(ctx, payload.AnimeID)
}

func (c *Config) reindexLoop(ctx context.Context) {
	ticker := time.NewTicker(c.ReindexEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.ReindexAll(ctx); err != nil {
				c.Log.Warn("reindex failed", zap.Error(err))
			}
		}
	}
}

func (c *Config) ReindexAll(ctx context.Context) error {
	ids, err := c.fetchAllAnimeIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := c.indexAnime(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) fetchAllAnimeIDs(ctx context.Context) ([]string, error) {
	resp, err := c.CatalogClient.GetAnimeIDs(ctx, &catalogv1.GetAnimeIDsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.AnimeIds, nil
}

func (c *Config) indexAnime(ctx context.Context, animeID string) error {
	resp, err := c.CatalogClient.GetAnimeByIDs(ctx, &catalogv1.GetAnimeByIDsRequest{AnimeIds: []string{animeID}})
	if err != nil {
		return err
	}
	if len(resp.Anime) == 0 {
		return nil
	}
	anime := resp.Anime[0]
	doc := AnimeDoc{
		AnimeID:       anime.Id,
		Title:         anime.Title,
		TitleEnglish:  anime.TitleEnglish,
		TitleJapanese: anime.TitleJapanese,
		Image:         anime.Image,
		Description:   anime.Description,
		Genres:        anime.Genres,
		Score:         anime.Score,
		Status:        anime.Status,
		Type:          anime.Type,
		TotalEpisodes: anime.TotalEpisodes,
	}
	return c.Meili.AddDocuments(ctx, indexName, []AnimeDoc{doc})
}

package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const catalogEventAnimeUpserted = "catalog.anime.upserted"

// PostgresCatalogStore is the production Postgres-backed implementation.
type PostgresCatalogStore struct {
	db *pgxpool.Pool
}

func NewPostgresCatalogStore(db *pgxpool.Pool) *PostgresCatalogStore {
	return &PostgresCatalogStore{db: db}
}

// ── Anime reads ────────────────────────────────────────────────────────────

func (s *PostgresCatalogStore) GetAnimeByIDs(ctx context.Context, ids []string) ([]Anime, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
SELECT id, title, title_english, title_japanese, image, description, genres, score, status, type, total_episodes
FROM anime WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()

	var out []Anime
	for rows.Next() {
		var a Anime
		var genresJSON []byte
		if err := rows.Scan(&a.ID, &a.Title, &a.TitleEnglish, &a.TitleJapanese, &a.Image, &a.Description, &genresJSON, &a.Score, &a.Status, &a.Type, &a.TotalEpisodes); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		_ = json.Unmarshal(genresJSON, &a.Genres)
		out = append(out, a)
	}
	return out, nil
}

func (s *PostgresCatalogStore) GetAllAnimeIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT id FROM anime ORDER BY updated_at DESC`)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *PostgresCatalogStore) ResolveAnimeIDByExternalID(ctx context.Context, provider, externalID string) (string, error) {
	var id uuid.UUID
	err := s.db.QueryRow(ctx,
		`SELECT anime_id FROM external_anime_ids WHERE provider=$1 AND provider_anime_id=$2`,
		provider, externalID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", status.Error(codes.NotFound, "not found")
		}
		return "", status.Error(codes.Internal, "db")
	}
	return id.String(), nil
}

// ── Anime writes ───────────────────────────────────────────────────────────

func (s *PostgresCatalogStore) AttachExternalAnimeID(ctx context.Context, provider, externalID, animeID string) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id)
VALUES ($1,$2,$3::uuid)
ON CONFLICT (provider, provider_anime_id) DO UPDATE SET anime_id = EXCLUDED.anime_id`,
		provider, externalID, animeID)
	if err != nil {
		return status.Error(codes.Internal, "db")
	}
	return nil
}

func (s *PostgresCatalogStore) UpsertJikanAnime(ctx context.Context, a JikanAnimeInput) (string, error) {
	externalID := fmt.Sprintf("%d", a.MalID)
	genresJSON, _ := json.Marshal(a.Genres)
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", status.Error(codes.Internal, "db begin")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var animeID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT anime_id FROM external_anime_ids WHERE provider='mal' AND provider_anime_id=$1`, externalID,
	).Scan(&animeID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", status.Error(codes.Internal, "db")
		}
		animeID = uuid.New()
		if _, err = tx.Exec(ctx, `
INSERT INTO anime (id, title, title_english, title_japanese, url, image, description, genres, sub_or_dub, type, status, other_name, total_episodes, score, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'unknown',$9,$10,'',$11,$12,$13,$14)`,
			animeID, a.Title, a.TitleEnglish, a.TitleJapanese, "", a.Image, a.Synopsis,
			genresJSON, a.Type, a.Status, a.TotalEpisodes, a.Score, now, now,
		); err != nil {
			return "", status.Error(codes.Internal, "db")
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id) VALUES ('mal',$1,$2)`,
			externalID, animeID,
		); err != nil {
			return "", status.Error(codes.Internal, "db")
		}
	} else {
		if _, err := tx.Exec(ctx, `
UPDATE anime
SET title=$2, title_english=$3, title_japanese=$4, image=$5, description=$6, genres=$7, type=$8, status=$9, total_episodes=$10, score=$11, updated_at=$12
WHERE id=$1`,
			animeID, a.Title, a.TitleEnglish, a.TitleJapanese, a.Image, a.Synopsis,
			genresJSON, a.Type, a.Status, a.TotalEpisodes, a.Score, now,
		); err != nil {
			return "", status.Error(codes.Internal, "db")
		}
	}

	if err := insertOutboxEvent(ctx, tx, map[string]any{"anime_id": animeID.String()}); err != nil {
		return "", status.Error(codes.Internal, "db outbox")
	}
	if err := tx.Commit(ctx); err != nil {
		return "", status.Error(codes.Internal, "db commit")
	}
	return animeID.String(), nil
}

// ── Episode reads ──────────────────────────────────────────────────────────

func (s *PostgresCatalogStore) GetEpisodesByAnimeID(ctx context.Context, animeID string) ([]Episode, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, anime_id, number, title, aired_at
FROM episodes WHERE anime_id=$1::uuid ORDER BY number ASC`, animeID)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()
	return scanEpisodes(rows)
}

func (s *PostgresCatalogStore) GetEpisodesByIDs(ctx context.Context, ids []string) ([]Episode, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
SELECT id, anime_id, number, title, aired_at
FROM episodes WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()
	return scanEpisodes(rows)
}

func (s *PostgresCatalogStore) GetProviderEpisodeID(ctx context.Context, episodeID, provider string) (string, error) {
	var providerEpisodeID string
	err := s.db.QueryRow(ctx,
		`SELECT provider_episode_id FROM external_episode_ids WHERE episode_id=$1::uuid AND provider=$2 ORDER BY provider_episode_id ASC LIMIT 1`,
		episodeID, provider,
	).Scan(&providerEpisodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", status.Error(codes.NotFound, "provider episode not found")
		}
		return "", status.Error(codes.Internal, "db query")
	}
	return providerEpisodeID, nil
}

// ── Episode writes ─────────────────────────────────────────────────────────

func (s *PostgresCatalogStore) UpsertHiAnimeEpisodes(ctx context.Context, animeID, slug string, episodes []EpisodeInput) ([]string, error) {
	id, err := uuid.Parse(strings.TrimSpace(animeID))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid anime_id")
	}
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, "db begin")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id)
VALUES ('hianime',$1,$2)
ON CONFLICT (provider, provider_anime_id) DO UPDATE SET anime_id = EXCLUDED.anime_id`,
		slug, id,
	); err != nil {
		return nil, status.Error(codes.Internal, "db")
	}

	episodeIDs, err := upsertEpisodes(ctx, tx, "hianime", id, episodes, now)
	if err != nil {
		return nil, err
	}

	if err := insertOutboxEvent(ctx, tx, map[string]any{"anime_id": animeID}); err != nil {
		return nil, status.Error(codes.Internal, "db outbox")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "db commit")
	}
	return episodeIDs, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func scanEpisodes(rows pgx.Rows) ([]Episode, error) {
	var out []Episode
	for rows.Next() {
		var ep Episode
		if err := rows.Scan(&ep.ID, &ep.AnimeID, &ep.Number, &ep.Title, &ep.AiredAt); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		out = append(out, ep)
	}
	return out, nil
}

func insertOutboxEvent(ctx context.Context, tx pgx.Tx, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO catalog_outbox (id, event_type, payload) VALUES ($1,$2,$3)`,
		uuid.New(), catalogEventAnimeUpserted, b,
	)
	return err
}

func upsertEpisodes(ctx context.Context, tx pgx.Tx, provider string, animeID uuid.UUID, episodes []EpisodeInput, now time.Time) ([]string, error) {
	ids := make([]string, 0, len(episodes))
	for _, ep := range episodes {
		if ep.ProviderEpisodeID == "" {
			continue
		}

		var epID uuid.UUID
		err := tx.QueryRow(ctx,
			`SELECT episode_id FROM external_episode_ids WHERE provider=$1 AND provider_episode_id=$2`,
			provider, ep.ProviderEpisodeID,
		).Scan(&epID)

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, status.Error(codes.Internal, "db")
			}
			epID = uuid.New()
			if err := writeEpisode(ctx, tx, epID, animeID, ep, now, true); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO external_episode_ids (provider, provider_episode_id, episode_id) VALUES ($1,$2,$3)`,
				provider, ep.ProviderEpisodeID, epID,
			); err != nil {
				return nil, status.Error(codes.Internal, "db")
			}
		} else {
			if err := writeEpisode(ctx, tx, epID, animeID, ep, now, false); err != nil {
				return nil, err
			}
		}
		ids = append(ids, epID.String())
	}
	return ids, nil
}

func writeEpisode(ctx context.Context, tx pgx.Tx, epID, animeID uuid.UUID, ep EpisodeInput, now time.Time, insert bool) error {
	var q string
	var args []any
	switch {
	case insert && ep.HasIsFiller:
		q = `INSERT INTO episodes (id, anime_id, number, title, url, is_filler, updated_at) VALUES ($1,$2,$3,$4,'',$5,$6)`
		args = []any{epID, animeID, ep.Number, ep.Title, ep.IsFiller, now}
	case insert:
		q = `INSERT INTO episodes (id, anime_id, number, title, url, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`
		args = []any{epID, animeID, ep.Number, ep.Title, ep.URL, now}
	case ep.HasIsFiller:
		q = `UPDATE episodes SET anime_id=$2, number=$3, title=$4, is_filler=$5, updated_at=$6 WHERE id=$1`
		args = []any{epID, animeID, ep.Number, ep.Title, ep.IsFiller, now}
	default:
		q = `UPDATE episodes SET anime_id=$2, number=$3, title=$4, url=$5, updated_at=$6 WHERE id=$1`
		args = []any{epID, animeID, ep.Number, ep.Title, ep.URL, now}
	}
	if _, err := tx.Exec(ctx, q, args...); err != nil {
		return status.Error(codes.Internal, "db")
	}
	return nil
}

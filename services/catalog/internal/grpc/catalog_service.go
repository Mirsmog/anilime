package grpcapi

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

	catalogv1 "github.com/example/anime-platform/gen/catalog/v1"
)

type CatalogService struct {
	catalogv1.UnimplementedCatalogServiceServer
	DB *pgxpool.Pool
}

const (
	catalogEventAnimeUpserted = "catalog.anime.upserted"
)

const qSelectAnimeIDByExternal = `SELECT anime_id FROM external_anime_ids WHERE provider=$1 AND provider_anime_id=$2`

func (s *CatalogService) insertOutboxEvent(ctx context.Context, tx pgx.Tx, eventType string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO catalog_outbox (id, event_type, payload) VALUES ($1,$2,$3)`, uuid.New(), eventType, b)
	return err
}

// episodeInput — общая структура для upsert-а эпизода из любого провайдера.
type episodeInput struct {
	providerEpisodeID string
	number            int32
	title             string
	url               string
	isFiller          bool
	hasIsFiller       bool // true если поле is_filler задано (hianime)
}

// upsertEpisodes выполняет upsert эпизодов внутри транзакции, устраняя дублирование
// между UpsertAnimeKaiAnime и UpsertHiAnimeEpisodes.
func upsertEpisodes(ctx context.Context, tx pgx.Tx, provider string, animeID uuid.UUID, episodes []episodeInput, now time.Time) ([]string, error) {
	episodeIDs := make([]string, 0, len(episodes))
	for _, ep := range episodes {
		if ep.providerEpisodeID == "" {
			continue
		}

		var episodeID uuid.UUID
		qFind := `SELECT episode_id FROM external_episode_ids WHERE provider=$1 AND provider_episode_id=$2`
		err := tx.QueryRow(ctx, qFind, provider, ep.providerEpisodeID).Scan(&episodeID)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, status.Error(codes.Internal, "db")
			}
			episodeID = uuid.New()
			if ep.hasIsFiller {
				qIns := `INSERT INTO episodes (id, anime_id, number, title, url, is_filler, updated_at) VALUES ($1,$2,$3,$4,'',$5,$6)`
				if _, err := tx.Exec(ctx, qIns, episodeID, animeID, ep.number, ep.title, ep.isFiller, now); err != nil {
					return nil, status.Error(codes.Internal, "db")
				}
			} else {
				qIns := `INSERT INTO episodes (id, anime_id, number, title, url, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`
				if _, err := tx.Exec(ctx, qIns, episodeID, animeID, ep.number, ep.title, ep.url, now); err != nil {
					return nil, status.Error(codes.Internal, "db")
				}
			}
			qInsMap := `INSERT INTO external_episode_ids (provider, provider_episode_id, episode_id) VALUES ($1,$2,$3)`
			if _, err := tx.Exec(ctx, qInsMap, provider, ep.providerEpisodeID, episodeID); err != nil {
				return nil, status.Error(codes.Internal, "db")
			}
		} else {
			if ep.hasIsFiller {
				qUpd := `UPDATE episodes SET anime_id=$2, number=$3, title=$4, is_filler=$5, updated_at=$6 WHERE id=$1`
				if _, err := tx.Exec(ctx, qUpd, episodeID, animeID, ep.number, ep.title, ep.isFiller, now); err != nil {
					return nil, status.Error(codes.Internal, "db")
				}
			} else {
				qUpd := `UPDATE episodes SET anime_id=$2, number=$3, title=$4, url=$5, updated_at=$6 WHERE id=$1`
				if _, err := tx.Exec(ctx, qUpd, episodeID, animeID, ep.number, ep.title, ep.url, now); err != nil {
					return nil, status.Error(codes.Internal, "db")
				}
			}
		}
		episodeIDs = append(episodeIDs, episodeID.String())
	}
	return episodeIDs, nil
}

func (s *CatalogService) GetEpisodesByIDs(ctx context.Context, req *catalogv1.GetEpisodesByIDsRequest) (*catalogv1.GetEpisodesByIDsResponse, error) {
	ids := req.GetEpisodeIds()
	if len(ids) == 0 {
		return &catalogv1.GetEpisodesByIDsResponse{Episodes: nil}, nil
	}

	q := `
SELECT id::text, anime_id::text, number, title, aired_at
FROM episodes
WHERE id::text = ANY($1)
`

	rows, err := s.DB.Query(ctx, q, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()

	resp := &catalogv1.GetEpisodesByIDsResponse{}
	for rows.Next() {
		var (
			id, animeID, title string
			number             int32
			airedAt            *time.Time
		)
		if err := rows.Scan(&id, &animeID, &number, &title, &airedAt); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		pb := &catalogv1.Episode{Id: id, AnimeId: animeID, Number: number, Title: title}
		if airedAt != nil {
			pb.AiredAtRfc3339 = airedAt.UTC().Format(time.RFC3339)
		}
		resp.Episodes = append(resp.Episodes, pb)
	}
	return resp, nil
}

func (s *CatalogService) GetProviderEpisodeID(ctx context.Context, req *catalogv1.GetProviderEpisodeIDRequest) (*catalogv1.GetProviderEpisodeIDResponse, error) {
	episodeID := strings.TrimSpace(req.GetEpisodeId())
	provider := strings.TrimSpace(req.GetProvider())
	if episodeID == "" || provider == "" {
		return nil, status.Error(codes.InvalidArgument, "episode_id and provider required")
	}

	var providerEpisodeID string
	err := s.DB.QueryRow(ctx, `SELECT provider_episode_id FROM external_episode_ids WHERE episode_id::text = $1 AND provider = $2 ORDER BY provider_episode_id ASC LIMIT 1`, episodeID, provider).Scan(&providerEpisodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "provider episode not found")
		}
		return nil, status.Error(codes.Internal, "db query")
	}
	return &catalogv1.GetProviderEpisodeIDResponse{ProviderEpisodeId: providerEpisodeID}, nil
}

func (s *CatalogService) GetAnimeIDs(ctx context.Context, _ *catalogv1.GetAnimeIDsRequest) (*catalogv1.GetAnimeIDsResponse, error) {
	rows, err := s.DB.Query(ctx, `SELECT id::text FROM anime ORDER BY updated_at DESC`)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()

	resp := &catalogv1.GetAnimeIDsResponse{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		resp.AnimeIds = append(resp.AnimeIds, id)
	}
	return resp, nil
}

func (s *CatalogService) GetAnimeByIDs(ctx context.Context, req *catalogv1.GetAnimeByIDsRequest) (*catalogv1.GetAnimeByIDsResponse, error) {
	ids := req.GetAnimeIds()
	if len(ids) == 0 {
		return &catalogv1.GetAnimeByIDsResponse{Anime: nil}, nil
	}

	q := `
SELECT id::text, title, title_english, title_japanese, image, description, genres, score, status, type, total_episodes
FROM anime
WHERE id::text = ANY($1)
`
	rows, err := s.DB.Query(ctx, q, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "db query")
	}
	defer rows.Close()

	resp := &catalogv1.GetAnimeByIDsResponse{}
	for rows.Next() {
		var (
			id, title, titleEnglish, titleJapanese, image, description, animeStatus, animeType string
			genresJSON                                                                         []byte
			score                                                                              float32
			totalEpisodes                                                                      int32
		)
		if err := rows.Scan(&id, &title, &titleEnglish, &titleJapanese, &image, &description, &genresJSON, &score, &animeStatus, &animeType, &totalEpisodes); err != nil {
			return nil, status.Error(codes.Internal, "db scan")
		}
		var genres []string
		_ = json.Unmarshal(genresJSON, &genres)
		resp.Anime = append(resp.Anime, &catalogv1.Anime{
			Id:            id,
			Title:         title,
			TitleEnglish:  titleEnglish,
			TitleJapanese: titleJapanese,
			Image:         image,
			Description:   description,
			Genres:        genres,
			Score:         score,
			Status:        animeStatus,
			Type:          animeType,
			TotalEpisodes: totalEpisodes,
		})
	}
	return resp, nil
}

func (s *CatalogService) AttachExternalAnimeID(ctx context.Context, req *catalogv1.AttachExternalAnimeIDRequest) (*catalogv1.AttachExternalAnimeIDResponse, error) {
	animeID, err := uuid.Parse(strings.TrimSpace(req.GetAnimeId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid anime_id")
	}
	provider := strings.TrimSpace(req.GetProvider())
	externalID := strings.TrimSpace(req.GetExternalId())
	if provider == "" || externalID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider and external_id are required")
	}

	q := `
INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id)
VALUES ($1,$2,$3)
ON CONFLICT (provider, provider_anime_id)
DO UPDATE SET anime_id = EXCLUDED.anime_id;
`
	if _, err := s.DB.Exec(ctx, q, provider, externalID, animeID); err != nil {
		return nil, status.Error(codes.Internal, "db")
	}
	return &catalogv1.AttachExternalAnimeIDResponse{}, nil
}

func (s *CatalogService) ResolveAnimeIDByExternalID(ctx context.Context, req *catalogv1.ResolveAnimeIDByExternalIDRequest) (*catalogv1.ResolveAnimeIDByExternalIDResponse, error) {
	provider := strings.TrimSpace(req.GetProvider())
	externalID := strings.TrimSpace(req.GetExternalId())
	if provider == "" || externalID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider and external_id are required")
	}

	var animeID uuid.UUID
	q := qSelectAnimeIDByExternal
	if err := s.DB.QueryRow(ctx, q, provider, externalID).Scan(&animeID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Error(codes.Internal, "db")
	}
	return &catalogv1.ResolveAnimeIDByExternalIDResponse{AnimeId: animeID.String()}, nil
}

func (s *CatalogService) UpsertHiAnimeEpisodes(ctx context.Context, req *catalogv1.UpsertHiAnimeEpisodesRequest) (*catalogv1.UpsertHiAnimeEpisodesResponse, error) {
	const provider = "hianime"
	animeID, err := uuid.Parse(strings.TrimSpace(req.GetAnimeId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid anime_id")
	}
	slug := strings.TrimSpace(req.GetHianimeSlug())
	if slug == "" {
		return nil, status.Error(codes.InvalidArgument, "hianime_slug is required")
	}

	now := time.Now().UTC()

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, "db begin")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ensure hianime slug mapping
	qMapAnime := `
INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id)
VALUES ($1,$2,$3)
ON CONFLICT (provider, provider_anime_id)
DO UPDATE SET anime_id = EXCLUDED.anime_id;
`
	if _, err := tx.Exec(ctx, qMapAnime, provider, slug, animeID); err != nil {
		return nil, status.Error(codes.Internal, "db")
	}

	// Собираем эпизоды в единый формат и делегируем в upsertEpisodes
	episodes := make([]episodeInput, 0, len(req.GetEpisodes()))
	for _, ep := range req.GetEpisodes() {
		if ep == nil {
			continue
		}
		episodes = append(episodes, episodeInput{
			providerEpisodeID: strings.TrimSpace(ep.GetProviderEpisodeId()),
			number:            ep.GetNumber(),
			title:             ep.GetTitle(),
			isFiller:          ep.GetIsFiller(),
			hasIsFiller:       true,
		})
	}
	episodeIDs, err := upsertEpisodes(ctx, tx, provider, animeID, episodes, now)
	if err != nil {
		return nil, err
	}

	if err := s.insertOutboxEvent(ctx, tx, catalogEventAnimeUpserted, map[string]any{"anime_id": animeID.String()}); err != nil {
		return nil, status.Error(codes.Internal, "db outbox")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "db commit")
	}
	return &catalogv1.UpsertHiAnimeEpisodesResponse{EpisodeIds: episodeIDs}, nil
}

func (s *CatalogService) UpsertJikanAnime(ctx context.Context, req *catalogv1.UpsertJikanAnimeRequest) (*catalogv1.UpsertJikanAnimeResponse, error) {
	anime := req.GetAnime()
	if anime == nil {
		return nil, status.Error(codes.InvalidArgument, "anime is required")
	}
	malID := anime.GetMalId()
	if malID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "mal_id is required")
	}

	provider := "mal"
	externalID := fmt.Sprintf("%d", malID)
	genresJSON, _ := json.Marshal(anime.GetGenres())
	now := time.Now().UTC()

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, "db begin")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var animeID uuid.UUID
	qFind := qSelectAnimeIDByExternal
	err = tx.QueryRow(ctx, qFind, provider, externalID).Scan(&animeID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.Internal, "db")
		}
		animeID = uuid.New()
		qIns := `
INSERT INTO anime (id, title, title_english, title_japanese, url, image, description, genres, sub_or_dub, type, status, other_name, total_episodes, score, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'unknown',$9,$10,'',$11,$12,$13,$14)
`
		_, err = tx.Exec(ctx, qIns,
			animeID,
			anime.GetTitle(),
			anime.GetTitleEnglish(),
			anime.GetTitleJapanese(),
			"",
			anime.GetImage(),
			anime.GetSynopsis(),
			genresJSON,
			anime.GetType(),
			anime.GetStatus(),
			anime.GetEpisodes(),
			anime.GetScore(),
			now, now,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "db: "+err.Error())
		}
		qMap := `INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id) VALUES ($1,$2,$3)`
		if _, err := tx.Exec(ctx, qMap, provider, externalID, animeID); err != nil {
			return nil, status.Error(codes.Internal, "db: "+err.Error())
		}
	} else {
		qUpd := `
UPDATE anime
SET title=$2, title_english=$3, title_japanese=$4, image=$5, description=$6, genres=$7, type=$8, status=$9, total_episodes=$10, score=$11, updated_at=$12
WHERE id=$1
`
		if _, err := tx.Exec(ctx, qUpd,
			animeID,
			anime.GetTitle(),
			anime.GetTitleEnglish(),
			anime.GetTitleJapanese(),
			anime.GetImage(),
			anime.GetSynopsis(),
			genresJSON,
			anime.GetType(),
			anime.GetStatus(),
			anime.GetEpisodes(),
			anime.GetScore(),
			now,
		); err != nil {
			return nil, status.Error(codes.Internal, "db")
		}
	}

	if err := s.insertOutboxEvent(ctx, tx, catalogEventAnimeUpserted, map[string]any{"anime_id": animeID.String()}); err != nil {
		return nil, status.Error(codes.Internal, "db outbox")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "db commit")
	}
	return &catalogv1.UpsertJikanAnimeResponse{AnimeId: animeID.String()}, nil
}

func (s *CatalogService) UpsertAnimeKaiAnime(ctx context.Context, req *catalogv1.UpsertAnimeKaiAnimeRequest) (*catalogv1.UpsertAnimeKaiAnimeResponse, error) {
	const provider = "animekai"

	anime := req.GetAnime()
	if anime == nil {
		return nil, status.Error(codes.InvalidArgument, "anime is required")
	}
	provAnimeID := strings.TrimSpace(anime.GetProviderAnimeId())
	if provAnimeID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_anime_id is required")
	}

	genresJSON, _ := json.Marshal(anime.GetGenres())
	now := time.Now().UTC()

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, "db begin")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1) resolve or create anime_id
	var animeID uuid.UUID
	qFindAnime := `SELECT anime_id FROM external_anime_ids WHERE provider=$1 AND provider_anime_id=$2`
	err = tx.QueryRow(ctx, qFindAnime, provider, provAnimeID).Scan(&animeID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.Internal, "db")
		}
		animeID = uuid.New()

		qInsertAnime := `
INSERT INTO anime (id, title, url, image, description, genres, sub_or_dub, type, status, other_name, total_episodes, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
`
		_, err = tx.Exec(ctx, qInsertAnime,
			animeID,
			anime.GetTitle(),
			anime.GetUrl(),
			anime.GetImage(),
			anime.GetDescription(),
			genresJSON,
			anime.GetSubOrDub(),
			anime.GetType(),
			anime.GetStatus(),
			anime.GetOtherName(),
			anime.GetTotalEpisodes(),
			now, now,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "db")
		}

		qInsertExt := `INSERT INTO external_anime_ids (provider, provider_anime_id, anime_id) VALUES ($1,$2,$3)`
		_, err = tx.Exec(ctx, qInsertExt, provider, provAnimeID, animeID)
		if err != nil {
			return nil, status.Error(codes.Internal, "db")
		}
	} else {
		// update existing anime
		qUpdateAnime := `
UPDATE anime
SET title=$2, url=$3, image=$4, description=$5, genres=$6, sub_or_dub=$7, type=$8, status=$9, other_name=$10, total_episodes=$11, updated_at=$12
WHERE id=$1
`
		_, err = tx.Exec(ctx, qUpdateAnime,
			animeID,
			anime.GetTitle(),
			anime.GetUrl(),
			anime.GetImage(),
			anime.GetDescription(),
			genresJSON,
			anime.GetSubOrDub(),
			anime.GetType(),
			anime.GetStatus(),
			anime.GetOtherName(),
			anime.GetTotalEpisodes(),
			now,
		)
		if err != nil {
			return nil, status.Error(codes.Internal, "db")
		}
	}

	// Собираем эпизоды в единый формат и делегируем в upsertEpisodes
	episodes := make([]episodeInput, 0, len(anime.GetEpisodes()))
	for _, ep := range anime.GetEpisodes() {
		if ep == nil {
			continue
		}
		episodes = append(episodes, episodeInput{
			providerEpisodeID: strings.TrimSpace(ep.GetProviderEpisodeId()),
			number:            ep.GetNumber(),
			title:             ep.GetTitle(),
			url:               ep.GetUrl(),
		})
	}
	episodeIDs, err := upsertEpisodes(ctx, tx, provider, animeID, episodes, now)
	if err != nil {
		return nil, err
	}

	if err := s.insertOutboxEvent(ctx, tx, catalogEventAnimeUpserted, map[string]any{"anime_id": animeID.String()}); err != nil {
		return nil, status.Error(codes.Internal, "db outbox")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, status.Error(codes.Internal, "db commit")
	}

	return &catalogv1.UpsertAnimeKaiAnimeResponse{AnimeId: animeID.String(), EpisodeIds: episodeIDs}, nil
}

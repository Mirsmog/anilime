package store

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresCommentStore persists comments in Postgres.
type PostgresCommentStore struct {
	pool *pgxpool.Pool
}

// NewPostgresCommentStore creates a store backed by Postgres.
func NewPostgresCommentStore(pool *pgxpool.Pool) *PostgresCommentStore {
	return &PostgresCommentStore{pool: pool}
}

func (s *PostgresCommentStore) Create(ctx context.Context, c Comment) (Comment, error) {
	const q = `INSERT INTO comments (anime_id, user_id, parent_id, body)
	           VALUES ($1, $2, $3, $4)
	           RETURNING id, anime_id, user_id, parent_id, body, score, created_at, updated_at, deleted_at`
	row := s.pool.QueryRow(ctx, q, c.AnimeID, c.UserID, c.ParentID, c.Body)
	var out Comment
	err := row.Scan(&out.ID, &out.AnimeID, &out.UserID, &out.ParentID,
		&out.Body, &out.Score, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	return out, err
}

func (s *PostgresCommentStore) GetThread(ctx context.Context, animeID, sort string, limit int, cursor string) ([]CommentTreeNode, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var roots []Comment
	var err error
	switch sort {
	case "top":
		roots, err = s.queryRootsTop(ctx, animeID, limit+1, cursor)
	default: // "new"
		roots, err = s.queryRootsNew(ctx, animeID, limit+1, cursor)
	}
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(roots) > limit {
		last := roots[limit-1]
		roots = roots[:limit]
		switch sort {
		case "top":
			nextCursor = encodeTopCursor(last.Score, last.CreatedAt, last.ID)
		default:
			nextCursor = encodeNewCursor(last.CreatedAt, last.ID)
		}
	}

	if len(roots) == 0 {
		return []CommentTreeNode{}, "", nil
	}

	rootIDs := make([]string, len(roots))
	for i, r := range roots {
		rootIDs[i] = r.ID
	}

	replies, err := s.queryReplies(ctx, rootIDs)
	if err != nil {
		return nil, "", err
	}

	replyMap := make(map[string][]Comment)
	for _, r := range replies {
		if r.ParentID != nil {
			replyMap[*r.ParentID] = append(replyMap[*r.ParentID], r)
		}
	}

	nodes := make([]CommentTreeNode, len(roots))
	for i, r := range roots {
		nodes[i] = CommentTreeNode{
			Comment: r,
			Replies: replyMap[r.ID],
		}
		if nodes[i].Replies == nil {
			nodes[i].Replies = []Comment{}
		}
	}
	return nodes, nextCursor, nil
}

func (s *PostgresCommentStore) queryRootsNew(ctx context.Context, animeID string, limit int, cursor string) ([]Comment, error) {
	var q string
	var args []any

	if cursor == "" {
		q = `SELECT id, anime_id, user_id, parent_id, body, score, created_at, updated_at, deleted_at
		     FROM comments
		     WHERE anime_id = $1 AND parent_id IS NULL
		     ORDER BY created_at DESC, id DESC
		     LIMIT $2`
		args = []any{animeID, limit}
	} else {
		cursorTime, cursorID, err := decodeNewCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		q = `SELECT id, anime_id, user_id, parent_id, body, score, created_at, updated_at, deleted_at
		     FROM comments
		     WHERE anime_id = $1 AND parent_id IS NULL
		       AND (created_at, id) < ($3, $4)
		     ORDER BY created_at DESC, id DESC
		     LIMIT $2`
		args = []any{animeID, limit, cursorTime, cursorID}
	}
	return s.scanComments(ctx, q, args...)
}

func (s *PostgresCommentStore) queryRootsTop(ctx context.Context, animeID string, limit int, cursor string) ([]Comment, error) {
	var q string
	var args []any

	if cursor == "" {
		q = `SELECT id, anime_id, user_id, parent_id, body, score, created_at, updated_at, deleted_at
		     FROM comments
		     WHERE anime_id = $1 AND parent_id IS NULL
		     ORDER BY score DESC, created_at DESC, id DESC
		     LIMIT $2`
		args = []any{animeID, limit}
	} else {
		cursorScore, cursorTime, cursorID, err := decodeTopCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		q = `SELECT id, anime_id, user_id, parent_id, body, score, created_at, updated_at, deleted_at
		     FROM comments
		     WHERE anime_id = $1 AND parent_id IS NULL
		       AND (score, created_at, id) < ($3, $4, $5)
		     ORDER BY score DESC, created_at DESC, id DESC
		     LIMIT $2`
		args = []any{animeID, limit, cursorScore, cursorTime, cursorID}
	}
	return s.scanComments(ctx, q, args...)
}

func (s *PostgresCommentStore) queryReplies(ctx context.Context, parentIDs []string) ([]Comment, error) {
	q := `SELECT id, anime_id, user_id, parent_id, body, score, created_at, updated_at, deleted_at
	      FROM comments
	      WHERE parent_id = ANY($1)
	      ORDER BY created_at ASC`
	return s.scanComments(ctx, q, parentIDs)
}

func (s *PostgresCommentStore) scanComments(ctx context.Context, q string, args ...any) ([]Comment, error) {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.AnimeID, &c.UserID, &c.ParentID,
			&c.Body, &c.Score, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PostgresCommentStore) UpdateBody(ctx context.Context, commentID, userID, body string) error {
	const q = `UPDATE comments SET body = $1, updated_at = now()
	           WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL`
	tag, err := s.pool.Exec(ctx, q, body, commentID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFoundOrForbidden
	}
	return nil
}

func (s *PostgresCommentStore) SoftDelete(ctx context.Context, commentID, userID string) error {
	const q = `UPDATE comments SET body = '[deleted]', deleted_at = now()
	           WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	tag, err := s.pool.Exec(ctx, q, commentID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFoundOrForbidden
	}
	return nil
}

func (s *PostgresCommentStore) Vote(ctx context.Context, commentID, userID string, vote int16) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check comment exists
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)`, commentID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFoundOrForbidden
	}

	// Get old vote if any
	var oldVote int16
	err = tx.QueryRow(ctx,
		`SELECT vote FROM comment_votes WHERE comment_id = $1 AND user_id = $2`,
		commentID, userID).Scan(&oldVote)

	var delta int16
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// New vote
		delta = vote
		_, err = tx.Exec(ctx,
			`INSERT INTO comment_votes (comment_id, user_id, vote) VALUES ($1, $2, $3)`,
			commentID, userID, vote)
	case err != nil:
		return err
	default:
		// Update existing vote
		delta = vote - oldVote
		_, err = tx.Exec(ctx,
			`UPDATE comment_votes SET vote = $1 WHERE comment_id = $2 AND user_id = $3`,
			vote, commentID, userID)
	}
	if err != nil {
		return err
	}

	if delta != 0 {
		_, err = tx.Exec(ctx,
			`UPDATE comments SET score = score + $1 WHERE id = $2`,
			delta, commentID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Sentinel errors
var ErrNotFoundOrForbidden = errors.New("comment not found or not owned by user")

// Cursor encoding helpers

func encodeNewCursor(t time.Time, id string) string {
	raw := fmt.Sprintf("%d|%s", t.UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

func decodeNewCursor(c string) (time.Time, string, error) {
	raw, err := base64.URLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("malformed cursor")
	}
	var nanos int64
	if _, err := fmt.Sscanf(parts[0], "%d", &nanos); err != nil {
		return time.Time{}, "", err
	}
	return time.Unix(0, nanos), parts[1], nil
}

func encodeTopCursor(score int, t time.Time, id string) string {
	raw := fmt.Sprintf("%d|%d|%s", score, t.UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

func decodeTopCursor(c string) (int, time.Time, string, error) {
	raw, err := base64.URLEncoding.DecodeString(c)
	if err != nil {
		return 0, time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "|", 3)
	if len(parts) != 3 {
		return 0, time.Time{}, "", errors.New("malformed cursor")
	}
	var score int
	if _, err := fmt.Sscanf(parts[0], "%d", &score); err != nil {
		return 0, time.Time{}, "", err
	}
	var nanos int64
	if _, err := fmt.Sscanf(parts[1], "%d", &nanos); err != nil {
		return 0, time.Time{}, "", err
	}
	return score, time.Unix(0, nanos), parts[2], nil
}

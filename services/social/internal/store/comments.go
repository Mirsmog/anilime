package store

import (
	"context"
	"time"
)

// Comment represents a single comment row.
type Comment struct {
	ID        string     `json:"id"`
	AnimeID   string     `json:"anime_id"`
	UserID    string     `json:"user_id"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Body      string     `json:"body"`
	Score     int        `json:"score"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// CommentTreeNode is a root comment with its direct replies.
type CommentTreeNode struct {
	Comment Comment   `json:"comment"`
	Replies []Comment `json:"replies"`
}

// CommentStore defines the contract for comment persistence.
type CommentStore interface {
	Create(ctx context.Context, c Comment) (Comment, error)
	GetThread(ctx context.Context, animeID, sort string, limit int, cursor string) ([]CommentTreeNode, string, error)
	UpdateBody(ctx context.Context, commentID, userID, body string) error
	SoftDelete(ctx context.Context, commentID, userID string) error
	Vote(ctx context.Context, commentID, userID string, vote int16) error
}

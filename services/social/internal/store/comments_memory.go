package store

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryCommentStore is a development-only in-memory implementation.
type InMemoryCommentStore struct {
	mu       sync.RWMutex
	comments map[string]Comment          // id -> comment
	votes    map[string]map[string]int16 // commentID -> userID -> vote
}

func NewInMemoryCommentStore() *InMemoryCommentStore {
	return &InMemoryCommentStore{
		comments: make(map[string]Comment),
		votes:    make(map[string]map[string]int16),
	}
}

func (s *InMemoryCommentStore) Create(_ context.Context, c Comment) (Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c.ID = uuid.New().String()
	c.CreatedAt = time.Now().UTC()
	c.Score = 0
	s.comments[c.ID] = c
	return c, nil
}

func (s *InMemoryCommentStore) GetThread(_ context.Context, animeID, sortBy string, limit int, cursor string) ([]CommentTreeNode, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var roots []Comment
	for _, c := range s.comments {
		if c.AnimeID == animeID && c.ParentID == nil {
			roots = append(roots, c)
		}
	}

	switch sortBy {
	case SortTop:
		sort.Slice(roots, func(i, j int) bool {
			if roots[i].Score != roots[j].Score {
				return roots[i].Score > roots[j].Score
			}
			if !roots[i].CreatedAt.Equal(roots[j].CreatedAt) {
				return roots[i].CreatedAt.After(roots[j].CreatedAt)
			}
			return roots[i].ID > roots[j].ID
		})
	default:
		sort.Slice(roots, func(i, j int) bool {
			if !roots[i].CreatedAt.Equal(roots[j].CreatedAt) {
				return roots[i].CreatedAt.After(roots[j].CreatedAt)
			}
			return roots[i].ID > roots[j].ID
		})
	}

	// Simple cursor: skip past cursor position (development only, not production-grade)
	startIdx := 0
	if cursor != "" {
		for i, r := range roots {
			if r.ID == cursor {
				startIdx = i + 1
				break
			}
		}
	}

	if startIdx >= len(roots) {
		return []CommentTreeNode{}, "", nil
	}
	roots = roots[startIdx:]

	var nextCursor string
	if len(roots) > limit {
		nextCursor = roots[limit-1].ID
		roots = roots[:limit]
	}

	nodes := make([]CommentTreeNode, len(roots))
	for i, root := range roots {
		var replies []Comment
		for _, c := range s.comments {
			if c.ParentID != nil && *c.ParentID == root.ID {
				replies = append(replies, c)
			}
		}
		sort.Slice(replies, func(a, b int) bool {
			return replies[a].CreatedAt.Before(replies[b].CreatedAt)
		})
		if replies == nil {
			replies = []Comment{}
		}
		nodes[i] = CommentTreeNode{Comment: root, Replies: replies}
	}
	return nodes, nextCursor, nil
}

func (s *InMemoryCommentStore) UpdateBody(_ context.Context, commentID, userID, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.comments[commentID]
	if !ok || c.UserID != userID || c.DeletedAt != nil {
		return ErrNotFoundOrForbidden
	}
	c.Body = body
	now := time.Now().UTC()
	c.UpdatedAt = &now
	s.comments[commentID] = c
	return nil
}

func (s *InMemoryCommentStore) SoftDelete(_ context.Context, commentID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.comments[commentID]
	if !ok || c.UserID != userID || c.DeletedAt != nil {
		return ErrNotFoundOrForbidden
	}
	c.Body = "[deleted]"
	now := time.Now().UTC()
	c.DeletedAt = &now
	s.comments[commentID] = c
	return nil
}

func (s *InMemoryCommentStore) Vote(_ context.Context, commentID, userID string, vote int16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.comments[commentID]
	if !ok {
		return ErrNotFoundOrForbidden
	}

	if vote != 1 && vote != -1 {
		return errors.New("vote must be 1 or -1")
	}

	if s.votes[commentID] == nil {
		s.votes[commentID] = make(map[string]int16)
	}

	oldVote := s.votes[commentID][userID]
	delta := vote - oldVote
	s.votes[commentID][userID] = vote
	c.Score += int(delta)
	s.comments[commentID] = c
	return nil
}

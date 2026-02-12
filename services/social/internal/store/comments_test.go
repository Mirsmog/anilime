package store

import (
	"context"
	"testing"
)

func TestInMemoryCommentStore_Create(t *testing.T) {
	s := NewInMemoryCommentStore()
	ctx := context.Background()

	c, err := s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-a", Body: "hello"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if c.Body != "hello" {
		t.Fatalf("expected body 'hello', got %q", c.Body)
	}
	if c.Score != 0 {
		t.Fatalf("expected score 0, got %d", c.Score)
	}
}

func TestInMemoryCommentStore_GetThread(t *testing.T) {
	s := NewInMemoryCommentStore()
	ctx := context.Background()

	root1, _ := s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-a", Body: "root 1"})
	root2, _ := s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-b", Body: "root 2"})

	// Reply to root1
	pid := root1.ID
	_, _ = s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-c", ParentID: &pid, Body: "reply 1"})

	nodes, _, err := s.GetThread(ctx, "anime-1", "new", 50, "")
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 root nodes, got %d", len(nodes))
	}
	// "new" sort: most recent first
	if nodes[0].Comment.ID != root2.ID {
		t.Fatalf("expected root2 first (newest), got %s", nodes[0].Comment.ID)
	}
	if len(nodes[1].Replies) != 1 {
		t.Fatalf("expected 1 reply to root1, got %d", len(nodes[1].Replies))
	}
}

func TestInMemoryCommentStore_UpdateBody_AuthorOnly(t *testing.T) {
	s := NewInMemoryCommentStore()
	ctx := context.Background()

	c, _ := s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-a", Body: "original"})

	// Non-author cannot edit
	err := s.UpdateBody(ctx, c.ID, "user-b", "hacked")
	if err != ErrNotFoundOrForbidden {
		t.Fatalf("expected ErrNotFoundOrForbidden for non-author, got %v", err)
	}

	// Author can edit
	err = s.UpdateBody(ctx, c.ID, "user-a", "updated")
	if err != nil {
		t.Fatalf("author update: %v", err)
	}
}

func TestInMemoryCommentStore_SoftDelete(t *testing.T) {
	s := NewInMemoryCommentStore()
	ctx := context.Background()

	c, _ := s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-a", Body: "will delete"})

	// Non-author cannot delete
	err := s.SoftDelete(ctx, c.ID, "user-b")
	if err != ErrNotFoundOrForbidden {
		t.Fatalf("expected ErrNotFoundOrForbidden for non-author, got %v", err)
	}

	// Author deletes
	err = s.SoftDelete(ctx, c.ID, "user-a")
	if err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// Verify body replaced
	nodes, _, _ := s.GetThread(ctx, "anime-1", "new", 50, "")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Comment.Body != "[deleted]" {
		t.Fatalf("expected body '[deleted]', got %q", nodes[0].Comment.Body)
	}
	if nodes[0].Comment.DeletedAt == nil {
		t.Fatal("expected deleted_at to be set")
	}

	// Cannot delete again
	err = s.SoftDelete(ctx, c.ID, "user-a")
	if err != ErrNotFoundOrForbidden {
		t.Fatalf("expected ErrNotFoundOrForbidden for double delete, got %v", err)
	}
}

func TestInMemoryCommentStore_Vote_Idempotent(t *testing.T) {
	s := NewInMemoryCommentStore()
	ctx := context.Background()

	c, _ := s.Create(ctx, Comment{AnimeID: "anime-1", UserID: "user-a", Body: "voteable"})

	// First upvote
	if err := s.Vote(ctx, c.ID, "user-b", 1); err != nil {
		t.Fatalf("vote: %v", err)
	}
	nodes, _, _ := s.GetThread(ctx, "anime-1", "new", 50, "")
	if nodes[0].Comment.Score != 1 {
		t.Fatalf("expected score 1, got %d", nodes[0].Comment.Score)
	}

	// Same upvote again (idempotent)
	if err := s.Vote(ctx, c.ID, "user-b", 1); err != nil {
		t.Fatalf("vote idempotent: %v", err)
	}
	nodes, _, _ = s.GetThread(ctx, "anime-1", "new", 50, "")
	if nodes[0].Comment.Score != 1 {
		t.Fatalf("expected score 1 after idempotent vote, got %d", nodes[0].Comment.Score)
	}

	// Switch to downvote
	if err := s.Vote(ctx, c.ID, "user-b", -1); err != nil {
		t.Fatalf("vote switch: %v", err)
	}
	nodes, _, _ = s.GetThread(ctx, "anime-1", "new", 50, "")
	if nodes[0].Comment.Score != -1 {
		t.Fatalf("expected score -1 after switch, got %d", nodes[0].Comment.Score)
	}

	// Vote on non-existent comment
	err := s.Vote(ctx, "non-existent", "user-b", 1)
	if err != ErrNotFoundOrForbidden {
		t.Fatalf("expected ErrNotFoundOrForbidden for non-existent, got %v", err)
	}
}

func TestCommentStoreInterface(t *testing.T) {
	var _ CommentStore = (*InMemoryCommentStore)(nil)
	var _ CommentStore = (*PostgresCommentStore)(nil)
}

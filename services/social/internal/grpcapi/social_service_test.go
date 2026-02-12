package grpcapi

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
	"github.com/example/anime-platform/services/social/internal/store"
)

func ctxWithUser(userID string) context.Context {
	md := metadata.New(map[string]string{"user_id": userID})
	return metadata.NewIncomingContext(context.Background(), md)
}

func ctxNoUser() context.Context {
	return context.Background()
}

func newService() *SocialService {
	return &SocialService{Comments: store.NewInMemoryCommentStore()}
}

func TestCreateComment_Success(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	resp, err := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{
		AnimeId: "anime-1",
		Body:    "Great episode!",
	})
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	c := resp.GetComment()
	if c.GetId() == "" {
		t.Fatal("expected non-empty id")
	}
	if c.GetAnimeId() != "anime-1" {
		t.Fatalf("expected anime_id 'anime-1', got %q", c.GetAnimeId())
	}
	if c.GetUserId() != "user-a" {
		t.Fatalf("expected user_id 'user-a', got %q", c.GetUserId())
	}
	if c.GetBody() != "Great episode!" {
		t.Fatalf("expected body 'Great episode!', got %q", c.GetBody())
	}
	if c.GetScore() != 0 {
		t.Fatalf("expected score 0, got %d", c.GetScore())
	}
}

func TestCreateComment_WithParent(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	parent, err := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{
		AnimeId: "anime-1",
		Body:    "root",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	pid := parent.GetComment().GetId()
	resp, err := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{
		AnimeId:  "anime-1",
		Body:     "reply",
		ParentId: &pid,
	})
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if resp.GetComment().ParentId == nil || *resp.GetComment().ParentId != pid {
		t.Fatal("expected parent_id to match")
	}
}

func TestCreateComment_Unauthenticated(t *testing.T) {
	svc := newService()

	_, err := svc.CreateComment(ctxNoUser(), &socialv1.CreateCommentRequest{
		AnimeId: "anime-1",
		Body:    "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestCreateComment_EmptyBody(t *testing.T) {
	svc := newService()

	_, err := svc.CreateComment(ctxWithUser("user-a"), &socialv1.CreateCommentRequest{
		AnimeId: "anime-1",
		Body:    "",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestCreateComment_MissingAnimeID(t *testing.T) {
	svc := newService()

	_, err := svc.CreateComment(ctxWithUser("user-a"), &socialv1.CreateCommentRequest{
		Body: "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestListComments(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	_, _ = svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "first"})
	_, _ = svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "second"})

	resp, err := svc.ListComments(context.Background(), &socialv1.ListCommentsRequest{
		AnimeId: "anime-1",
		Sort:    "new",
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(resp.GetComments()) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(resp.GetComments()))
	}
}

func TestListComments_MissingAnimeID(t *testing.T) {
	svc := newService()

	_, err := svc.ListComments(context.Background(), &socialv1.ListCommentsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestVoteComment_Success(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	created, _ := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "voteable"})
	cid := created.GetComment().GetId()

	voterCtx := ctxWithUser("user-b")
	_, err := svc.VoteComment(voterCtx, &socialv1.VoteCommentRequest{CommentId: cid, Vote: 1})
	if err != nil {
		t.Fatalf("VoteComment: %v", err)
	}

	// Verify score
	resp, _ := svc.ListComments(context.Background(), &socialv1.ListCommentsRequest{AnimeId: "anime-1"})
	if resp.GetComments()[0].GetComment().GetScore() != 1 {
		t.Fatalf("expected score 1, got %d", resp.GetComments()[0].GetComment().GetScore())
	}
}

func TestVoteComment_InvalidVote(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	created, _ := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "voteable"})
	cid := created.GetComment().GetId()

	_, err := svc.VoteComment(ctxWithUser("user-b"), &socialv1.VoteCommentRequest{CommentId: cid, Vote: 2})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestVoteComment_NotFound(t *testing.T) {
	svc := newService()

	_, err := svc.VoteComment(ctxWithUser("user-a"), &socialv1.VoteCommentRequest{CommentId: "non-existent", Vote: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestUpdateComment_AuthorOnly(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	created, _ := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "original"})
	cid := created.GetComment().GetId()

	// Non-author: permission denied
	_, err := svc.UpdateComment(ctxWithUser("user-b"), &socialv1.UpdateCommentRequest{CommentId: cid, Body: "hacked"})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}

	// Author: success
	_, err = svc.UpdateComment(ctx, &socialv1.UpdateCommentRequest{CommentId: cid, Body: "updated"})
	if err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
}

func TestUpdateComment_EmptyBody(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	created, _ := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "original"})
	cid := created.GetComment().GetId()

	_, err := svc.UpdateComment(ctx, &socialv1.UpdateCommentRequest{CommentId: cid, Body: ""})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestDeleteComment_AuthorOnly(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	created, _ := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "will delete"})
	cid := created.GetComment().GetId()

	// Non-author: permission denied
	_, err := svc.DeleteComment(ctxWithUser("user-b"), &socialv1.DeleteCommentRequest{CommentId: cid})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}

	// Author: success
	_, err = svc.DeleteComment(ctx, &socialv1.DeleteCommentRequest{CommentId: cid})
	if err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}

	// Verify body replaced
	resp, _ := svc.ListComments(context.Background(), &socialv1.ListCommentsRequest{AnimeId: "anime-1"})
	if len(resp.GetComments()) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(resp.GetComments()))
	}
	if resp.GetComments()[0].GetComment().GetBody() != "[deleted]" {
		t.Fatalf("expected body '[deleted]', got %q", resp.GetComments()[0].GetComment().GetBody())
	}
}

func TestDeleteComment_DoubleDelete(t *testing.T) {
	svc := newService()
	ctx := ctxWithUser("user-a")

	created, _ := svc.CreateComment(ctx, &socialv1.CreateCommentRequest{AnimeId: "anime-1", Body: "will delete"})
	cid := created.GetComment().GetId()

	_, _ = svc.DeleteComment(ctx, &socialv1.DeleteCommentRequest{CommentId: cid})

	// Second delete: permission denied
	_, err := svc.DeleteComment(ctx, &socialv1.DeleteCommentRequest{CommentId: cid})
	if err == nil {
		t.Fatal("expected error on double delete")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}

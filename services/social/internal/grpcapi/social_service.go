package grpcapi

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	socialv1 "github.com/example/anime-platform/gen/social/v1"
	"github.com/example/anime-platform/services/social/internal/store"
)

// SocialService implements the SocialServiceServer gRPC interface.
type SocialService struct {
	socialv1.UnimplementedSocialServiceServer
	Comments store.CommentStore
}

func userIDFromMD(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get("user_id")
	if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
		return "", status.Error(codes.Unauthenticated, "missing user_id in metadata")
	}
	return strings.TrimSpace(vals[0]), nil
}

func commentToProto(c store.Comment) *socialv1.Comment {
	pb := &socialv1.Comment{
		Id:        c.ID,
		AnimeId:   c.AnimeID,
		UserId:    c.UserID,
		Body:      c.Body,
		Score:     int32(c.Score),
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if c.ParentID != nil {
		pb.ParentId = c.ParentID
	}
	if c.UpdatedAt != nil {
		pb.UpdatedAt = timestamppb.New(*c.UpdatedAt)
	}
	if c.DeletedAt != nil {
		pb.DeletedAt = timestamppb.New(*c.DeletedAt)
	}
	return pb
}

func (s *SocialService) CreateComment(ctx context.Context, req *socialv1.CreateCommentRequest) (*socialv1.CreateCommentResponse, error) {
	userID, err := userIDFromMD(ctx)
	if err != nil {
		return nil, err
	}

	animeID := strings.TrimSpace(req.GetAnimeId())
	if animeID == "" {
		return nil, status.Error(codes.InvalidArgument, "anime_id is required")
	}
	body := strings.TrimSpace(req.GetBody())
	if body == "" {
		return nil, status.Error(codes.InvalidArgument, "body must not be empty")
	}

	c := store.Comment{
		AnimeID: animeID,
		UserID:  userID,
		Body:    req.GetBody(),
	}
	if req.ParentId != nil {
		pid := strings.TrimSpace(*req.ParentId)
		c.ParentID = &pid
	}

	created, err := s.Comments.Create(ctx, c)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create comment")
	}
	return &socialv1.CreateCommentResponse{Comment: commentToProto(created)}, nil
}

func (s *SocialService) ListComments(ctx context.Context, req *socialv1.ListCommentsRequest) (*socialv1.ListCommentsResponse, error) {
	animeID := strings.TrimSpace(req.GetAnimeId())
	if animeID == "" {
		return nil, status.Error(codes.InvalidArgument, "anime_id is required")
	}

	sortParam := strings.ToLower(strings.TrimSpace(req.GetSort()))
	if sortParam != store.SortTop {
		sortParam = "new"
	}

	limit := int(req.GetLimit())
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	cursor := strings.TrimSpace(req.GetCursor())

	nodes, nextCursor, err := s.Comments.GetThread(ctx, animeID, sortParam, limit, cursor)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list comments")
	}

	resp := &socialv1.ListCommentsResponse{NextCursor: nextCursor}
	for _, n := range nodes {
		pbNode := &socialv1.CommentTreeNode{Comment: commentToProto(n.Comment)}
		for _, r := range n.Replies {
			pbNode.Replies = append(pbNode.Replies, commentToProto(r))
		}
		resp.Comments = append(resp.Comments, pbNode)
	}
	return resp, nil
}

func (s *SocialService) VoteComment(ctx context.Context, req *socialv1.VoteCommentRequest) (*socialv1.VoteCommentResponse, error) {
	userID, err := userIDFromMD(ctx)
	if err != nil {
		return nil, err
	}

	commentID := strings.TrimSpace(req.GetCommentId())
	if commentID == "" {
		return nil, status.Error(codes.InvalidArgument, "comment_id is required")
	}

	vote := int16(req.GetVote())
	if vote != 1 && vote != -1 {
		return nil, status.Error(codes.InvalidArgument, "vote must be 1 or -1")
	}

	if err := s.Comments.Vote(ctx, commentID, userID, vote); err != nil {
		if err == store.ErrNotFoundOrForbidden {
			return nil, status.Error(codes.NotFound, "comment not found")
		}
		return nil, status.Error(codes.Internal, "failed to vote")
	}
	return &socialv1.VoteCommentResponse{}, nil
}

func (s *SocialService) UpdateComment(ctx context.Context, req *socialv1.UpdateCommentRequest) (*socialv1.UpdateCommentResponse, error) {
	userID, err := userIDFromMD(ctx)
	if err != nil {
		return nil, err
	}

	commentID := strings.TrimSpace(req.GetCommentId())
	if commentID == "" {
		return nil, status.Error(codes.InvalidArgument, "comment_id is required")
	}

	body := strings.TrimSpace(req.GetBody())
	if body == "" {
		return nil, status.Error(codes.InvalidArgument, "body must not be empty")
	}

	if err := s.Comments.UpdateBody(ctx, commentID, userID, req.GetBody()); err != nil {
		if err == store.ErrNotFoundOrForbidden {
			return nil, status.Error(codes.PermissionDenied, "not found or not the author")
		}
		return nil, status.Error(codes.Internal, "failed to update comment")
	}
	return &socialv1.UpdateCommentResponse{}, nil
}

func (s *SocialService) DeleteComment(ctx context.Context, req *socialv1.DeleteCommentRequest) (*socialv1.DeleteCommentResponse, error) {
	userID, err := userIDFromMD(ctx)
	if err != nil {
		return nil, err
	}

	commentID := strings.TrimSpace(req.GetCommentId())
	if commentID == "" {
		return nil, status.Error(codes.InvalidArgument, "comment_id is required")
	}

	if err := s.Comments.SoftDelete(ctx, commentID, userID); err != nil {
		if err == store.ErrNotFoundOrForbidden {
			return nil, status.Error(codes.PermissionDenied, "not found or not the author")
		}
		return nil, status.Error(codes.Internal, "failed to delete comment")
	}
	return &socialv1.DeleteCommentResponse{}, nil
}

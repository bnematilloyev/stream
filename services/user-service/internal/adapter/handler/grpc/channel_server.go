package grpc

import (
	"context"

	"github.com/google/uuid"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/usecase"
)

type ChannelServer struct {
	userv1.UnimplementedChannelServiceServer
	uc *usecase.ChannelUseCase
}

func NewChannelServer(uc *usecase.ChannelUseCase) *ChannelServer {
	return &ChannelServer{uc: uc}
}

func (s *ChannelServer) CreateChannel(ctx context.Context, req *userv1.CreateChannelRequest) (*userv1.Channel, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	ch, err := s.uc.Create(ctx, userID, req.GetSlug(), req.GetTitle(), req.GetDescription(), req.GetCategoryId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toChannel(ch), nil
}

func (s *ChannelServer) GetChannel(ctx context.Context, req *userv1.GetChannelRequest) (*userv1.Channel, error) {
	ch, err := s.uc.GetBySlug(ctx, req.GetSlug())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toChannel(ch), nil
}

func (s *ChannelServer) GetMyChannel(ctx context.Context, req *userv1.GetMyChannelRequest) (*userv1.Channel, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	ch, err := s.uc.GetMyChannel(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toChannel(ch), nil
}

func (s *ChannelServer) UpdateChannel(ctx context.Context, req *userv1.UpdateChannelRequest) (*userv1.Channel, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	ch, err := s.uc.Update(ctx, userID, req.GetSlug(), req.Title, req.Description, req.BannerUrl, req.AvatarUrl, req.CategoryId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toChannel(ch), nil
}

func (s *ChannelServer) Follow(ctx context.Context, req *userv1.FollowRequest) (*userv1.FollowResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	count, err := s.uc.Follow(ctx, userID, req.GetChannelSlug())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.FollowResponse{Success: true, FollowerCount: int32(count)}, nil
}

func (s *ChannelServer) Unfollow(ctx context.Context, req *userv1.UnfollowRequest) (*userv1.UnfollowResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	count, err := s.uc.Unfollow(ctx, userID, req.GetChannelSlug())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.UnfollowResponse{Success: true, FollowerCount: int32(count)}, nil
}

func (s *ChannelServer) ListFollowers(ctx context.Context, req *userv1.ListFollowersRequest) (*userv1.ListFollowersResponse, error) {
	list, meta, err := s.uc.ListFollowers(ctx, req.GetChannelSlug(), int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*userv1.Follower, 0, len(list))
	for _, f := range list {
		out = append(out, &userv1.Follower{
			UserId:          f.UserID.String(),
			Username:        f.Username,
			DisplayName:     f.DisplayName,
			FollowedAtUnix:  f.FollowedAt.Unix(),
		})
	}
	return &userv1.ListFollowersResponse{
		Followers: out,
		Total:     int32(meta.Total),
		Page:      int32(meta.Page),
		Limit:     int32(meta.Limit),
	}, nil
}

func (s *ChannelServer) IsFollowing(ctx context.Context, req *userv1.IsFollowingRequest) (*userv1.IsFollowingResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	following, err := s.uc.IsFollowing(ctx, userID, req.GetChannelSlug())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.IsFollowingResponse{Following: following}, nil
}

func (s *ChannelServer) GetIngestKey(ctx context.Context, req *userv1.GetIngestKeyRequest) (*userv1.IngestKeyResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	key, err := s.uc.GetIngestKey(ctx, userID, req.GetChannelSlug())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toIngestKey(key), nil
}

func (s *ChannelServer) RotateIngestKey(ctx context.Context, req *userv1.RotateIngestKeyRequest) (*userv1.IngestKeyResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	key, err := s.uc.RotateIngestKey(ctx, userID, req.GetChannelSlug())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toIngestKey(key), nil
}

func toChannel(ch *domain.Channel) *userv1.Channel {
	out := &userv1.Channel{
		Id:            ch.ID.String(),
		UserId:        ch.UserID.String(),
		Slug:          ch.Slug,
		Title:         ch.Title,
		IsVerified:    ch.IsVerified,
		IsLive:        ch.IsLive,
		FollowerCount: int32(ch.FollowerCount),
		CreatedAtUnix: ch.CreatedAt.Unix(),
		UpdatedAtUnix: ch.UpdatedAt.Unix(),
	}
	if ch.Description != nil {
		out.Description = *ch.Description
	}
	if ch.BannerURL != nil {
		out.BannerUrl = *ch.BannerURL
	}
	if ch.AvatarURL != nil {
		out.AvatarUrl = *ch.AvatarURL
	}
	if ch.CategoryID != nil {
		out.CategoryId = ch.CategoryID.String()
	}
	if ch.CategorySlug != nil {
		out.CategorySlug = *ch.CategorySlug
	}
	return out
}

func toIngestKey(k *usecase.IngestKeyResult) *userv1.IngestKeyResponse {
	return &userv1.IngestKeyResponse{
		StreamKey: k.StreamKey,
		RtmpUrl:   k.RTMPURL,
		SrtUrl:    k.SRTURL,
		KeyPrefix: k.KeyPrefix,
	}
}

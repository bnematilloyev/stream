package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
)

type ProfileRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Profile, error)
	GetByUsername(ctx context.Context, username string) (*Profile, error)
	Update(ctx context.Context, id uuid.UUID, displayName, avatarURL *string) (*Profile, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) error
	GetBySlug(ctx context.Context, slug string) (*Channel, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Channel, error)
	GetByMarketplaceSellerID(ctx context.Context, sellerID int64) (*Channel, error)
	Update(ctx context.Context, slug string, userID uuid.UUID, title, description, bannerURL, avatarURL *string, categoryID *uuid.UUID) (*Channel, error)
	SetLive(ctx context.Context, channelID uuid.UUID, live bool) error
	List(ctx context.Context, search string, marketplaceOnly bool, page, limit int) ([]Channel, int, error)
	SetVerified(ctx context.Context, slug string, verified bool) (*Channel, error)
}

type FollowerRepository interface {
	Follow(ctx context.Context, followerID, channelID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, channelID uuid.UUID) error
	IsFollowing(ctx context.Context, followerID, channelID uuid.UUID) (bool, error)
	List(ctx context.Context, channelID uuid.UUID, p pagination.Params) ([]Follower, int, error)
	IncrementFollowerCount(ctx context.Context, channelID uuid.UUID, delta int) (int, error)
}

type StreamKeyRepository interface {
	Create(ctx context.Context, key *StreamKey) error
	DeactivateByChannel(ctx context.Context, channelID uuid.UUID) error
	GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*StreamKey, error)
	GetByLookup(ctx context.Context, lookup string) (*StreamKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

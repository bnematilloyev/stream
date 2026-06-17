package domain

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID            uuid.UUID
	Email         string
	Username      string
	DisplayName   string
	AvatarURL     *string
	Role          string
	EmailVerified bool
	CreatedAt     time.Time
}

type Channel struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	Slug                string
	Title               string
	Description         *string
	BannerURL           *string
	AvatarURL           *string
	CategoryID          *uuid.UUID
	CategorySlug        *string
	IsVerified          bool
	IsLive              bool
	FollowerCount       int
	MarketplaceSellerID *int64
	MarketplaceShopID   *int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Follower struct {
	UserID      uuid.UUID
	Username    string
	DisplayName string
	FollowedAt  time.Time
}

type StreamKey struct {
	ID         uuid.UUID
	ChannelID  uuid.UUID
	KeyLookup  string
	KeyPrefix  string
	Label      string
	IsActive   bool
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
	"github.com/sahiy/sahiy-stream/pkg/slug"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
)

type IngestKeyResult struct {
	StreamKey string
	RTMPURL   string
	SRTURL    string
	KeyPrefix string
}

type ChannelUseCase struct {
	channels   domain.ChannelRepository
	followers  domain.FollowerRepository
	streamKeys domain.StreamKeyRepository
	rtmpURL    string
	srtURL     string
}

func NewChannelUseCase(
	channels domain.ChannelRepository,
	followers domain.FollowerRepository,
	streamKeys domain.StreamKeyRepository,
	rtmpURL, srtURL string,
) *ChannelUseCase {
	return &ChannelUseCase{
		channels:   channels,
		followers:  followers,
		streamKeys: streamKeys,
		rtmpURL:    rtmpURL,
		srtURL:     srtURL,
	}
}

func (uc *ChannelUseCase) Create(ctx context.Context, userID uuid.UUID, channelSlug, title, description, categoryID string) (*domain.Channel, error) {
	channelSlug = slug.Normalize(channelSlug)
	if !slug.Validate(channelSlug) {
		return nil, apperrors.Validation("invalid channel slug", map[string]any{"slug": "3-50 chars, lowercase alphanumeric, _ -"})
	}
	title = strings.TrimSpace(title)
	if len(title) < 2 || len(title) > 100 {
		return nil, apperrors.Validation("title must be 2-100 characters", nil)
	}

	existing, err := uc.channels.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if existing != nil {
		return nil, apperrors.Conflict(apperrors.CodeConflict, "user already has a channel")
	}
	if taken, _ := uc.channels.GetBySlug(ctx, channelSlug); taken != nil {
		return nil, apperrors.Conflict(apperrors.CodeConflict, "channel slug already taken")
	}

	ch := &domain.Channel{
		UserID: userID,
		Slug:   channelSlug,
		Title:  title,
	}
	if description != "" {
		desc := strings.TrimSpace(description)
		ch.Description = &desc
	}
	if categoryID != "" {
		cid, err := uuid.Parse(categoryID)
		if err != nil {
			return nil, apperrors.Validation("invalid category_id", nil)
		}
		ch.CategoryID = &cid
	}

	if err := uc.channels.Create(ctx, ch); err != nil {
		return nil, apperrors.Internal(err)
	}

	if _, err := uc.ensureIngestKey(ctx, ch.ID); err != nil {
		return nil, err
	}

	return uc.channels.GetBySlug(ctx, ch.Slug)
}

func (uc *ChannelUseCase) GetBySlug(ctx context.Context, channelSlug string) (*domain.Channel, error) {
	ch, err := uc.channels.GetBySlug(ctx, channelSlug)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if ch == nil {
		return nil, apperrors.NotFound("channel not found")
	}
	return ch, nil
}

func (uc *ChannelUseCase) GetMyChannel(ctx context.Context, userID uuid.UUID) (*domain.Channel, error) {
	ch, err := uc.channels.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if ch == nil {
		return nil, apperrors.NotFound("channel not found")
	}
	return ch, nil
}

func (uc *ChannelUseCase) Update(ctx context.Context, userID uuid.UUID, channelSlug string, title, description, bannerURL, avatarURL, categoryID *string) (*domain.Channel, error) {
	var catID *uuid.UUID
	if categoryID != nil && *categoryID != "" {
		id, err := uuid.Parse(*categoryID)
		if err != nil {
			return nil, apperrors.Validation("invalid category_id", nil)
		}
		catID = &id
	}
	ch, err := uc.channels.Update(ctx, channelSlug, userID, title, description, bannerURL, avatarURL, catID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if ch == nil {
		return nil, apperrors.Forbidden("channel not found or access denied")
	}
	return ch, nil
}

func (uc *ChannelUseCase) Follow(ctx context.Context, userID uuid.UUID, channelSlug string) (int, error) {
	ch, err := uc.getChannelOrError(ctx, channelSlug)
	if err != nil {
		return 0, err
	}
	if ch.UserID == userID {
		return 0, apperrors.Validation("cannot follow your own channel", nil)
	}
	if err := uc.followers.Follow(ctx, userID, ch.ID); err != nil {
		return 0, apperrors.Internal(err)
	}
	count, err := uc.followers.IncrementFollowerCount(ctx, ch.ID, 1)
	if err != nil {
		return 0, apperrors.Internal(err)
	}
	return count, nil
}

func (uc *ChannelUseCase) Unfollow(ctx context.Context, userID uuid.UUID, channelSlug string) (int, error) {
	ch, err := uc.getChannelOrError(ctx, channelSlug)
	if err != nil {
		return 0, err
	}
	following, err := uc.followers.IsFollowing(ctx, userID, ch.ID)
	if err != nil {
		return 0, apperrors.Internal(err)
	}
	if !following {
		return ch.FollowerCount, nil
	}
	if err := uc.followers.Unfollow(ctx, userID, ch.ID); err != nil {
		return 0, apperrors.Internal(err)
	}
	return uc.followers.IncrementFollowerCount(ctx, ch.ID, -1)
}

func (uc *ChannelUseCase) ListFollowers(ctx context.Context, channelSlug string, page, limit int) ([]domain.Follower, pagination.Result, error) {
	ch, err := uc.getChannelOrError(ctx, channelSlug)
	if err != nil {
		return nil, pagination.Result{}, err
	}
	p := pagination.Normalize(page, limit)
	list, total, err := uc.followers.List(ctx, ch.ID, p)
	if err != nil {
		return nil, pagination.Result{}, apperrors.Internal(err)
	}
	return list, pagination.Result{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (uc *ChannelUseCase) IsFollowing(ctx context.Context, userID uuid.UUID, channelSlug string) (bool, error) {
	ch, err := uc.getChannelOrError(ctx, channelSlug)
	if err != nil {
		return false, err
	}
	return uc.followers.IsFollowing(ctx, userID, ch.ID)
}

func (uc *ChannelUseCase) GetIngestKey(ctx context.Context, userID uuid.UUID, channelSlug string) (*IngestKeyResult, error) {
	ch, err := uc.authorizeChannel(ctx, userID, channelSlug)
	if err != nil {
		return nil, err
	}
	return uc.ensureIngestKey(ctx, ch.ID)
}

func (uc *ChannelUseCase) RotateIngestKey(ctx context.Context, userID uuid.UUID, channelSlug string) (*IngestKeyResult, error) {
	ch, err := uc.authorizeChannel(ctx, userID, channelSlug)
	if err != nil {
		return nil, err
	}
	if err := uc.streamKeys.DeactivateByChannel(ctx, ch.ID); err != nil {
		return nil, apperrors.Internal(err)
	}
	return uc.createIngestKey(ctx, ch.ID)
}

func (uc *ChannelUseCase) ensureIngestKey(ctx context.Context, channelID uuid.UUID) (*IngestKeyResult, error) {
	active, err := uc.streamKeys.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if active != nil {
		return &IngestKeyResult{
			StreamKey: "",
			RTMPURL:   uc.rtmpURL,
			SRTURL:    uc.srtURL,
			KeyPrefix: active.KeyPrefix,
		}, nil
	}
	return uc.createIngestKey(ctx, channelID)
}

func (uc *ChannelUseCase) createIngestKey(ctx context.Context, channelID uuid.UUID) (*IngestKeyResult, error) {
	plain, prefix, lookup, err := generateStreamKey()
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	key := &domain.StreamKey{ChannelID: channelID, KeyLookup: lookup, KeyPrefix: prefix, Label: "default"}
	if err := uc.streamKeys.Create(ctx, key); err != nil {
		return nil, apperrors.Internal(err)
	}
	return &IngestKeyResult{
		StreamKey: plain,
		RTMPURL:   uc.rtmpURL,
		SRTURL:    uc.srtURL,
		KeyPrefix: prefix,
	}, nil
}

func (uc *ChannelUseCase) getChannelOrError(ctx context.Context, channelSlug string) (*domain.Channel, error) {
	ch, err := uc.channels.GetBySlug(ctx, channelSlug)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if ch == nil {
		return nil, apperrors.NotFound("channel not found")
	}
	return ch, nil
}

func (uc *ChannelUseCase) authorizeChannel(ctx context.Context, userID uuid.UUID, channelSlug string) (*domain.Channel, error) {
	ch, err := uc.getChannelOrError(ctx, channelSlug)
	if err != nil {
		return nil, err
	}
	if ch.UserID != userID {
		return nil, apperrors.Forbidden("access denied")
	}
	return ch, nil
}

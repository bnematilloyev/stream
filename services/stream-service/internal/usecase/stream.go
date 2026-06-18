package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/crypto"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

type StreamUseCase struct {
	streams    domain.StreamRepository
	channels   domain.ChannelRepository
	streamKeys domain.StreamKeyRepository
	viewers    *ViewerUseCase
}

func NewStreamUseCase(streams domain.StreamRepository, channels domain.ChannelRepository, streamKeys domain.StreamKeyRepository, viewers *ViewerUseCase) *StreamUseCase {
	return &StreamUseCase{streams: streams, channels: channels, streamKeys: streamKeys, viewers: viewers}
}

func (uc *StreamUseCase) Create(ctx context.Context, userID uuid.UUID, channelSlug, title, description, ingestProtocol, latencyMode, visibility, categoryID string, tags []string, scheduledAt *time.Time) (*domain.Stream, error) {
	ch, err := uc.authorizeChannel(ctx, userID, channelSlug)
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if len(title) < 3 || len(title) > 200 {
		return nil, apperrors.Validation("title must be 3-200 characters", nil)
	}
	ingestProtocol, err = normalizeIngestProtocol(ingestProtocol)
	if err != nil {
		return nil, err
	}
	latencyMode, err = normalizeLatencyMode(latencyMode)
	if err != nil {
		return nil, err
	}
	visibility, err = normalizeVisibility(visibility)
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []string{}
	}
	tags, err = normalizeTags(tags)
	if err != nil {
		return nil, err
	}
	s := &domain.Stream{
		ChannelID: ch.ID, Title: title, IngestProtocol: ingestProtocol,
		LatencyMode: latencyMode, Visibility: visibility, Tags: tags, Status: domain.StatusScheduled,
	}
	if description != "" {
		d := strings.TrimSpace(description)
		s.Description = &d
	}
	if categoryID != "" {
		id, err := uuid.Parse(categoryID)
		if err != nil {
			return nil, apperrors.Validation("invalid category_id", nil)
		}
		s.CategoryID = &id
	}
	s.ScheduledAt = scheduledAt
	if err := uc.streams.Create(ctx, s); err != nil {
		return nil, apperrors.Internal(err)
	}
	return uc.streams.GetByID(ctx, s.ID)
}

func (uc *StreamUseCase) Get(ctx context.Context, streamID uuid.UUID) (*domain.Stream, error) {
	if err := uc.streams.ReconcileLiveStream(ctx, streamID); err != nil {
		return nil, apperrors.Internal(err)
	}
	s, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if s == nil {
		return nil, apperrors.NotFound("stream not found")
	}
	uc.enrichStream(ctx, s)
	return s, nil
}

func (uc *StreamUseCase) Update(ctx context.Context, userID, streamID uuid.UUID, title, description, visibility *string, categoryID *uuid.UUID, tags []string) (*domain.Stream, error) {
	if title != nil {
		t := strings.TrimSpace(*title)
		if len(t) < 3 || len(t) > 200 {
			return nil, apperrors.Validation("title must be 3-200 characters", nil)
		}
		title = &t
	}
	if description != nil {
		d := strings.TrimSpace(*description)
		description = &d
	}
	if visibility != nil {
		v, err := normalizeVisibility(*visibility)
		if err != nil {
			return nil, err
		}
		visibility = &v
	}
	if tags != nil {
		var err error
		tags, err = normalizeTags(tags)
		if err != nil {
			return nil, err
		}
	}
	s, err := uc.streams.Update(ctx, streamID, userID, title, description, visibility, categoryID, tags)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if s == nil {
		return nil, apperrors.Forbidden("stream not found or access denied")
	}
	return s, nil
}

func (uc *StreamUseCase) Delete(ctx context.Context, userID, streamID uuid.UUID) error {
	if err := uc.streams.Delete(ctx, streamID, userID); err != nil {
		return apperrors.Forbidden("stream not found or cannot be deleted")
	}
	return nil
}

func (uc *StreamUseCase) ListLive(ctx context.Context, page, limit int) ([]domain.Stream, pagination.Result, error) {
	p := pagination.Normalize(page, limit)
	list, total, err := uc.streams.ListLive(ctx, p)
	if err != nil {
		return nil, pagination.Result{}, apperrors.Internal(err)
	}
	uc.enrichStreams(ctx, list)
	return list, pagination.Result{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (uc *StreamUseCase) ListMarketplaceLive(ctx context.Context, page, limit int) ([]domain.Stream, pagination.Result, error) {
	p := pagination.Normalize(page, limit)
	list, total, err := uc.streams.ListMarketplaceLive(ctx, p)
	if err != nil {
		return nil, pagination.Result{}, apperrors.Internal(err)
	}
	uc.enrichStreams(ctx, list)
	return list, pagination.Result{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (uc *StreamUseCase) ListByChannel(ctx context.Context, channelSlug, status string, page, limit int) ([]domain.Stream, pagination.Result, error) {
	ch, err := uc.channels.GetBySlug(ctx, channelSlug)
	if err != nil {
		return nil, pagination.Result{}, apperrors.Internal(err)
	}
	if ch == nil {
		return nil, pagination.Result{}, apperrors.NotFound("channel not found")
	}
	p := pagination.Normalize(page, limit)
	list, total, err := uc.streams.ListByChannel(ctx, ch.ID, status, p)
	if err != nil {
		return nil, pagination.Result{}, apperrors.Internal(err)
	}
	uc.enrichStreams(ctx, list)
	return list, pagination.Result{Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (uc *StreamUseCase) Start(ctx context.Context, userID, streamID uuid.UUID) (*domain.Stream, error) {
	s, err := uc.getOwnedStream(ctx, userID, streamID)
	if err != nil {
		return nil, err
	}
	if s.Status == domain.StatusLive {
		return s, nil
	}
	if s.Status != domain.StatusScheduled {
		return nil, apperrors.Validation("stream cannot be started", nil)
	}
	now := time.Now()
	if err := uc.streams.SetStatus(ctx, streamID, domain.StatusLive, &now, nil); err != nil {
		return nil, apperrors.Internal(err)
	}
	_ = uc.channels.SetLive(ctx, s.ChannelID, true)
	s.Status = domain.StatusLive
	s.StartedAt = &now
	s.EndedAt = nil
	return s, nil
}

func normalizeIngestProtocol(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "rtmp", nil
	}
	switch value {
	case "rtmp", "srt", "whip":
		return value, nil
	default:
		return "", apperrors.Validation("invalid ingest_protocol", map[string]any{"allowed": []string{"rtmp", "srt", "whip"}})
	}
}

func normalizeLatencyMode(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "standard", nil
	}
	switch value {
	case "standard", "ultra-low":
		return value, nil
	default:
		return "", apperrors.Validation("invalid latency_mode", map[string]any{"allowed": []string{"standard", "ultra-low"}})
	}
}

func normalizeVisibility(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "public", nil
	}
	switch value {
	case "public", "unlisted", "private":
		return value, nil
	default:
		return "", apperrors.Validation("invalid visibility", map[string]any{"allowed": []string{"public", "unlisted", "private"}})
	}
}

func normalizeTags(tags []string) ([]string, error) {
	if len(tags) > 20 {
		return nil, apperrors.Validation("too many tags", map[string]any{"max": 20})
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if len(tag) > 50 {
			return nil, apperrors.Validation("tag is too long", map[string]any{"max": 50})
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out, nil
}

func (uc *StreamUseCase) End(ctx context.Context, userID, streamID uuid.UUID) (*domain.Stream, error) {
	s, err := uc.getOwnedStream(ctx, userID, streamID)
	if err != nil {
		return nil, err
	}
	if s.Status == domain.StatusEnded {
		return s, nil
	}
	if s.Status != domain.StatusLive {
		return nil, apperrors.Validation("stream is not live", nil)
	}
	now := time.Now()
	if err := uc.streams.SetStatus(ctx, streamID, domain.StatusEnded, nil, &now); err != nil {
		return nil, apperrors.Internal(err)
	}
	count, err := uc.streams.CountLiveByChannel(ctx, s.ChannelID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if count == 0 {
		_ = uc.channels.SetLive(ctx, s.ChannelID, false)
	}
	if uc.viewers != nil {
		uc.viewers.ClearStream(ctx, streamID)
	}
	return uc.streams.GetByID(ctx, streamID)
}

func (uc *StreamUseCase) AdminForceEnd(ctx context.Context, streamID uuid.UUID) (*domain.Stream, error) {
	s, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if s == nil {
		return nil, apperrors.NotFound("stream not found")
	}
	if s.Status == domain.StatusEnded {
		return s, nil
	}
	if s.Status != domain.StatusLive {
		return nil, apperrors.Validation("stream is not live", nil)
	}
	now := time.Now()
	if err := uc.streams.SetStatus(ctx, streamID, domain.StatusEnded, nil, &now); err != nil {
		return nil, apperrors.Internal(err)
	}
	count, err := uc.streams.CountLiveByChannel(ctx, s.ChannelID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if count == 0 {
		_ = uc.channels.SetLive(ctx, s.ChannelID, false)
	}
	if uc.viewers != nil {
		uc.viewers.ClearStream(ctx, streamID)
	}
	return uc.streams.GetByID(ctx, streamID)
}

type ValidateKeyResult struct {
	Valid       bool
	ChannelID   uuid.UUID
	ChannelSlug string
	StreamID    string
}

func (uc *StreamUseCase) ValidateStreamKey(ctx context.Context, plainKey string) (*ValidateKeyResult, error) {
	lookup := crypto.SHA256Hex(strings.TrimSpace(plainKey))
	key, err := uc.streamKeys.GetByLookup(ctx, lookup)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if key == nil {
		return &ValidateKeyResult{Valid: false}, nil
	}
	_ = uc.streamKeys.UpdateLastUsed(ctx, key.ID)
	ch, err := uc.channels.GetByID(ctx, key.ChannelID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if ch == nil {
		return &ValidateKeyResult{Valid: false}, nil
	}
	res := &ValidateKeyResult{Valid: true, ChannelID: ch.ID, ChannelSlug: ch.Slug}
	if live, err := uc.streams.GetActiveLiveByChannelAndProtocol(ctx, ch.ID, "rtmp"); err != nil {
		return nil, apperrors.Internal(err)
	} else if live != nil {
		res.StreamID = live.ID.String()
	}
	return res, nil
}

func (uc *StreamUseCase) authorizeChannel(ctx context.Context, userID uuid.UUID, slug string) (*domain.Channel, error) {
	ch, err := uc.channels.GetBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if ch == nil {
		return nil, apperrors.NotFound("channel not found")
	}
	if ch.UserID != userID {
		return nil, apperrors.Forbidden("access denied")
	}
	return ch, nil
}

func (uc *StreamUseCase) getOwnedStream(ctx context.Context, userID, streamID uuid.UUID) (*domain.Stream, error) {
	s, err := uc.streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if s == nil {
		return nil, apperrors.NotFound("stream not found")
	}
	ch, err := uc.channels.GetBySlug(ctx, s.ChannelSlug)
	if err != nil || ch == nil || ch.UserID != userID {
		return nil, apperrors.Forbidden("access denied")
	}
	return s, nil
}

func (uc *StreamUseCase) enrichStream(ctx context.Context, s *domain.Stream) {
	if uc.viewers != nil {
		uc.viewers.EnrichStream(ctx, s)
	}
}

func (uc *StreamUseCase) enrichStreams(ctx context.Context, list []domain.Stream) {
	if uc.viewers != nil {
		uc.viewers.EnrichStreams(ctx, list)
	}
}

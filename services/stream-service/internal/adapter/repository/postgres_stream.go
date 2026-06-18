package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

type PostgresStreamRepository struct{ db *database.Router }

func NewPostgresStreamRepository(db *database.Router) *PostgresStreamRepository {
	return &PostgresStreamRepository{db: db}
}

const streamSelect = `
	SELECT s.id, s.channel_id, c.slug, c.title, s.title, s.description, s.thumbnail_url,
	       s.status, s.ingest_protocol, s.latency_mode, s.visibility, s.category_id, s.tags,
	       s.scheduled_at, s.started_at, s.ended_at, s.viewer_count, s.peak_viewers,
	       c.marketplace_seller_id, c.marketplace_shop_id,
	       s.created_at, s.updated_at
	FROM streams s JOIN channels c ON c.id = s.channel_id
`

func (r *PostgresStreamRepository) Create(ctx context.Context, s *domain.Stream) error {
	query := `
		INSERT INTO streams (channel_id, title, description, ingest_protocol, latency_mode, visibility, category_id, tags, scheduled_at, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, viewer_count, peak_viewers, created_at, updated_at
	`
	return r.db.Write().QueryRow(ctx, query,
		s.ChannelID, s.Title, s.Description, s.IngestProtocol, s.LatencyMode, s.Visibility, s.CategoryID, s.Tags, s.ScheduledAt, s.Status,
	).Scan(&s.ID, &s.ViewerCount, &s.PeakViewers, &s.CreatedAt, &s.UpdatedAt)
}

func (r *PostgresStreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Stream, error) {
	return r.scanOne(ctx, streamSelect+` WHERE s.id = $1`, id)
}

func (r *PostgresStreamRepository) Update(ctx context.Context, id, userID uuid.UUID, title, description, visibility *string, categoryID *uuid.UUID, tags []string) (*domain.Stream, error) {
	query := `
		UPDATE streams s SET
			title = COALESCE($3, s.title),
			description = COALESCE($4, s.description),
			visibility = COALESCE($5, s.visibility),
			category_id = COALESCE($6, s.category_id),
			tags = CASE WHEN $7::text[] IS NOT NULL THEN $7 ELSE s.tags END,
			updated_at = NOW()
		FROM channels c
		WHERE s.id = $1 AND s.channel_id = c.id AND c.user_id = $2
	`
	tag, err := r.db.Write().Exec(ctx, query, id, userID, title, description, visibility, categoryID, tags)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresStreamRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Write().Exec(ctx, `
		DELETE FROM streams s USING channels c
		WHERE s.id = $1 AND s.channel_id = c.id AND c.user_id = $2 AND s.status IN ('scheduled','ended')
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

const liveWithIngest = `
	FROM streams s
	JOIN channels c ON c.id = s.channel_id
	JOIN stream_media sm ON sm.stream_id = s.id AND sm.status = 'ingesting'
`

func (r *PostgresStreamRepository) ListLive(ctx context.Context, p pagination.Params) ([]domain.Stream, int, error) {
	var total int
	if err := r.db.Read().QueryRow(ctx, `SELECT COUNT(*) `+liveWithIngest+` WHERE s.status = 'live' AND s.visibility = 'public'`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Read().Query(ctx, `
		SELECT s.id, s.channel_id, c.slug, c.title, s.title, s.description, s.thumbnail_url,
		       s.status, s.ingest_protocol, s.latency_mode, s.visibility, s.category_id, s.tags,
		       s.scheduled_at, s.started_at, s.ended_at, s.viewer_count, s.peak_viewers,
		       c.marketplace_seller_id, c.marketplace_shop_id,
		       s.created_at, s.updated_at
	`+liveWithIngest+` WHERE s.status = 'live' AND s.visibility = 'public'
		ORDER BY s.viewer_count DESC LIMIT $1 OFFSET $2`, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return r.scanRows(rows, total)
}

func (r *PostgresStreamRepository) ListMarketplaceLive(ctx context.Context, p pagination.Params) ([]domain.Stream, int, error) {
	where := liveWithIngest + ` WHERE s.status = 'live' AND s.visibility = 'public' AND c.marketplace_seller_id IS NOT NULL`
	var total int
	if err := r.db.Read().QueryRow(ctx, `SELECT COUNT(*) `+where).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Read().Query(ctx, `
		SELECT s.id, s.channel_id, c.slug, c.title, s.title, s.description, s.thumbnail_url,
		       s.status, s.ingest_protocol, s.latency_mode, s.visibility, s.category_id, s.tags,
		       s.scheduled_at, s.started_at, s.ended_at, s.viewer_count, s.peak_viewers,
		       c.marketplace_seller_id, c.marketplace_shop_id,
		       s.created_at, s.updated_at
	`+where+`
		ORDER BY s.viewer_count DESC LIMIT $1 OFFSET $2`, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return r.scanRows(rows, total)
}

func (r *PostgresStreamRepository) EndStaleLive(ctx context.Context) error {
	_, err := r.db.Write().Exec(ctx, `
		WITH stale AS (
			SELECT s.id, s.channel_id
			FROM streams s
			WHERE s.status = 'live'
			  AND NOT EXISTS (
			    SELECT 1 FROM stream_media sm
			    WHERE sm.stream_id = s.id AND sm.status = 'ingesting'
			  )
			  AND (s.started_at IS NULL OR s.started_at < NOW() - INTERVAL '90 seconds')
		),
		ended AS (
			UPDATE streams
			SET status = 'ended', ended_at = NOW(), updated_at = NOW()
			WHERE id IN (SELECT id FROM stale)
			RETURNING channel_id
		)
		UPDATE channels c
		SET is_live = FALSE, updated_at = NOW()
		WHERE c.id IN (
			SELECT DISTINCT e.channel_id FROM ended e
			WHERE NOT EXISTS (
				SELECT 1 FROM streams s2
				WHERE s2.channel_id = e.channel_id AND s2.status = 'live'
			)
		)
	`)
	return err
}

func (r *PostgresStreamRepository) ReconcileLiveStream(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Write().Exec(ctx, `
		WITH ended AS (
			UPDATE streams s
			SET status = 'ended', ended_at = NOW(), updated_at = NOW()
			WHERE s.id = $1
			  AND s.status = 'live'
			  AND NOT EXISTS (
			    SELECT 1 FROM stream_media sm
			    WHERE sm.stream_id = s.id AND sm.status = 'ingesting'
			  )
			  AND (s.started_at IS NULL OR s.started_at < NOW() - INTERVAL '90 seconds')
			RETURNING channel_id
		)
		UPDATE channels c
		SET is_live = FALSE, updated_at = NOW()
		WHERE c.id IN (
			SELECT DISTINCT e.channel_id FROM ended e
			WHERE NOT EXISTS (
				SELECT 1 FROM streams s2
				WHERE s2.channel_id = e.channel_id AND s2.status = 'live'
			)
		)
	`, id)
	return err
}

func (r *PostgresStreamRepository) ListByChannel(ctx context.Context, channelID uuid.UUID, status string, p pagination.Params) ([]domain.Stream, int, error) {
	where := ` WHERE s.channel_id = $1`
	args := []any{channelID}
	if status != "" {
		where += ` AND s.status = $2`
		args = append(args, status)
	}
	var total int
	countQ := `SELECT COUNT(*) FROM streams s` + where
	if err := r.db.Read().QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, p.Limit, p.Offset())
	listQ := streamSelect + where + fmt.Sprintf(` ORDER BY s.created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.db.Read().Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return r.scanRows(rows, total)
}

func (r *PostgresStreamRepository) SetStatus(ctx context.Context, id uuid.UUID, status string, startedAt, endedAt *time.Time) error {
	_, err := r.db.Write().Exec(ctx, `
		UPDATE streams
		SET status = $2::varchar,
		    started_at = CASE WHEN $2::varchar = 'live' THEN COALESCE($3, NOW()) ELSE COALESCE($3, started_at) END,
		    ended_at = CASE WHEN $2::varchar = 'live' THEN NULL ELSE COALESCE($4, ended_at) END,
		    updated_at = NOW()
		WHERE id = $1
	`, id, status, startedAt, endedAt)
	return err
}

func (r *PostgresStreamRepository) GetActiveLiveByChannel(ctx context.Context, channelID uuid.UUID) (*domain.Stream, error) {
	return r.scanOne(ctx, streamSelect+` WHERE s.channel_id = $1 AND s.status = 'live' ORDER BY s.started_at DESC LIMIT 1`, channelID)
}

func (r *PostgresStreamRepository) GetActiveLiveByChannelAndProtocol(ctx context.Context, channelID uuid.UUID, ingestProtocol string) (*domain.Stream, error) {
	return r.scanOneArgs(ctx, streamSelect+` WHERE s.channel_id = $1 AND s.status = 'live' AND s.ingest_protocol = $2 ORDER BY s.started_at DESC LIMIT 1`, channelID, ingestProtocol)
}

func (r *PostgresStreamRepository) GetLatestScheduledByChannel(ctx context.Context, channelID uuid.UUID) (*domain.Stream, error) {
	return r.scanOne(ctx, streamSelect+` WHERE s.channel_id = $1 AND s.status = 'scheduled' ORDER BY s.created_at DESC LIMIT 1`, channelID)
}

func (r *PostgresStreamRepository) GetLatestScheduledByChannelAndProtocol(ctx context.Context, channelID uuid.UUID, ingestProtocol string) (*domain.Stream, error) {
	return r.scanOneArgs(ctx, streamSelect+` WHERE s.channel_id = $1 AND s.status = 'scheduled' AND s.ingest_protocol = $2 ORDER BY s.created_at DESC LIMIT 1`, channelID, ingestProtocol)
}

func (r *PostgresStreamRepository) CountLiveByChannel(ctx context.Context, channelID uuid.UUID) (int, error) {
	var n int
	err := r.db.Read().QueryRow(ctx, `SELECT COUNT(*) FROM streams WHERE channel_id = $1 AND status = 'live'`, channelID).Scan(&n)
	return n, err
}

func (r *PostgresStreamRepository) UpdateViewerStats(ctx context.Context, id uuid.UUID, concurrent, unique int) error {
	_, err := r.db.Write().Exec(ctx, `
		UPDATE streams
		SET viewer_count = $2,
		    peak_viewers = GREATEST(peak_viewers, $2, $3),
		    updated_at = NOW()
		WHERE id = $1
	`, id, concurrent, unique)
	return err
}

func (r *PostgresStreamRepository) ListLiveStreamIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Read().Query(ctx, `SELECT id FROM streams WHERE status = 'live'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresStreamRepository) scanOne(ctx context.Context, query string, arg any) (*domain.Stream, error) {
	return r.scanOneArgs(ctx, query, arg)
}

func (r *PostgresStreamRepository) scanOneArgs(ctx context.Context, query string, args ...any) (*domain.Stream, error) {
	rows, err := r.db.Read().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, _, err := r.scanRows(rows, 0)
	if err != nil || len(list) == 0 {
		if err == nil {
			return nil, nil
		}
		return nil, err
	}
	return &list[0], nil
}

func (r *PostgresStreamRepository) scanRows(rows pgx.Rows, total int) ([]domain.Stream, int, error) {
	var list []domain.Stream
	for rows.Next() {
		var s domain.Stream
		if err := rows.Scan(
			&s.ID, &s.ChannelID, &s.ChannelSlug, &s.ChannelTitle, &s.Title, &s.Description, &s.ThumbnailURL,
			&s.Status, &s.IngestProtocol, &s.LatencyMode, &s.Visibility, &s.CategoryID, &s.Tags,
			&s.ScheduledAt, &s.StartedAt, &s.EndedAt, &s.ViewerCount, &s.PeakViewers,
			&s.MarketplaceSellerID, &s.MarketplaceShopID,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, s)
	}
	return list, total, rows.Err()
}

type PostgresChannelRepository struct{ db *database.Router }

func NewPostgresChannelRepository(db *database.Router) *PostgresChannelRepository {
	return &PostgresChannelRepository{db: db}
}

func (r *PostgresChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	var c domain.Channel
	err := r.db.Read().QueryRow(ctx, `SELECT id, user_id, slug, title FROM channels WHERE id = $1`, id).Scan(&c.ID, &c.UserID, &c.Slug, &c.Title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *PostgresChannelRepository) GetBySlug(ctx context.Context, slug string) (*domain.Channel, error) {
	var c domain.Channel
	err := r.db.Read().QueryRow(ctx, `SELECT id, user_id, slug, title FROM channels WHERE slug = $1`, slug).Scan(&c.ID, &c.UserID, &c.Slug, &c.Title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *PostgresChannelRepository) SetLive(ctx context.Context, channelID uuid.UUID, live bool) error {
	_, err := r.db.Write().Exec(ctx, `UPDATE channels SET is_live = $2, updated_at = NOW() WHERE id = $1`, channelID, live)
	return err
}

type PostgresStreamKeyRepository struct{ db *database.Router }

func NewPostgresStreamKeyRepository(db *database.Router) *PostgresStreamKeyRepository {
	return &PostgresStreamKeyRepository{db: db}
}

func (r *PostgresStreamKeyRepository) GetByLookup(ctx context.Context, lookup string) (*domain.StreamKey, error) {
	var k domain.StreamKey
	err := r.db.Read().QueryRow(ctx, `
		SELECT id, channel_id, key_lookup FROM stream_keys WHERE key_lookup = $1 AND is_active = TRUE
	`, lookup).Scan(&k.ID, &k.ChannelID, &k.KeyLookup)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &k, err
}

func (r *PostgresStreamKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Write().Exec(ctx, `UPDATE stream_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
)

type PostgresChannelRepository struct{ pool *pgxpool.Pool }

func NewPostgresChannelRepository(pool *pgxpool.Pool) *PostgresChannelRepository {
	return &PostgresChannelRepository{pool: pool}
}

const channelSelect = `
	SELECT c.id, c.user_id, c.slug, c.title, c.description, c.banner_url, c.avatar_url,
	       c.category_id, cat.slug, c.is_verified, c.is_live, c.follower_count,
	       c.marketplace_seller_id, c.marketplace_shop_id,
	       c.created_at, c.updated_at
	FROM channels c
	LEFT JOIN categories cat ON cat.id = c.category_id
`

func (r *PostgresChannelRepository) Create(ctx context.Context, ch *domain.Channel) error {
	query := `
		INSERT INTO channels (user_id, slug, title, description, category_id, marketplace_seller_id, marketplace_shop_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, is_verified, is_live, follower_count, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, ch.UserID, ch.Slug, ch.Title, ch.Description, ch.CategoryID, ch.MarketplaceSellerID, ch.MarketplaceShopID).Scan(
		&ch.ID, &ch.IsVerified, &ch.IsLive, &ch.FollowerCount, &ch.CreatedAt, &ch.UpdatedAt,
	)
}

func (r *PostgresChannelRepository) GetBySlug(ctx context.Context, slug string) (*domain.Channel, error) {
	return r.scanOne(ctx, channelSelect+` WHERE c.slug = $1`, slug)
}

func (r *PostgresChannelRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Channel, error) {
	return r.scanOne(ctx, channelSelect+` WHERE c.user_id = $1`, userID)
}

func (r *PostgresChannelRepository) GetByMarketplaceSellerID(ctx context.Context, sellerID int64) (*domain.Channel, error) {
	return r.scanOne(ctx, channelSelect+` WHERE c.marketplace_seller_id = $1`, sellerID)
}

func (r *PostgresChannelRepository) Update(ctx context.Context, slug string, userID uuid.UUID, title, description, bannerURL, avatarURL *string, categoryID *uuid.UUID) (*domain.Channel, error) {
	query := `
		UPDATE channels SET
			title = COALESCE($3, title),
			description = COALESCE($4, description),
			banner_url = COALESCE($5, banner_url),
			avatar_url = COALESCE($6, avatar_url),
			category_id = COALESCE($7, category_id),
			updated_at = NOW()
		WHERE slug = $1 AND user_id = $2
	`
	tag, err := r.pool.Exec(ctx, query, slug, userID, title, description, bannerURL, avatarURL, categoryID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return r.GetBySlug(ctx, slug)
}

func (r *PostgresChannelRepository) SetLive(ctx context.Context, channelID uuid.UUID, live bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE channels SET is_live = $2, updated_at = NOW() WHERE id = $1`, channelID, live)
	return err
}

func (r *PostgresChannelRepository) List(ctx context.Context, search string, marketplaceOnly bool, page, limit int) ([]domain.Channel, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	where := "WHERE 1=1"
	args := []any{}
	i := 1
	if marketplaceOnly {
		where += " AND c.marketplace_seller_id IS NOT NULL AND c.marketplace_seller_id > 0"
	}
	if search != "" {
		where += fmt.Sprintf(" AND (c.slug ILIKE $%d OR c.title ILIKE $%d)", i, i)
		args = append(args, "%"+search+"%")
		i++
	}

	countQuery := `SELECT COUNT(*) FROM channels c ` + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := channelSelect + where + fmt.Sprintf(" ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Channel, 0, limit)
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(
			&ch.ID, &ch.UserID, &ch.Slug, &ch.Title, &ch.Description, &ch.BannerURL, &ch.AvatarURL,
			&ch.CategoryID, &ch.CategorySlug, &ch.IsVerified, &ch.IsLive, &ch.FollowerCount,
			&ch.MarketplaceSellerID, &ch.MarketplaceShopID,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, ch)
	}
	return out, total, rows.Err()
}

func (r *PostgresChannelRepository) SetVerified(ctx context.Context, slug string, verified bool) (*domain.Channel, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE channels SET is_verified = $2, updated_at = NOW() WHERE slug = $1`, slug, verified)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return r.GetBySlug(ctx, slug)
}

func (r *PostgresChannelRepository) scanOne(ctx context.Context, query string, arg any) (*domain.Channel, error) {
	var ch domain.Channel
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&ch.ID, &ch.UserID, &ch.Slug, &ch.Title, &ch.Description, &ch.BannerURL, &ch.AvatarURL,
		&ch.CategoryID, &ch.CategorySlug, &ch.IsVerified, &ch.IsLive, &ch.FollowerCount,
		&ch.MarketplaceSellerID, &ch.MarketplaceShopID,
		&ch.CreatedAt, &ch.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("channel query: %w", err)
	}
	return &ch, nil
}

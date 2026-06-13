package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/pkg/pagination"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
)

type PostgresFollowerRepository struct{ pool *pgxpool.Pool }

func NewPostgresFollowerRepository(pool *pgxpool.Pool) *PostgresFollowerRepository {
	return &PostgresFollowerRepository{pool: pool}
}

func (r *PostgresFollowerRepository) Follow(ctx context.Context, followerID, channelID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO followers (follower_id, channel_id) VALUES ($1, $2)
		ON CONFLICT (follower_id, channel_id) DO NOTHING
	`, followerID, channelID)
	return err
}

func (r *PostgresFollowerRepository) Unfollow(ctx context.Context, followerID, channelID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM followers WHERE follower_id = $1 AND channel_id = $2`, followerID, channelID)
	return err
}

func (r *PostgresFollowerRepository) IsFollowing(ctx context.Context, followerID, channelID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = $1 AND channel_id = $2)
	`, followerID, channelID).Scan(&exists)
	return exists, err
}

func (r *PostgresFollowerRepository) List(ctx context.Context, channelID uuid.UUID, p pagination.Params) ([]domain.Follower, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM followers WHERE channel_id = $1`, channelID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.username, u.display_name, f.created_at
		FROM followers f
		JOIN users u ON u.id = f.follower_id
		WHERE f.channel_id = $1
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`, channelID, p.Limit, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []domain.Follower
	for rows.Next() {
		var f domain.Follower
		if err := rows.Scan(&f.UserID, &f.Username, &f.DisplayName, &f.FollowedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, f)
	}
	return list, total, rows.Err()
}

func (r *PostgresFollowerRepository) IncrementFollowerCount(ctx context.Context, channelID uuid.UUID, delta int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		UPDATE channels SET follower_count = GREATEST(0, follower_count + $2), updated_at = NOW()
		WHERE id = $1 RETURNING follower_count
	`, channelID, delta).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("update follower count: %w", err)
	}
	return count, nil
}

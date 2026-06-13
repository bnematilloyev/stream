package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/domain"
)

type PostgresStreamKeyRepository struct{ pool *pgxpool.Pool }

func NewPostgresStreamKeyRepository(pool *pgxpool.Pool) *PostgresStreamKeyRepository {
	return &PostgresStreamKeyRepository{pool: pool}
}

func (r *PostgresStreamKeyRepository) Create(ctx context.Context, key *domain.StreamKey) error {
	query := `
		INSERT INTO stream_keys (channel_id, key_lookup, key_prefix, label, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
		RETURNING id, created_at
	`
	return r.pool.QueryRow(ctx, query, key.ChannelID, key.KeyLookup, key.KeyPrefix, key.Label).Scan(&key.ID, &key.CreatedAt)
}

func (r *PostgresStreamKeyRepository) DeactivateByChannel(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE stream_keys SET is_active = FALSE WHERE channel_id = $1`, channelID)
	return err
}

func (r *PostgresStreamKeyRepository) GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*domain.StreamKey, error) {
	return r.scanOne(ctx, `
		SELECT id, channel_id, key_lookup, key_prefix, label, is_active, last_used_at, created_at
		FROM stream_keys WHERE channel_id = $1 AND is_active = TRUE ORDER BY created_at DESC LIMIT 1
	`, channelID)
}

func (r *PostgresStreamKeyRepository) GetByLookup(ctx context.Context, lookup string) (*domain.StreamKey, error) {
	return r.scanOne(ctx, `
		SELECT id, channel_id, key_lookup, key_prefix, label, is_active, last_used_at, created_at
		FROM stream_keys WHERE key_lookup = $1 AND is_active = TRUE
	`, lookup)
}

func (r *PostgresStreamKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE stream_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PostgresStreamKeyRepository) scanOne(ctx context.Context, query string, arg any) (*domain.StreamKey, error) {
	var k domain.StreamKey
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&k.ID, &k.ChannelID, &k.KeyLookup, &k.KeyPrefix, &k.Label, &k.IsActive, &k.LastUsedAt, &k.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &k, err
}

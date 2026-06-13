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

type PostgresProfileRepository struct{ pool *pgxpool.Pool }

func NewPostgresProfileRepository(pool *pgxpool.Pool) *PostgresProfileRepository {
	return &PostgresProfileRepository{pool: pool}
}

func (r *PostgresProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	return r.getOne(ctx, `SELECT id, email, username, display_name, NULL::text, role, email_verified, created_at FROM users WHERE id = $1`, id)
}

func (r *PostgresProfileRepository) GetByUsername(ctx context.Context, username string) (*domain.Profile, error) {
	return r.getOne(ctx, `SELECT id, email, username, display_name, NULL::text, role, email_verified, created_at FROM users WHERE username = $1`, username)
}

func (r *PostgresProfileRepository) Update(ctx context.Context, id uuid.UUID, displayName, avatarURL *string) (*domain.Profile, error) {
	query := `
		UPDATE users SET
			display_name = COALESCE($2, display_name),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, username, display_name, NULL::text, role, email_verified, created_at
	`
	return r.getOne(ctx, query, id, displayName)
}

func (r *PostgresProfileRepository) getOne(ctx context.Context, query string, args ...any) (*domain.Profile, error) {
	var p domain.Profile
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.Email, &p.Username, &p.DisplayName, &p.AvatarURL,
		&p.Role, &p.EmailVerified, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("profile query: %w", err)
	}
	return &p, nil
}

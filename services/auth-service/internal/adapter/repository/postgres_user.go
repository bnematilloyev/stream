package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, username, display_name, password_hash, role, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email_verified, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		user.Email, user.Username, user.DisplayName, user.PasswordHash, user.Role, user.Status,
	).Scan(&user.ID, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.getOne(ctx, `SELECT id, email, username, display_name, password_hash, email_verified, role, status, last_login_at, created_at, updated_at FROM users WHERE id = $1`, id)
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getOne(ctx, `SELECT id, email, username, display_name, password_hash, email_verified, role, status, last_login_at, created_at, updated_at FROM users WHERE email = $1`, email)
}

func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.getOne(ctx, `SELECT id, email, username, display_name, password_hash, email_verified, role, status, last_login_at, created_at, updated_at FROM users WHERE username = $1`, username)
}

func (r *PostgresUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PostgresUserRepository) getOne(ctx context.Context, query string, arg any) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash,
		&u.EmailVerified, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &u, nil
}

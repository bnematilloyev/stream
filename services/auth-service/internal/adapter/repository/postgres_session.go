package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
)

var ErrSessionNotFound = errors.New("session not found")

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	deviceJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("marshal device info: %w", err)
	}

	query := `
		INSERT INTO sessions (user_id, refresh_token, device_info, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err = r.pool.QueryRow(ctx, query,
		session.UserID, session.RefreshToken, deviceJSON, session.IPAddress, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token, device_info, host(ip_address), expires_at, created_at
		FROM sessions WHERE refresh_token = $1
	`
	var s domain.Session
	var deviceJSON []byte
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.UserID, &s.RefreshToken, &deviceJSON, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}
	if len(deviceJSON) > 0 {
		_ = json.Unmarshal(deviceJSON, &s.DeviceInfo)
	}
	return &s, nil
}

func (r *PostgresSessionRepository) ReplaceByRefreshToken(ctx context.Context, oldToken string, session *domain.Session) error {
	deviceJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("marshal device info: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM sessions WHERE refresh_token = $1`, oldToken)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	query := `
		INSERT INTO sessions (user_id, refresh_token, device_info, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, query,
		session.UserID, session.RefreshToken, deviceJSON, session.IPAddress, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresSessionRepository) DeleteByRefreshToken(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE refresh_token = $1`, token)
	return err
}

func (r *PostgresSessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

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

func (r *PostgresUserRepository) List(ctx context.Context, status, role, search string, page, limit int) ([]domain.User, int, error) {
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
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, status)
		i++
	}
	if role != "" {
		where += fmt.Sprintf(" AND role = $%d", i)
		args = append(args, role)
		i++
	}
	if search != "" {
		where += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d OR display_name ILIKE $%d)", i, i, i)
		args = append(args, "%"+search+"%")
		i++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	query := `SELECT id, email, username, display_name, password_hash, email_verified, role, status, last_login_at, created_at, updated_at FROM users ` +
		where + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	out := make([]domain.User, 0, limit)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash,
			&u.EmailVerified, &u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func (r *PostgresUserRepository) UpdateAdmin(ctx context.Context, id uuid.UUID, role, status *string) (*domain.User, error) {
	query := `
		UPDATE users SET
			role = COALESCE($2, role),
			status = COALESCE($3, status),
			updated_at = NOW()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, id, role, status)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresUserRepository) CountByStatus(ctx context.Context) (int64, map[string]int64, int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM users GROUP BY status`)
	if err != nil {
		return 0, nil, 0, err
	}
	defer rows.Close()

	byStatus := make(map[string]int64)
	var total int64
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return 0, nil, 0, err
		}
		byStatus[status] = count
		total += count
	}
	if err := rows.Err(); err != nil {
		return 0, nil, 0, err
	}

	var admins int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&admins); err != nil {
		return 0, nil, 0, err
	}
	return total, byStatus, admins, nil
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

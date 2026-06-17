package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
)

type PostgresAuditRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuditRepository(pool *pgxpool.Pool) *PostgresAuditRepository {
	return &PostgresAuditRepository{pool: pool}
}

func (r *PostgresAuditRepository) Log(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details map[string]any, ip *string) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, actorID, action, resourceType, resourceID, detailsJSON, ip)
	return err
}

func (r *PostgresAuditRepository) List(ctx context.Context, page, limit int) ([]domain.AuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, action, resource_type, resource_id, details, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.AuditLog, 0, limit)
	for rows.Next() {
		var entry domain.AuditLog
		var details []byte
		if err := rows.Scan(&entry.ID, &entry.ActorID, &entry.Action, &entry.ResourceType, &entry.ResourceID, &details, &entry.CreatedAt); err != nil {
			return nil, 0, err
		}
		entry.DetailsJSON = string(details)
		out = append(out, entry)
	}
	return out, total, rows.Err()
}

package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

type PostgresStreamMediaRepository struct{ pool *pgxpool.Pool }

func NewPostgresStreamMediaRepository(pool *pgxpool.Pool) *PostgresStreamMediaRepository {
	return &PostgresStreamMediaRepository{pool: pool}
}

func (r *PostgresStreamMediaRepository) GetByStreamID(ctx context.Context, streamID uuid.UUID) (*domain.StreamMedia, error) {
	var m domain.StreamMedia
	err := r.pool.QueryRow(ctx, `
		SELECT stream_id, status, hls_path, playback_url, ingest_name, started_at, stopped_at, updated_at
		FROM stream_media WHERE stream_id = $1
	`, streamID).Scan(&m.StreamID, &m.Status, &m.HLSPath, &m.PlaybackURL, &m.IngestName, &m.StartedAt, &m.StoppedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *PostgresStreamMediaRepository) Upsert(ctx context.Context, m *domain.StreamMedia) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stream_media (stream_id, status, hls_path, playback_url, ingest_name, started_at, stopped_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (stream_id) DO UPDATE SET
			status = EXCLUDED.status,
			hls_path = EXCLUDED.hls_path,
			playback_url = EXCLUDED.playback_url,
			ingest_name = EXCLUDED.ingest_name,
			started_at = COALESCE(EXCLUDED.started_at, stream_media.started_at),
			stopped_at = EXCLUDED.stopped_at,
			updated_at = NOW()
	`, m.StreamID, m.Status, m.HLSPath, m.PlaybackURL, m.IngestName, m.StartedAt, m.StoppedAt)
	return err
}

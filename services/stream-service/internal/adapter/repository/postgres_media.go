package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/domain"
)

type PostgresStreamMediaRepository struct{ db *database.Router }

func NewPostgresStreamMediaRepository(db *database.Router) *PostgresStreamMediaRepository {
	return &PostgresStreamMediaRepository{db: db}
}

func (r *PostgresStreamMediaRepository) GetByStreamID(ctx context.Context, streamID uuid.UUID) (*domain.StreamMedia, error) {
	var m domain.StreamMedia
	err := r.db.Read().QueryRow(ctx, `
		SELECT stream_id, status, hls_path, playback_url, ingest_name, ffmpeg_pid, started_at, stopped_at, updated_at
		FROM stream_media WHERE stream_id = $1
	`, streamID).Scan(&m.StreamID, &m.Status, &m.HLSPath, &m.PlaybackURL, &m.IngestName, &m.FFmpegPID, &m.StartedAt, &m.StoppedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *PostgresStreamMediaRepository) Upsert(ctx context.Context, m *domain.StreamMedia) error {
	_, err := r.db.Write().Exec(ctx, `
		INSERT INTO stream_media (stream_id, status, hls_path, playback_url, ingest_name, started_at, stopped_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (stream_id) DO UPDATE SET
			status = EXCLUDED.status,
			hls_path = COALESCE(EXCLUDED.hls_path, stream_media.hls_path),
			playback_url = COALESCE(EXCLUDED.playback_url, stream_media.playback_url),
			ingest_name = COALESCE(EXCLUDED.ingest_name, stream_media.ingest_name),
			started_at = COALESCE(EXCLUDED.started_at, stream_media.started_at),
			stopped_at = EXCLUDED.stopped_at,
			updated_at = NOW()
	`, m.StreamID, m.Status, m.HLSPath, m.PlaybackURL, m.IngestName, m.StartedAt, m.StoppedAt)
	return err
}

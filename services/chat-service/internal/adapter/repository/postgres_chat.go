package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/domain"
)

type PostgresChatRepository struct{ pool *pgxpool.Pool }

func NewPostgresChatRepository(pool *pgxpool.Pool) *PostgresChatRepository {
	return &PostgresChatRepository{pool: pool}
}

func (r *PostgresChatRepository) Insert(ctx context.Context, msg *domain.Message) error {
	const q = `
		INSERT INTO chat_messages (stream_id, user_id, username, display_name, content, type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	return r.pool.QueryRow(ctx, q,
		msg.StreamID, msg.UserID, msg.Username, msg.DisplayName, msg.Content, msg.Type, msg.CreatedAt,
	).Scan(&msg.ID)
}

func (r *PostgresChatRepository) ListHistory(ctx context.Context, streamID uuid.UUID, beforeID int64, limit int) ([]domain.Message, error) {
	var rows pgx.Rows
	var err error
	if beforeID > 0 {
		const q = `
			SELECT id, stream_id, user_id, username, display_name, content, type, created_at
			FROM chat_messages
			WHERE stream_id = $1 AND id < $2
			ORDER BY id DESC
			LIMIT $3`
		rows, err = r.pool.Query(ctx, q, streamID, beforeID, limit)
	} else {
		const q = `
			SELECT id, stream_id, user_id, username, display_name, content, type, created_at
			FROM chat_messages
			WHERE stream_id = $1
			ORDER BY id DESC
			LIMIT $2`
		rows, err = r.pool.Query(ctx, q, streamID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Message, 0, limit)
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.StreamID, &m.UserID, &m.Username, &m.DisplayName, &m.Content, &m.Type, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *PostgresChatRepository) Delete(ctx context.Context, streamID uuid.UUID, messageID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM chat_messages WHERE stream_id = $1 AND id = $2`, streamID, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresChatRepository) IsBanned(ctx context.Context, streamID, userID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM chat_bans
			WHERE stream_id = $1 AND user_id = $2
			  AND (expires_at IS NULL OR expires_at > NOW())
		)`
	var banned bool
	err := r.pool.QueryRow(ctx, q, streamID, userID).Scan(&banned)
	return banned, err
}

var _ domain.ChatRepository = (*PostgresChatRepository)(nil)

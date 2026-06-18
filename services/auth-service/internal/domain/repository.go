package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	List(ctx context.Context, status, role, search string, page, limit int) ([]User, int, error)
	UpdateAdmin(ctx context.Context, id uuid.UUID, role, status *string) (*User, error)
	CountByStatus(ctx context.Context) (total int64, byStatus map[string]int64, admins int64, err error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByRefreshToken(ctx context.Context, token string) (*Session, error)
	ReplaceByRefreshToken(ctx context.Context, oldToken string, session *Session) error
	DeleteByRefreshToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type AuditLog struct {
	ID           int64
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	DetailsJSON  string
	CreatedAt    time.Time
}

type AuditRepository interface {
	Log(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details map[string]any, ip *string) error
	List(ctx context.Context, page, limit int) ([]AuditLog, int, error)
}

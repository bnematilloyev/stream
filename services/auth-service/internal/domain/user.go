package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID
	Email         string
	Username      string
	DisplayName   string
	PasswordHash  string
	EmailVerified bool
	Role          string
	Status        string
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	DeviceInfo   map[string]any
	IPAddress    *string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"

	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusBanned    = "banned"
)

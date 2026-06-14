package auth

import "time"

// Principal is the authenticated user identity attached to a request context.
type Principal struct {
	ID            string
	Email         string
	Username      string
	DisplayName   string
	Role          string
	Status        string
	EmailVerified bool
	CreatedAt     time.Time
}

func (p *Principal) IsActive() bool {
	return p != nil && p.Status == StatusActive
}

const StatusActive = "active"

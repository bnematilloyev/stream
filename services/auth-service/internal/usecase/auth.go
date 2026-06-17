package usecase

import (
	"context"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sahiy/sahiy-stream/pkg/crypto"
	apperrors "github.com/sahiy/sahiy-stream/pkg/errors"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/domain"
)

var (
	emailRegex    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
)

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type AuthResult struct {
	User   *domain.User
	Tokens AuthTokens
}

type AuthUseCase struct {
	users    domain.UserRepository
	sessions domain.SessionRepository
	audit    domain.AuditRepository
	jwt      *crypto.JWTManager
	cache    sessionCache
}

func NewAuthUseCase(
	users domain.UserRepository,
	sessions domain.SessionRepository,
	audit domain.AuditRepository,
	jwt *crypto.JWTManager,
	cache sessionCache,
) *AuthUseCase {
	return &AuthUseCase{
		users:    users,
		sessions: sessions,
		audit:    audit,
		jwt:      jwt,
		cache:    cache,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, email, username, displayName, password string) (*AuthResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)

	if err := validateRegistration(email, username, displayName, password); err != nil {
		return nil, err
	}

	if existing, _ := uc.users.GetByEmail(ctx, email); existing != nil {
		return nil, apperrors.Conflict(apperrors.CodeEmailTaken, "email already registered")
	}
	if existing, _ := uc.users.GetByUsername(ctx, username); existing != nil {
		return nil, apperrors.Conflict(apperrors.CodeUsernameTaken, "username already taken")
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	user := &domain.User{
		Email:        email,
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: hash,
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, apperrors.Internal(err)
	}

	tokens, err := uc.issueTokens(ctx, user, nil, nil)
	if err != nil {
		return nil, err
	}

	uc.cacheUserSession(ctx, user)

	_ = uc.audit.Log(ctx, &user.ID, "user.registered", "user", &user.ID, map[string]any{"email": email}, nil)

	return &AuthResult{User: user, Tokens: *tokens}, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password, deviceInfo, ip string) (*AuthResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if user == nil || !crypto.VerifyPassword(user.PasswordHash, password) {
		return nil, apperrors.New(apperrors.CodeInvalidCreds, "invalid email or password", 401)
	}
	if user.Status != domain.StatusActive {
		return nil, apperrors.Forbidden("account is not active")
	}

	clientIP := normalizeIP(ip)
	tokens, err := uc.issueTokens(ctx, user, &deviceInfo, clientIP)
	if err != nil {
		return nil, err
	}

	uc.cacheUserSession(ctx, user)

	_ = uc.users.UpdateLastLogin(ctx, user.ID)
	_ = uc.audit.Log(ctx, &user.ID, "user.login", "user", &user.ID, map[string]any{}, clientIP)

	return &AuthResult{User: user, Tokens: *tokens}, nil
}

const provisionEmailSuffix = "@broadcast.internal.sahiy"

// SyncProvisionLogin refreshes the internal marketplace seller password and issues tokens.
// Used when SERVICE_TOKEN / BROADCAST_PROVISION_SECRET changes after initial provisioning.
func (uc *AuthUseCase) SyncProvisionLogin(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.HasSuffix(email, provisionEmailSuffix) {
		return nil, apperrors.Forbidden("not a provision account")
	}
	if len(strings.TrimSpace(password)) < 8 {
		return nil, apperrors.Validation("invalid password", nil)
	}

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if user == nil {
		return nil, apperrors.NotFound("user not found")
	}
	if user.Status != domain.StatusActive {
		return nil, apperrors.Forbidden("account is not active")
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if err := uc.users.UpdatePasswordHash(ctx, user.ID, hash); err != nil {
		return nil, apperrors.Internal(err)
	}
	user.PasswordHash = hash

	tokens, err := uc.issueTokens(ctx, user, nil, nil)
	if err != nil {
		return nil, err
	}
	uc.cacheUserSession(ctx, user)
	_ = uc.users.UpdateLastLogin(ctx, user.ID)
	_ = uc.audit.Log(ctx, &user.ID, "user.provision_login", "user", &user.ID, map[string]any{}, nil)

	return &AuthResult{User: user, Tokens: *tokens}, nil
}

func (uc *AuthUseCase) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := uc.jwt.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid refresh token", 401)
	}

	session, err := uc.sessions.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if session == nil || time.Now().After(session.ExpiresAt) {
		return nil, apperrors.New(apperrors.CodeTokenExpired, "refresh token expired", 401)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid token subject", 401)
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if user == nil || user.Status != domain.StatusActive {
		return nil, apperrors.Forbidden("account is not active")
	}

	// Rotate refresh token
	_ = uc.sessions.DeleteByRefreshToken(ctx, refreshToken)

	var ip *string
	if session.IPAddress != nil {
		ip = session.IPAddress
	}
	tokens, err := uc.issueTokens(ctx, user, nil, ip)
	if err != nil {
		return nil, err
	}

	uc.cacheUserSession(ctx, user)

	return &AuthResult{User: user, Tokens: *tokens}, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return apperrors.Validation("refresh_token is required", nil)
	}
	session, err := uc.sessions.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return apperrors.Internal(err)
	}
	if err := uc.sessions.DeleteByRefreshToken(ctx, refreshToken); err != nil {
		return apperrors.Internal(err)
	}
	if session != nil {
		uc.revokeUserSession(ctx, session.UserID)
	}
	return nil
}

func (uc *AuthUseCase) ValidateAccess(ctx context.Context, accessToken string) (*domain.User, error) {
	claims, err := uc.jwt.ValidateAccess(accessToken)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid access token", 401)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid token subject", 401)
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if user == nil || user.Status != domain.StatusActive {
		return nil, apperrors.Forbidden("account is not active")
	}
	return user, nil
}

func (uc *AuthUseCase) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if user == nil {
		return nil, apperrors.NotFound("user not found")
	}
	return user, nil
}

func (uc *AuthUseCase) issueTokens(ctx context.Context, user *domain.User, deviceInfo, ip *string) (*AuthTokens, error) {
	pair, err := uc.jwt.GeneratePair(user.ID.String(), user.Username, user.Role)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	session := &domain.Session{
		UserID:       user.ID,
		RefreshToken: pair.RefreshToken,
		DeviceInfo:   map[string]any{},
		IPAddress:    ip,
		ExpiresAt:    time.Now().Add(uc.jwt.RefreshTTL()),
	}
	if deviceInfo != nil {
		session.DeviceInfo["raw"] = *deviceInfo
	}

	if err := uc.sessions.Create(ctx, session); err != nil {
		return nil, apperrors.Internal(err)
	}

	return &AuthTokens{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
	}, nil
}

func validateRegistration(email, username, displayName, password string) error {
	details := map[string]any{}

	if !emailRegex.MatchString(email) {
		details["email"] = "invalid format"
	}
	if !usernameRegex.MatchString(username) {
		details["username"] = "must be 3-50 chars, alphanumeric or underscore"
	}
	if len(displayName) < 2 || len(displayName) > 100 {
		details["display_name"] = "must be 2-100 characters"
	}
	if len(password) < 8 {
		details["password"] = "must be at least 8 characters"
	}
	if len(details) > 0 {
		return apperrors.Validation("validation failed", details)
	}
	return nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func normalizeIP(addr string) *string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if net.ParseIP(addr) != nil {
			return &addr
		}
		return nil
	}
	if net.ParseIP(host) != nil {
		return &host
	}
	return nil
}

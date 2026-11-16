// Package auth implements the authentication domain end to end: user and
// refresh-token persistence, login/refresh/logout, JWT handling, and the HTTP
// middleware that protects the other domains.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/entity"
)

// AdminSeed is the first admin account, taken from the environment at
// startup. It is only used when there is no active user yet.
type AdminSeed struct {
	Username string
	Password string
	Name     string
}

// Service implements login, refresh, logout, and access-token authentication.
type Service struct {
	users   *UserRepository
	refresh *RefreshTokenRepository
	tokens  *Manager
}

func NewService(users *UserRepository, refresh *RefreshTokenRepository, tokens *Manager) *Service {
	return &Service{users: users, refresh: refresh, tokens: tokens}
}

// Login verifies credentials and returns a new access/refresh token pair.
// Unknown usernames and wrong passwords produce the same error so clients
// cannot enumerate accounts.
func (s *Service) Login(ctx context.Context, input LoginInput) (TokenResponse, error) {
	username, err := validateLoginInput(input)
	if err != nil {
		return TokenResponse{}, err
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return TokenResponse{}, apperr.ErrInvalidCredentials
		}
		return TokenResponse{}, err
	}
	if !VerifyPassword(user.Password, input.Password) {
		return TokenResponse{}, apperr.ErrInvalidCredentials
	}

	access, rawRefresh, next, err := s.newTokenPair(user)
	if err != nil {
		return TokenResponse{}, err
	}
	if err := s.refresh.Create(ctx, next); err != nil {
		return TokenResponse{}, err
	}
	return tokenResponse(user, access, rawRefresh, s.tokens), nil
}

// Refresh rotates a valid refresh token. The old token is deleted in the same
// transaction that stores the replacement, so it cannot be reused.
func (s *Service) Refresh(ctx context.Context, input RefreshTokenInput) (TokenResponse, error) {
	raw, err := validateRefreshTokenInput(input)
	if err != nil {
		return TokenResponse{}, err
	}

	hash := HashRefreshToken(raw)
	now := time.Now().UTC().Format(time.RFC3339)
	stored, err := s.refresh.GetActiveByHash(ctx, hash, now)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return TokenResponse{}, apperr.ErrInvalidToken
		}
		return TokenResponse{}, err
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return TokenResponse{}, apperr.ErrInvalidToken
		}
		return TokenResponse{}, err
	}

	access, newRaw, next, err := s.newTokenPair(user)
	if err != nil {
		return TokenResponse{}, err
	}
	if err := s.refresh.Rotate(ctx, hash, user.ID, next); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return TokenResponse{}, apperr.ErrInvalidToken
		}
		return TokenResponse{}, err
	}
	return tokenResponse(user, access, newRaw, s.tokens), nil
}

// Logout revokes the given refresh token. It is idempotent: a token that is
// already gone is not an error.
func (s *Service) Logout(ctx context.Context, userID int64, input RefreshTokenInput) error {
	raw, err := validateRefreshTokenInput(input)
	if err != nil {
		return err
	}

	if err := s.refresh.RevokeByHash(ctx, HashRefreshToken(raw), userID); err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return err
	}
	return nil
}

// Authenticate validates an access token and loads its active user. The user
// is read from the database on every request so soft deletes and role changes
// take effect immediately.
func (s *Service) Authenticate(ctx context.Context, raw string) (*User, error) {
	claims, err := s.tokens.ParseAccess(raw)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.ErrInvalidToken
		}
		return nil, err
	}
	return user, nil
}

// newTokenPair issues an access token and prepares a refresh token record. The
// record is not inserted yet so callers can choose create or rotate.
func (s *Service) newTokenPair(user *User) (access, rawRefresh string, next *RefreshToken, err error) {
	access, err = s.tokens.IssueAccess(user)
	if err != nil {
		return "", "", nil, err
	}

	rawRefresh, hash, err := s.tokens.NewRefreshToken()
	if err != nil {
		return "", "", nil, err
	}

	next = &RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: s.tokens.RefreshExpiry(),
		CreatedAt: entity.NowMillis(),
	}
	return access, rawRefresh, next, nil
}

// EnsureAdmin creates the first admin when there is no active user. It returns
// true when a user was inserted. When an active user already exists, the seed
// is ignored so restarts never create duplicates or overwrite accounts.
func EnsureAdmin(ctx context.Context, users *UserRepository, seed AdminSeed) (bool, error) {
	count, err := users.CountActive(ctx)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	username := strings.TrimSpace(seed.Username)
	name := strings.TrimSpace(seed.Name)
	switch {
	case username == "":
		return false, fmt.Errorf("ADMIN_USERNAME wajib diisi")
	case utf8.RuneCountInString(username) > maxUsernameLength:
		return false, fmt.Errorf("ADMIN_USERNAME maksimal 50 karakter")
	}
	switch {
	case name == "":
		return false, fmt.Errorf("ADMIN_NAME wajib diisi")
	case utf8.RuneCountInString(name) > maxNameLength:
		return false, fmt.Errorf("ADMIN_NAME maksimal 150 karakter")
	}
	switch {
	case seed.Password == "":
		return false, fmt.Errorf("belum ada user aktif; set ADMIN_PASSWORD untuk membuat admin pertama")
	case len(seed.Password) < MinPasswordLength:
		return false, fmt.Errorf("ADMIN_PASSWORD minimal %d karakter", MinPasswordLength)
	case len(seed.Password) > MaxPasswordLength:
		return false, fmt.Errorf("ADMIN_PASSWORD maksimal %d byte", MaxPasswordLength)
	}

	hash, err := HashPassword(seed.Password)
	if err != nil {
		return false, err
	}

	user := &User{
		Username: username,
		Password: hash,
		Name:     name,
		Role:     RoleAdmin,
	}
	user.MarkCreated(0)
	if err := users.Create(ctx, user); err != nil {
		return false, err
	}
	return true, nil
}

// tokenResponse builds the wire representation shared by login and refresh.
func tokenResponse(user *User, access, refresh string, tokens *Manager) TokenResponse {
	return TokenResponse{
		AccessToken:      access,
		RefreshToken:     refresh,
		TokenType:        "Bearer",
		ExpiresIn:        int64(tokens.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(tokens.RefreshTTL().Seconds()),
		User: AuthUser{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.Name,
			Role:     user.Role,
		},
	}
}

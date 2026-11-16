package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
)

// AdminSeed is the first admin account, taken from the environment at
// startup. It is only used when there is no active user yet.
type AdminSeed struct {
	Username string
	Password string
	Name     string
}

// AuthService implements login, refresh, logout, and access-token
// authentication.
type AuthService struct {
	users   *repository.UserRepository
	refresh *repository.RefreshTokenRepository
	tokens  *auth.Manager
}

func NewAuth(users *repository.UserRepository, refresh *repository.RefreshTokenRepository, tokens *auth.Manager) *AuthService {
	return &AuthService{users: users, refresh: refresh, tokens: tokens}
}

// Login verifies credentials and returns a new access/refresh token pair.
// Unknown usernames and wrong passwords produce the same error so clients
// cannot enumerate accounts.
func (s *AuthService) Login(ctx context.Context, input dto.LoginInput) (dto.TokenResponse, error) {
	username, err := validateLoginInput(input)
	if err != nil {
		return dto.TokenResponse{}, err
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return dto.TokenResponse{}, apperr.ErrInvalidCredentials
		}
		return dto.TokenResponse{}, err
	}
	if !auth.VerifyPassword(user.Password, input.Password) {
		return dto.TokenResponse{}, apperr.ErrInvalidCredentials
	}

	access, rawRefresh, next, err := s.newTokenPair(user)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	if err := s.refresh.Create(ctx, next); err != nil {
		return dto.TokenResponse{}, err
	}
	return tokenResponse(user, access, rawRefresh, s.tokens), nil
}

// Refresh rotates a valid refresh token. The old token is deleted in the same
// transaction that stores the replacement, so it cannot be reused.
func (s *AuthService) Refresh(ctx context.Context, input dto.RefreshTokenInput) (dto.TokenResponse, error) {
	raw, err := validateRefreshTokenInput(input)
	if err != nil {
		return dto.TokenResponse{}, err
	}

	hash := auth.HashRefreshToken(raw)
	now := time.Now().UTC().Format(time.RFC3339)
	stored, err := s.refresh.GetActiveByHash(ctx, hash, now)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return dto.TokenResponse{}, apperr.ErrInvalidToken
		}
		return dto.TokenResponse{}, err
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return dto.TokenResponse{}, apperr.ErrInvalidToken
		}
		return dto.TokenResponse{}, err
	}

	access, newRaw, next, err := s.newTokenPair(user)
	if err != nil {
		return dto.TokenResponse{}, err
	}
	if err := s.refresh.Rotate(ctx, hash, user.ID, next); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return dto.TokenResponse{}, apperr.ErrInvalidToken
		}
		return dto.TokenResponse{}, err
	}
	return tokenResponse(user, access, newRaw, s.tokens), nil
}

// Logout revokes the given refresh token. It is idempotent: a token that is
// already gone is not an error.
func (s *AuthService) Logout(ctx context.Context, userID int64, input dto.RefreshTokenInput) error {
	raw, err := validateRefreshTokenInput(input)
	if err != nil {
		return err
	}

	if err := s.refresh.RevokeByHash(ctx, auth.HashRefreshToken(raw), userID); err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return err
	}
	return nil
}

// Authenticate validates an access token and loads its active user. The user
// is read from the database on every request so soft deletes and role changes
// take effect immediately.
func (s *AuthService) Authenticate(ctx context.Context, raw string) (*model.User, error) {
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
func (s *AuthService) newTokenPair(user *model.User) (access, rawRefresh string, next *model.RefreshToken, err error) {
	access, err = s.tokens.IssueAccess(user)
	if err != nil {
		return "", "", nil, err
	}

	rawRefresh, hash, err := s.tokens.NewRefreshToken()
	if err != nil {
		return "", "", nil, err
	}

	next = &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: s.tokens.RefreshExpiry(),
		CreatedAt: model.NowMillis(),
	}
	return access, rawRefresh, next, nil
}

// EnsureAdmin creates the first admin when there is no active user. It returns
// true when a user was inserted. When an active user already exists, the seed
// is ignored so restarts never create duplicates or overwrite accounts.
func EnsureAdmin(ctx context.Context, users *repository.UserRepository, seed AdminSeed) (bool, error) {
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
	case utf8.RuneCountInString(name) > maxNamaLength:
		return false, fmt.Errorf("ADMIN_NAME maksimal 150 karakter")
	}
	switch {
	case seed.Password == "":
		return false, fmt.Errorf("belum ada user aktif; set ADMIN_PASSWORD untuk membuat admin pertama")
	case len(seed.Password) < auth.MinPasswordLength:
		return false, fmt.Errorf("ADMIN_PASSWORD minimal %d karakter", auth.MinPasswordLength)
	case len(seed.Password) > auth.MaxPasswordLength:
		return false, fmt.Errorf("ADMIN_PASSWORD maksimal %d byte", auth.MaxPasswordLength)
	}

	hash, err := auth.HashPassword(seed.Password)
	if err != nil {
		return false, err
	}

	user := &model.User{
		Username: username,
		Password: hash,
		Name:     name,
		Role:     model.RoleAdmin,
	}
	user.MarkCreated(0)
	if err := users.Create(ctx, user); err != nil {
		return false, err
	}
	return true, nil
}

// tokenResponse builds the wire representation shared by login and refresh.
func tokenResponse(user *model.User, access, refresh string, tokens *auth.Manager) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken:      access,
		RefreshToken:     refresh,
		TokenType:        "Bearer",
		ExpiresIn:        int64(tokens.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(tokens.RefreshTTL().Seconds()),
		User: dto.AuthUser{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.Name,
			Role:     user.Role,
		},
	}
}

// validateLoginInput trims the username and collects missing/oversized fields.
// Passwords are never trimmed.
func validateLoginInput(input dto.LoginInput) (string, error) {
	username := strings.TrimSpace(input.Username)
	errs := apperr.FieldErrors{}

	switch {
	case username == "":
		errs.Add("username", "Username wajib diisi.")
	case utf8.RuneCountInString(username) > maxUsernameLength:
		errs.Add("username", "Username maksimal 50 karakter.")
	}

	switch {
	case input.Password == "":
		errs.Add("password", "Password wajib diisi.")
	case len(input.Password) > auth.MaxPasswordLength:
		errs.Add("password", "Password maksimal 72 karakter.")
	}

	if errs.Any() {
		return username, errs
	}
	return username, nil
}

// validateRefreshTokenInput trims the opaque token and enforces the length
// documented by the contract.
func validateRefreshTokenInput(input dto.RefreshTokenInput) (string, error) {
	token := strings.TrimSpace(input.RefreshToken)
	errs := apperr.FieldErrors{}

	switch {
	case token == "":
		errs.Add("refresh_token", "Refresh token wajib diisi.")
	case len(token) > maxRefreshTokenLength:
		errs.Add("refresh_token", "Refresh token maksimal 512 karakter.")
	}

	if errs.Any() {
		return token, errs
	}
	return token, nil
}

// ValidateLoginInput validates a login payload without touching storage. The
// handler uses it to combine structural and semantic errors in one response.
func ValidateLoginInput(input dto.LoginInput) error {
	_, err := validateLoginInput(input)
	return err
}

// ValidateRefreshTokenInput validates a refresh/logout payload without
// touching storage.
func ValidateRefreshTokenInput(input dto.RefreshTokenInput) error {
	_, err := validateRefreshTokenInput(input)
	return err
}

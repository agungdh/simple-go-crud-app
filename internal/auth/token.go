package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"simple-go-crud-app/internal/shared/apperr"
)

const (
	// MinSecretLength is the minimum JWT secret size accepted at startup.
	MinSecretLength = 32
	// Issuer is embedded in every access token and verified on parse.
	Issuer = "simple-go-crud-app"
	// refreshTokenBytes is the entropy of an opaque refresh token.
	refreshTokenBytes = 32
)

// Claims is the access-token payload. UserID is derived from the registered
// subject claim after parsing, so it is not serialized separately.
type Claims struct {
	UserID   int64  `json:"-"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Manager issues and validates access tokens and creates opaque refresh
// tokens. It holds no per-session state; refresh tokens live in the database.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewManager validates the configuration and returns a token manager.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	if len(secret) < MinSecretLength {
		return nil, fmt.Errorf("jwt secret must be at least %d bytes", MinSecretLength)
	}
	if accessTTL <= 0 {
		return nil, fmt.Errorf("access token ttl must be positive")
	}
	if refreshTTL <= 0 {
		return nil, fmt.Errorf("refresh token ttl must be positive")
	}
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

// AccessTTL is the lifetime of issued access tokens.
func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}

// RefreshTTL is the lifetime of issued refresh tokens.
func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

// IssueAccess returns a signed HS256 JWT for user.
func (m *Manager) IssueAccess(user *User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("issue access token: nil user")
	}

	now := m.now()
	claims := Claims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ParseAccess validates an access token and returns its claims. Every failure
// is reported as apperr.ErrInvalidToken so callers can map it to 401 without
// leaking why the token was rejected.
func (m *Manager) ParseAccess(raw string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)

	claims := new(Claims)
	token, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, apperr.ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, apperr.ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID < 1 {
		return nil, apperr.ErrInvalidToken
	}
	claims.UserID = userID
	return claims, nil
}

// NewRefreshToken returns a new opaque token and the hash to persist. The raw
// token is shown to the client exactly once.
func (m *Manager) NewRefreshToken() (raw, hash string, err error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashRefreshToken(raw), nil
}

// HashRefreshToken returns the hex-encoded SHA-256 hash of a raw refresh
// token. Tokens are never stored raw.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// RefreshExpiry returns the ISO-8601 UTC expiry for a refresh token created
// now. Expiry is stored as text and compared lexicographically.
func (m *Manager) RefreshExpiry() string {
	return m.now().Add(m.refreshTTL).UTC().Format(time.RFC3339)
}

package dto

// LoginInput is the request body for POST /auth/login. Password is the raw
// value and is never returned by the API.
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RefreshTokenInput is the request body for refresh and logout. The token is
// opaque and only shown once by login/refresh.
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthUser is the authenticated user representation embedded in a token
// response. The password hash and audit columns are never included.
type AuthUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// TokenResponse is returned by login and refresh. ExpiresIn and
// RefreshExpiresIn are lifetimes in seconds.
type TokenResponse struct {
	AccessToken      string   `json:"access_token"`
	RefreshToken     string   `json:"refresh_token"`
	TokenType        string   `json:"token_type"`
	ExpiresIn        int64    `json:"expires_in"`
	RefreshExpiresIn int64    `json:"refresh_expires_in"`
	User             AuthUser `json:"user"`
}

package model

import "github.com/uptrace/bun"

// RefreshToken is an opaque, long-lived session token. It is a technical
// record rather than a business entity, so it does not embed Base and does not
// need soft delete or full audit columns. Logout and rotation hard delete the
// row.
//
// Only the SHA-256 hash of the token is stored, so the raw token cannot be
// recovered from the database. ExpiresAt is ISO-8601 text, following the
// project convention for business timestamps.
type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_token,alias:rt"`

	ID        int64  `bun:"id,pk,autoincrement" json:"-"`
	UserID    int64  `bun:"user_id,notnull" json:"-"`
	TokenHash string `bun:"token_hash,notnull" json:"-"`
	ExpiresAt string `bun:"expires_at,notnull" json:"-"`
	CreatedAt int64  `bun:"created_at,notnull" json:"-"`
}

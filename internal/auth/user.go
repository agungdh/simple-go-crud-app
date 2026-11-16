package auth

import (
	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/shared/entity"
)

// User roles. Authorization is enforced by the HTTP layer: admins may read and
// write, regular users may only read.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User is an account that can log in. Password stores a bcrypt hash, never the
// plain password, and is hidden from JSON so it cannot leak into a response.
type User struct {
	bun.BaseModel `bun:"table:user,alias:u"`

	entity.Base

	Username string `bun:"username,notnull" json:"username"`
	Password string `bun:"password,notnull" json:"-"`
	Name     string `bun:"name,notnull" json:"name"`
	Role     string `bun:"role,notnull" json:"role"`
}

// IsAdmin reports whether the user has the admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/entity"
)

var (
	// userAuthColumns includes the password hash and is only used by login.
	userAuthColumns = []string{"id", "username", "password", "name", "role"}
	// userPublicColumns excludes the password hash for middleware and DTOs.
	userPublicColumns = []string{"id", "username", "name", "role"}
)

// UserRepository persists user accounts.
type UserRepository struct {
	db *bun.DB
}

func NewUserRepository(db *bun.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByUsername returns an active user including the password hash. It is used
// only by login.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	user := new(User)

	q := r.db.NewSelect().
		Model(user).
		Column(userAuthColumns...).
		Where("u.username = ?", username)
	q = entity.Active(q)

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}
	return user, nil
}

// GetByID returns an active user without the password hash. It is used to
// authenticate access tokens.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	user := new(User)

	q := r.db.NewSelect().
		Model(user).
		Column(userPublicColumns...).
		Where("u.id = ?", id)
	q = entity.Active(q)

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}
	return user, nil
}

// CountActive returns the number of non-deleted users. Bootstrapping uses it
// to decide whether the first admin must be seeded.
func (r *UserRepository) CountActive(ctx context.Context) (int64, error) {
	q := r.db.NewSelect().
		Model((*User)(nil))
	q = entity.Active(q)

	count, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active users: %w", err)
	}
	return count, nil
}

// Create inserts a new user and sets its generated id.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	if _, err := r.db.NewInsert().
		Model(user).
		Returning("id").
		Exec(ctx); err != nil {
		return fmt.Errorf("insert user %q: %w", user.Username, err)
	}
	return nil
}

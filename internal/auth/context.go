package auth

import (
	"context"

	"simple-go-crud-app/internal/model"
)

type userContextKey struct{}

// WithUser stores the authenticated user in the request context.
func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFromContext returns the authenticated user, if the request passed
// through the authentication middleware.
func UserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(*model.User)
	return user, ok
}

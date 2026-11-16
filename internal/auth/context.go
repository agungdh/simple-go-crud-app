package auth

import "context"

type userContextKey struct{}

// WithUser stores the authenticated user in the request context.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFromContext returns the authenticated user, if the request passed
// through the authentication middleware.
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey{}).(*User)
	return user, ok
}

// ActorID returns the authenticated user id used for audit columns, and reports
// whether the request carried a user. Protected routes already enforce this in
// RequireAuth, so a false result only guards future wiring.
func ActorID(ctx context.Context) (int64, bool) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return 0, false
	}
	return user.ID, true
}

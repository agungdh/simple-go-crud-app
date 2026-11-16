package auth

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/httpx"
)

// RequireAuth validates the Bearer access token and stores the active user in
// the request context. Missing or unusable tokens are 401.
func RequireAuth(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageAuthRequired)
				return
			}

			user, err := svc.Authenticate(r.Context(), raw)
			if err != nil {
				if errors.Is(err, apperr.ErrInvalidToken) {
					httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageInvalidToken)
					return
				}
				log.Printf("authenticate request: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, httpx.MessageInternal)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

// RequireAdmin rejects authenticated users whose role is not admin. It must run
// after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok || user.Role != RoleAdmin {
			httpx.WriteError(w, http.StatusForbidden, httpx.MessageForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bearerToken extracts the token from an Authorization header. The scheme is
// case-insensitive and the token itself must not contain whitespace.
func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

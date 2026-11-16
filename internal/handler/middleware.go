package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/service"
)

// RequireAuth validates the Bearer access token and stores the active user in
// the request context. Missing or unusable tokens are 401.
func RequireAuth(svc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeError(w, http.StatusUnauthorized, messageAuthRequired)
				return
			}

			user, err := svc.Authenticate(r.Context(), raw)
			if err != nil {
				if errors.Is(err, apperr.ErrInvalidToken) {
					writeError(w, http.StatusUnauthorized, messageInvalidToken)
					return
				}
				log.Printf("authenticate request: %v", err)
				writeError(w, http.StatusInternalServerError, messageInternal)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), user)))
		})
	}
}

// RequireAdmin rejects authenticated users whose role is not admin. It must run
// after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok || user.Role != model.RoleAdmin {
			writeError(w, http.StatusForbidden, messageForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireActor returns the authenticated user's id for audit columns. It
// writes 401 when the request somehow lacks a user; protected routes already
// enforce this in RequireAuth, so the fallback only guards future wiring.
func requireActor(w http.ResponseWriter, r *http.Request) (int64, bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, messageAuthRequired)
		return 0, false
	}
	return user.ID, true
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

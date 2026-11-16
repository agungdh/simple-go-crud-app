package handler

import (
	"net/http"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/service"
)

// Auth handles the authentication endpoints.
type Auth struct {
	service *service.AuthService
}

func NewAuth(svc *service.AuthService) *Auth {
	return &Auth{service: svc}
}

// Login handles POST /auth/login.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeLoginInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidateLoginInput(input))
		writeServiceError(w, "validate login", err)
		return
	}

	result, err := h.service.Login(r.Context(), input)
	if err != nil {
		writeServiceError(w, "login", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Refresh handles POST /auth/refresh.
func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeRefreshTokenInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidateRefreshTokenInput(input))
		writeServiceError(w, "validate refresh token", err)
		return
	}

	result, err := h.service.Refresh(r.Context(), input)
	if err != nil {
		writeServiceError(w, "refresh token", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Logout handles POST /auth/logout. The access token must be valid so a user
// can only revoke their own sessions.
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, messageAuthRequired)
		return
	}

	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeRefreshTokenInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidateRefreshTokenInput(input))
		writeServiceError(w, "validate refresh token", err)
		return
	}

	if err := h.service.Logout(r.Context(), user.ID, input); err != nil {
		writeServiceError(w, "logout", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

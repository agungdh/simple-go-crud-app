package auth

import (
	"net/http"

	"simple-go-crud-app/internal/shared/httpx"
)

// Handler handles the authentication endpoints.
type Handler struct {
	service *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	fields, ok := httpx.ReadObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeLoginInput(fields)
	if structural.Any() {
		_, semantic := validateLoginInput(input)
		err := httpx.CombineValidation(structural, semantic)
		httpx.WriteServiceError(w, "validate login", err)
		return
	}

	result, err := h.service.Login(r.Context(), input)
	if err != nil {
		httpx.WriteServiceError(w, "login", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	fields, ok := httpx.ReadObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeRefreshTokenInput(fields)
	if structural.Any() {
		_, semantic := validateRefreshTokenInput(input)
		err := httpx.CombineValidation(structural, semantic)
		httpx.WriteServiceError(w, "validate refresh token", err)
		return
	}

	result, err := h.service.Refresh(r.Context(), input)
	if err != nil {
		httpx.WriteServiceError(w, "refresh token", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// Logout handles POST /auth/logout. The access token must be valid so a user
// can only revoke their own sessions.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageAuthRequired)
		return
	}

	fields, ok := httpx.ReadObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeRefreshTokenInput(fields)
	if structural.Any() {
		_, semantic := validateRefreshTokenInput(input)
		err := httpx.CombineValidation(structural, semantic)
		httpx.WriteServiceError(w, "validate refresh token", err)
		return
	}

	if err := h.service.Logout(r.Context(), actor, input); err != nil {
		httpx.WriteServiceError(w, "logout", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

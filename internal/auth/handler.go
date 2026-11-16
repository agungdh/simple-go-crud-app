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

// RequireActor resolves the authenticated actor for write handlers. Protected
// routes already guarantee it; this guards future wiring and keeps handlers
// free of the 401 boilerplate.
func RequireActor(w http.ResponseWriter, r *http.Request) (int64, bool) {
	actor, ok := ActorID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageAuthRequired)
		return 0, false
	}
	return actor, true
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	input, ok := httpx.ReadInput(w, r, decodeLoginInput, validateLoginInput)
	if !ok {
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
	input, ok := httpx.ReadInput(w, r, decodeRefreshTokenInput, validateRefreshTokenInput)
	if !ok {
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
	actor, ok := RequireActor(w, r)
	if !ok {
		return
	}

	input, ok := httpx.ReadInput(w, r, decodeRefreshTokenInput, validateRefreshTokenInput)
	if !ok {
		return
	}

	if err := h.service.Logout(r.Context(), actor, input); err != nil {
		httpx.WriteServiceError(w, "logout", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

package bidang

import (
	"encoding/json"
	"net/http"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/httpx"
)

// Handler handles the bidang endpoints.
type Handler struct {
	service *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// List handles GET /bidang.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		httpx.WriteServiceError(w, "list bidang", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope[Response]{Data: items})
}

// Get handles GET /bidang/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(w, r)
	if !ok {
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, "get bidang", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Envelope[Response]{Data: item})
}

// Create handles POST /bidang.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.ActorID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageAuthRequired)
		return
	}

	fields, ok := httpx.ReadObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeInput(fields)
	if structural.Any() {
		_, semantic := validateInput(input)
		err := httpx.CombineValidation(structural, semantic)
		httpx.WriteServiceError(w, "validate bidang", err)
		return
	}

	if err := h.service.Create(r.Context(), input, actor); err != nil {
		httpx.WriteServiceError(w, "create bidang", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Update handles PUT /bidang/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.ActorID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageAuthRequired)
		return
	}

	id, ok := httpx.ParseID(w, r)
	if !ok {
		return
	}

	fields, ok := httpx.ReadObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeInput(fields)
	if structural.Any() {
		_, semantic := validateInput(input)
		err := httpx.CombineValidation(structural, semantic)
		httpx.WriteServiceError(w, "validate bidang", err)
		return
	}

	if err := h.service.Update(r.Context(), id, input, actor); err != nil {
		httpx.WriteServiceError(w, "update bidang", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Delete handles DELETE /bidang/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.ActorID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.MessageAuthRequired)
		return
	}

	id, ok := httpx.ParseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id, actor); err != nil {
		httpx.WriteServiceError(w, "delete bidang", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeInput maps the wire object to the DTO and collects structural problems
// (unknown fields and wrong JSON types).
func decodeInput(fields map[string]json.RawMessage) (Input, apperr.FieldErrors) {
	errs := apperr.FieldErrors{}
	httpx.RejectUnknown(fields, map[string]struct{}{"bidang": {}}, errs)

	return Input{
		Bidang: httpx.DecodeString(fields, "bidang", "Bidang", errs),
	}, errs
}

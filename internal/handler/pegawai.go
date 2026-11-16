package handler

import (
	"net/http"

	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/service"
)

// Pegawai handles the pegawai endpoints.
type Pegawai struct {
	service *service.PegawaiService
}

func NewPegawai(svc *service.PegawaiService) *Pegawai {
	return &Pegawai{service: svc}
}

// List handles GET /pegawai.
func (h *Pegawai) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, "list pegawai", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.PegawaiListResponse{Data: items})
}

// Get handles GET /pegawai/{id}.
func (h *Pegawai) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, "get pegawai", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.PegawaiResponse{Data: item})
}

// Create handles POST /pegawai.
func (h *Pegawai) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}

	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodePegawaiInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidatePegawaiInput(input))
		writeServiceError(w, "validate pegawai", err)
		return
	}

	if err := h.service.Create(r.Context(), input, actor); err != nil {
		writeServiceError(w, "create pegawai", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Update handles PUT /pegawai/{id}.
func (h *Pegawai) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}

	id, ok := parseID(w, r)
	if !ok {
		return
	}

	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodePegawaiInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidatePegawaiInput(input))
		writeServiceError(w, "validate pegawai", err)
		return
	}

	if err := h.service.Update(r.Context(), id, input, actor); err != nil {
		writeServiceError(w, "update pegawai", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Delete handles DELETE /pegawai/{id}.
func (h *Pegawai) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}

	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id, actor); err != nil {
		writeServiceError(w, "delete pegawai", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

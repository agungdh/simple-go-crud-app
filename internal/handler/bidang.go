package handler

import (
	"net/http"

	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/service"
)

// Bidang handles the bidang endpoints.
type Bidang struct {
	service *service.BidangService
}

func NewBidang(svc *service.BidangService) *Bidang {
	return &Bidang{service: svc}
}

// List handles GET /bidang.
func (h *Bidang) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, "list bidang", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.BidangListResponse{Data: items})
}

// Get handles GET /bidang/{id}.
func (h *Bidang) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, "get bidang", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.BidangResponse{Data: item})
}

// Create handles POST /bidang.
func (h *Bidang) Create(w http.ResponseWriter, r *http.Request) {
	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeBidangInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidateBidangInput(input))
		writeServiceError(w, "validate bidang", err)
		return
	}

	if err := h.service.Create(r.Context(), input); err != nil {
		writeServiceError(w, "create bidang", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Update handles PUT /bidang/{id}.
func (h *Bidang) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	fields, ok := readObject(w, r)
	if !ok {
		return
	}

	input, structural := decodeBidangInput(fields)
	if structural.Any() {
		err := combineValidation(structural, service.ValidateBidangInput(input))
		writeServiceError(w, "validate bidang", err)
		return
	}

	if err := h.service.Update(r.Context(), id, input); err != nil {
		writeServiceError(w, "update bidang", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Delete handles DELETE /bidang/{id}.
func (h *Bidang) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeServiceError(w, "delete bidang", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/service"
)

// BidangHandler exposes the bidang CRUD endpoints.
type BidangHandler struct {
	svc *service.BidangService
}

func NewBidangHandler(svc *service.BidangService) *BidangHandler {
	return &BidangHandler{svc: svc}
}

// Routes registers the bidang endpoints on the given router.
func (h *BidangHandler) Routes(r chi.Router) {
	r.Get("/bidang", h.List)
	r.Post("/bidang", h.Create)
	r.Get("/bidang/{id}", h.Get)
	r.Put("/bidang/{id}", h.Update)
	r.Delete("/bidang/{id}", h.Delete)
}

type bidangRequest struct {
	Bidang string `json:"bidang"`
}

type bidangResponse struct {
	ID     int64  `json:"id"`
	Bidang string `json:"bidang"`
}

func (h *BidangHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	res := make([]bidangResponse, 0, len(items))
	for i := range items {
		res = append(res, toBidangResponse(&items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": res})
}

func (h *BidangHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	bidang, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toBidangResponse(bidang)})
}

func (h *BidangHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeBidangRequest(w, r)
	if !ok {
		return
	}

	bidang, err := h.svc.Create(r.Context(), toBidangInput(req))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toBidangResponse(bidang)})
}

func (h *BidangHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	req, ok := decodeBidangRequest(w, r)
	if !ok {
		return
	}

	bidang, err := h.svc.Update(r.Context(), id, toBidangInput(req))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toBidangResponse(bidang)})
}

func (h *BidangHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BidangHandler) writeError(w http.ResponseWriter, err error) {
	var validationErr *apperr.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeError(w, http.StatusUnprocessableEntity, validationErr.Message)
	case errors.Is(err, apperr.ErrNotFound):
		writeError(w, http.StatusNotFound, "bidang tidak ditemukan")
	case errors.Is(err, apperr.ErrConflict):
		writeError(w, http.StatusConflict, "bidang sudah terdaftar")
	default:
		log.Printf("bidang handler: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func decodeBidangRequest(w http.ResponseWriter, r *http.Request) (bidangRequest, bool) {
	var req bidangRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "body request terlalu besar")
			return req, false
		}
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return req, false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "body JSON hanya boleh berisi satu objek")
		return req, false
	}

	return req, true
}

func toBidangInput(req bidangRequest) service.BidangInput {
	return service.BidangInput{Bidang: req.Bidang}
}

func toBidangResponse(bidang *model.Bidang) bidangResponse {
	return bidangResponse{
		ID:     bidang.ID,
		Bidang: bidang.Bidang,
	}
}

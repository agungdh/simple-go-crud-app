package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/service"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// PegawaiHandler exposes the pegawai CRUD endpoints.
type PegawaiHandler struct {
	svc *service.PegawaiService
}

func NewPegawaiHandler(svc *service.PegawaiService) *PegawaiHandler {
	return &PegawaiHandler{svc: svc}
}

// Routes registers the pegawai endpoints on the given router.
func (h *PegawaiHandler) Routes(r chi.Router) {
	r.Get("/pegawai", h.List)
	r.Post("/pegawai", h.Create)
	r.Get("/pegawai/{id}", h.Get)
	r.Put("/pegawai/{id}", h.Update)
	r.Delete("/pegawai/{id}", h.Delete)
}

type pegawaiRequest struct {
	Nama         string `json:"nama"`
	NIP          string `json:"nip"`
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	JenisKelamin string `json:"jenis_kelamin"`
}

type pegawaiResponse struct {
	ID           int64  `json:"id"`
	Nama         string `json:"nama"`
	NIP          string `json:"nip"`
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	JenisKelamin string `json:"jenis_kelamin"`
}

func (h *PegawaiHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}

	res := make([]pegawaiResponse, 0, len(items))
	for i := range items {
		res = append(res, toResponse(&items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": res})
}

func (h *PegawaiHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	pegawai, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toResponse(pegawai)})
}

func (h *PegawaiHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeRequest(w, r)
	if !ok {
		return
	}

	pegawai, err := h.svc.Create(r.Context(), toInput(req))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toResponse(pegawai)})
}

func (h *PegawaiHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	req, ok := decodeRequest(w, r)
	if !ok {
		return
	}

	pegawai, err := h.svc.Update(r.Context(), id, toInput(req))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toResponse(pegawai)})
}

func (h *PegawaiHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *PegawaiHandler) writeError(w http.ResponseWriter, err error) {
	var validationErr *apperr.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeError(w, http.StatusUnprocessableEntity, validationErr.Message)
	case errors.Is(err, apperr.ErrNotFound):
		writeError(w, http.StatusNotFound, "pegawai tidak ditemukan")
	case errors.Is(err, apperr.ErrConflict):
		writeError(w, http.StatusConflict, "NIP sudah terdaftar")
	default:
		log.Printf("pegawai handler: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return 0, false
	}
	return id, true
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (pegawaiRequest, bool) {
	var req pegawaiRequest

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

func toInput(req pegawaiRequest) service.PegawaiInput {
	return service.PegawaiInput{
		Nama:         req.Nama,
		NIP:          req.NIP,
		TanggalLahir: req.TanggalLahir,
		Alamat:       req.Alamat,
		JenisKelamin: req.JenisKelamin,
	}
}

func toResponse(pegawai *model.Pegawai) pegawaiResponse {
	return pegawaiResponse{
		ID:           pegawai.ID,
		Nama:         pegawai.Nama,
		NIP:          pegawai.NIP,
		TanggalLahir: service.FormatTanggalLahir(pegawai.TanggalLahir),
		Alamat:       pegawai.Alamat,
		JenisKelamin: pegawai.JenisKelamin,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(message)})
}

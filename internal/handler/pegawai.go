// Package handler contains the HTTP handlers for the application.
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/repository"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// Pegawai serves the pegawai page and its JSON API.
type Pegawai struct {
	repo     *repository.PegawaiRepository
	renderer *render.Renderer
	page     pages.Page
}

// NewPegawai creates a pegawai handler.
func NewPegawai(repo *repository.PegawaiRepository, renderer *render.Renderer, page pages.Page) *Pegawai {
	return &Pegawai{repo: repo, renderer: renderer, page: page}
}

type pageData struct {
	pages.Page
	Pegawai    []model.Pegawai
	Item       *model.Pegawai
	FormAction string
	Method     string
	Status     string
}

type pegawaiPayload struct {
	NIP     string `json:"nip"`
	Nama    string `json:"nama"`
	Jabatan string `json:"jabatan"`
	Email   string `json:"email"`
	Telepon string `json:"telepon"`
	Alamat  string `json:"alamat"`
}

func (p *pegawaiPayload) normalize() {
	p.NIP = strings.TrimSpace(p.NIP)
	p.Nama = strings.TrimSpace(p.Nama)
	p.Jabatan = strings.TrimSpace(p.Jabatan)
	p.Email = strings.TrimSpace(p.Email)
	p.Telepon = strings.TrimSpace(p.Telepon)
	p.Alamat = strings.TrimSpace(p.Alamat)
}

func (p *pegawaiPayload) validate() string {
	if p.NIP == "" {
		return "NIP wajib diisi"
	}
	if p.Nama == "" {
		return "Nama wajib diisi"
	}
	if p.Email != "" {
		if _, err := mail.ParseAddress(p.Email); err != nil {
			return "Format email tidak valid"
		}
	}
	return ""
}

// Page renders the pegawai list page.
func (h *Pegawai) Page(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("pegawai page: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := pageData{
		Page:    h.page,
		Pegawai: items,
		Status:  r.URL.Query().Get("status"),
	}
	h.render(w, h.page, data)
}

// CreateForm renders the add-pegawai page.
func (h *Pegawai) CreateForm(w http.ResponseWriter, r *http.Request) {
	page := pages.PegawaiCreate
	h.render(w, page, pageData{
		Page:       page,
		FormAction: "/api/pegawai",
		Method:     http.MethodPost,
	})
}

// EditForm renders the edit-pegawai page with the current values filled in.
func (h *Pegawai) EditForm(w http.ResponseWriter, r *http.Request) {
	page := pages.PegawaiEdit

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID pegawai tidak valid", http.StatusBadRequest)
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		http.Error(w, "Data pegawai tidak ditemukan", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("pegawai edit form: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.render(w, page, pageData{
		Page:       page,
		Item:       item,
		FormAction: "/api/pegawai/" + strconv.FormatInt(id, 10),
		Method:     http.MethodPut,
	})
}

func (h *Pegawai) render(w http.ResponseWriter, page pages.Page, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderData(w, page, data); err != nil {
		log.Printf("render %s: %v", page.Route, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// List returns every pegawai as JSON.
func (h *Pegawai) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("pegawai list: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat data pegawai")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// Get returns a single pegawai as JSON.
func (h *Pegawai) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeRepoError(w, err, "Gagal memuat data pegawai")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

// Create stores a new pegawai and returns it as JSON.
func (h *Pegawai) Create(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodePayload(w, r)
	if !ok {
		return
	}

	item := &model.Pegawai{
		NIP:     payload.NIP,
		Nama:    payload.Nama,
		Jabatan: payload.Jabatan,
		Email:   payload.Email,
		Telepon: payload.Telepon,
		Alamat:  payload.Alamat,
	}
	if err := h.repo.Create(r.Context(), item); err != nil {
		writeRepoError(w, err, "Gagal menyimpan data pegawai")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item})
}

// Update changes an existing pegawai and returns it as JSON.
func (h *Pegawai) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	payload, ok := decodePayload(w, r)
	if !ok {
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeRepoError(w, err, "Gagal memuat data pegawai")
		return
	}

	item.NIP = payload.NIP
	item.Nama = payload.Nama
	item.Jabatan = payload.Jabatan
	item.Email = payload.Email
	item.Telepon = payload.Telepon
	item.Alamat = payload.Alamat

	if err := h.repo.Update(r.Context(), item); err != nil {
		writeRepoError(w, err, "Gagal menyimpan data pegawai")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

// Delete removes a pegawai.
func (h *Pegawai) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepoError(w, err, "Gagal menghapus data pegawai")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID pegawai tidak valid")
		return 0, false
	}
	return id, true
}

func decodePayload(w http.ResponseWriter, r *http.Request) (*pegawaiPayload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var payload pegawaiPayload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Body JSON tidak valid")
		return nil, false
	}

	payload.normalize()
	if msg := payload.validate(); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return nil, false
	}
	return &payload, true
}

func writeRepoError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "Data pegawai tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicateNIP):
		writeError(w, http.StatusConflict, "NIP sudah terdaftar")
	default:
		log.Printf("pegawai: %v", err)
		writeError(w, http.StatusInternalServerError, fallback)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

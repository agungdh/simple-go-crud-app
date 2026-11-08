package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/repository"
)

// Bidang serves the bidang page and its JSON API.
type Bidang struct {
	repo     *repository.BidangRepository
	renderer *render.Renderer
	page     pages.Page
}

// NewBidang creates a bidang handler.
func NewBidang(repo *repository.BidangRepository, renderer *render.Renderer, page pages.Page) *Bidang {
	return &Bidang{repo: repo, renderer: renderer, page: page}
}

type bidangPageData struct {
	pages.Page
	Item       *model.Bidang
	FormAction string
	Method     string
	Status     string
}

// bidangDatatableColumns maps DataTables column indexes to database columns.
var bidangDatatableColumns = []string{"b.id", "b.kode", "b.nama"}

type bidangPayload struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

func (p *bidangPayload) normalize() {
	p.Kode = strings.TrimSpace(p.Kode)
	p.Nama = strings.TrimSpace(p.Nama)
}

func (p *bidangPayload) validate() string {
	if p.Kode == "" {
		return "Kode wajib diisi"
	}
	if p.Nama == "" {
		return "Nama wajib diisi"
	}
	return ""
}

// Page renders the bidang list page. Rows are loaded by DataTables from
// /api/bidang/datatable, so no data is fetched here.
func (h *Bidang) Page(w http.ResponseWriter, r *http.Request) {
	h.render(w, h.page, bidangPageData{
		Page:   h.page,
		Status: r.URL.Query().Get("status"),
	})
}

// Datatable returns a server-side processed page for DataTables.
func (h *Bidang) Datatable(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	draw := queryInt(query.Get("draw"), 0)
	start := queryInt(query.Get("start"), 0)
	if start < 0 {
		start = 0
	}

	length := queryInt(query.Get("length"), 10)
	if length == -1 || length > 100 {
		length = 100
	}
	if length <= 0 {
		length = 10
	}

	search := strings.TrimSpace(query.Get("search[value]"))
	if utf8.RuneCountInString(search) > maxSearchLength {
		search = string([]rune(search)[:maxSearchLength])
	}

	orderBy := "b.id"
	if index, err := strconv.Atoi(query.Get("order[0][column]")); err == nil && index >= 0 && index < len(bidangDatatableColumns) {
		orderBy = bidangDatatableColumns[index]
	}
	orderDir := "asc"
	if strings.EqualFold(query.Get("order[0][dir]"), "desc") {
		orderDir = "desc"
	}

	items, total, filtered, err := h.repo.Query(r.Context(), repository.BidangQuery{
		Search:   search,
		OrderBy:  orderBy,
		OrderDir: orderDir,
		Limit:    length,
		Offset:   start,
	})
	if err != nil {
		log.Printf("bidang datatable: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat data bidang")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"draw":            draw,
		"recordsTotal":    total,
		"recordsFiltered": filtered,
		"data":            items,
	})
}

// Search returns bidang matching q for the Select2 autocomplete.
func (h *Bidang) Search(w http.ResponseWriter, r *http.Request) {
	keyword := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(keyword) > maxSearchLength {
		keyword = string([]rune(keyword)[:maxSearchLength])
	}

	items, err := h.repo.Search(r.Context(), keyword, 20)
	if err != nil {
		log.Printf("bidang search: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mencari bidang")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// CreateForm renders the add-bidang page.
func (h *Bidang) CreateForm(w http.ResponseWriter, r *http.Request) {
	page := pages.BidangCreate
	h.render(w, page, bidangPageData{
		Page:       page,
		FormAction: "/api/bidang",
		Method:     http.MethodPost,
	})
}

// EditForm renders the edit-bidang page with the current values filled in.
func (h *Bidang) EditForm(w http.ResponseWriter, r *http.Request) {
	page := pages.BidangEdit

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID bidang tidak valid", http.StatusBadRequest)
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		http.Error(w, "Data bidang tidak ditemukan", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("bidang edit form: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.render(w, page, bidangPageData{
		Page:       page,
		Item:       item,
		FormAction: "/api/bidang/" + strconv.FormatInt(id, 10),
		Method:     http.MethodPut,
	})
}

func (h *Bidang) render(w http.ResponseWriter, page pages.Page, data bidangPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderData(w, page, data); err != nil {
		log.Printf("render %s: %v", page.Route, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// List returns every bidang as JSON.
func (h *Bidang) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("bidang list: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat data bidang")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// Get returns a single bidang as JSON.
func (h *Bidang) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseBidangID(w, r)
	if !ok {
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeBidangRepoError(w, err, "Gagal memuat data bidang")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

// Create stores a new bidang and returns it as JSON.
func (h *Bidang) Create(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeBidangPayload(w, r)
	if !ok {
		return
	}

	item := &model.Bidang{
		Kode: payload.Kode,
		Nama: payload.Nama,
	}
	if err := h.repo.Create(r.Context(), item); err != nil {
		writeBidangRepoError(w, err, "Gagal menyimpan data bidang")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item})
}

// Update changes an existing bidang and returns it as JSON.
func (h *Bidang) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseBidangID(w, r)
	if !ok {
		return
	}
	payload, ok := decodeBidangPayload(w, r)
	if !ok {
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeBidangRepoError(w, err, "Gagal memuat data bidang")
		return
	}

	item.Kode = payload.Kode
	item.Nama = payload.Nama

	if err := h.repo.Update(r.Context(), item); err != nil {
		writeBidangRepoError(w, err, "Gagal menyimpan data bidang")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

// Delete removes a bidang.
func (h *Bidang) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseBidangID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeBidangRepoError(w, err, "Gagal menghapus data bidang")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseBidangID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID bidang tidak valid")
		return 0, false
	}
	return id, true
}

func decodeBidangPayload(w http.ResponseWriter, r *http.Request) (*bidangPayload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var payload bidangPayload
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

func writeBidangRepoError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "Data bidang tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicateKode):
		writeError(w, http.StatusConflict, "Kode bidang sudah terdaftar")
	case errors.Is(err, repository.ErrInUse):
		writeError(w, http.StatusConflict, "Bidang masih digunakan oleh pegawai")
	default:
		log.Printf("bidang: %v", err)
		writeError(w, http.StatusInternalServerError, fallback)
	}
}

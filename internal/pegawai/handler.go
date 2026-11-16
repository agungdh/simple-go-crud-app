package pegawai

import (
	"encoding/json"
	"net/http"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/httpx"
)

// Handler handles the pegawai endpoints.
type Handler struct {
	service *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// List handles GET /pegawai.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		httpx.WriteServiceError(w, "list pegawai", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope[Response]{Data: items})
}

// Get handles GET /pegawai/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.ParseID(w, r)
	if !ok {
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, "get pegawai", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Envelope[Response]{Data: item})
}

// Create handles POST /pegawai.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.RequireActor(w, r)
	if !ok {
		return
	}

	input, ok := httpx.ReadInput(w, r, decodeInput, validateInput)
	if !ok {
		return
	}

	if err := h.service.Create(r.Context(), input, actor); err != nil {
		httpx.WriteServiceError(w, "create pegawai", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Update handles PUT /pegawai/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.RequireActor(w, r)
	if !ok {
		return
	}

	id, ok := httpx.ParseID(w, r)
	if !ok {
		return
	}

	input, ok := httpx.ReadInput(w, r, decodeInput, validateInput)
	if !ok {
		return
	}

	if err := h.service.Update(r.Context(), id, input, actor); err != nil {
		httpx.WriteServiceError(w, "update pegawai", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Delete handles DELETE /pegawai/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.RequireActor(w, r)
	if !ok {
		return
	}

	id, ok := httpx.ParseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id, actor); err != nil {
		httpx.WriteServiceError(w, "delete pegawai", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeInput maps the wire object to the DTO and collects structural problems
// (unknown fields and wrong JSON types).
func decodeInput(fields map[string]json.RawMessage) (Input, apperr.FieldErrors) {
	errs := apperr.FieldErrors{}
	httpx.RejectUnknown(fields, map[string]struct{}{
		"bidang_id":     {},
		"nama":          {},
		"nip":           {},
		"tanggal_lahir": {},
		"alamat":        {},
		"jenis_kelamin": {},
	}, errs)

	return Input{
		BidangID:     httpx.DecodeInt64(fields, "bidang_id", "Bidang ID", errs),
		Nama:         httpx.DecodeString(fields, "nama", "Nama", errs),
		NIP:          httpx.DecodeString(fields, "nip", "NIP", errs),
		TanggalLahir: httpx.DecodeString(fields, "tanggal_lahir", "Tanggal lahir", errs),
		Alamat:       httpx.DecodeString(fields, "alamat", "Alamat", errs),
		JenisKelamin: JenisKelaminInput(httpx.DecodeString(fields, "jenis_kelamin", "Jenis kelamin", errs)),
	}, errs
}

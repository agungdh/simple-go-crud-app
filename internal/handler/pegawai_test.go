package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"simple-go-crud-app/internal/database"
	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/internal/service"
)

type pegawaiData struct {
	ID           int64  `json:"id"`
	BidangID     int64  `json:"bidang_id"`
	Nama         string `json:"nama"`
	NIP          string `json:"nip"`
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	JenisKelamin string `json:"jenis_kelamin"`
}

type apiResponse struct {
	Data  pegawaiData `json:"data"`
	Error string      `json:"error"`
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	if err := database.Migrate(context.Background(), db, os.DirFS("../.."), "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	bidangRepo := repository.NewBidangRepository(db)
	bidangSvc := service.NewBidangService(bidangRepo)

	repo := repository.NewPegawaiRepository(db)
	svc := service.NewPegawaiService(repo)
	return router.New(handler.NewPegawaiHandler(svc), handler.NewBidangHandler(bidangSvc))
}

func doRequest(t *testing.T, srv http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var res apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return res
}

func validPayload(bidangID int64) map[string]any {
	return map[string]any{
		"bidang_id":     bidangID,
		"nama":          "Budi Santoso",
		"nip":           "199001012020011001",
		"tanggal_lahir": "1990-01-01",
		"alamat":        "Jl. Merdeka No. 1, Jakarta",
		"jenis_kelamin": "pria",
	}
}

func createPegawai(t *testing.T, srv http.Handler) pegawaiData {
	t.Helper()

	bidang := createBidang(t, srv)
	rec := doRequest(t, srv, http.MethodPost, "/pegawai", validPayload(bidang.ID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body.String())
	}
	return decodeResponse(t, rec).Data
}

func TestPegawaiCRUD(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)
	if created.ID <= 0 {
		t.Fatalf("created id = %d, want > 0", created.ID)
	}
	if created.Nama != "Budi Santoso" || created.NIP != "199001012020011001" {
		t.Fatalf("unexpected created pegawai: %+v", created)
	}
	if created.TanggalLahir != "01-01-1990" {
		t.Fatalf("tanggal_lahir = %q, want 01-01-1990", created.TanggalLahir)
	}
	if created.JenisKelamin != "Pria" {
		t.Fatalf("jenis_kelamin = %q, want Pria", created.JenisKelamin)
	}

	rec := doRequest(t, srv, http.MethodGet, "/pegawai", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Data []pegawaiData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Data) != 1 {
		t.Fatalf("list length = %d, want 1", len(list.Data))
	}

	if created.BidangID <= 0 {
		t.Fatalf("created bidang_id = %d, want > 0", created.BidangID)
	}

	path := "/pegawai/" + itoa(created.ID)

	rec = doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := decodeResponse(t, rec).Data; got.ID != created.ID {
		t.Fatalf("get id = %d, want %d", got.ID, created.ID)
	}

	updatePayload := validPayload(created.BidangID)
	updatePayload["nama"] = "Budi Santoso Baru"
	updatePayload["tanggal_lahir"] = "03-02-1991"
	updatePayload["jenis_kelamin"] = "wanita"
	rec = doRequest(t, srv, http.MethodPut, path, updatePayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated := decodeResponse(t, rec).Data
	if updated.Nama != "Budi Santoso Baru" || updated.TanggalLahir != "03-02-1991" {
		t.Fatalf("unexpected updated pegawai: %+v", updated)
	}
	if updated.JenisKelamin != "Wanita" {
		t.Fatalf("jenis_kelamin = %q, want Wanita", updated.JenisKelamin)
	}

	rec = doRequest(t, srv, http.MethodDelete, path, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("delete body = %q, want empty", rec.Body.String())
	}

	rec = doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", rec.Code)
	}
}

func TestCreatePegawaiValidation(t *testing.T) {
	srv := newTestServer(t)
	bidang := createBidang(t, srv)

	tests := []struct {
		name       string
		mutate     func(map[string]any)
		wantStatus int
	}{
		{
			name:       "missing nama",
			mutate:     func(p map[string]any) { p["nama"] = "  " },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid tanggal lahir",
			mutate:     func(p map[string]any) { p["tanggal_lahir"] = "1990/01/01" },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "future tanggal lahir",
			mutate:     func(p map[string]any) { p["tanggal_lahir"] = "2999-01-01" },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing jenis_kelamin",
			mutate:     func(p map[string]any) { delete(p, "jenis_kelamin") },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid jenis_kelamin",
			mutate:     func(p map[string]any) { p["jenis_kelamin"] = "X" },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid jenis_kelamin type",
			mutate:     func(p map[string]any) { p["jenis_kelamin"] = 1 },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing bidang_id",
			mutate:     func(p map[string]any) { delete(p, "bidang_id") },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "unknown field",
			mutate:     func(p map[string]any) { p["created_at"] = 123 },
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validPayload(bidang.ID)
			tt.mutate(payload)

			rec := doRequest(t, srv, http.MethodPost, "/pegawai", payload)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if decodeResponse(t, rec).Error == "" {
				t.Fatal("error message is empty")
			}
		})
	}
}

func TestCreatePegawaiUnknownBidang(t *testing.T) {
	srv := newTestServer(t)

	rec := doRequest(t, srv, http.MethodPost, "/pegawai", validPayload(999))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPegawaiNIPConflict(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)

	rec := doRequest(t, srv, http.MethodPost, "/pegawai", validPayload(created.BidangID))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}

	secondPayload := validPayload(created.BidangID)
	secondPayload["nip"] = "199001012020011002"
	secondPayload["nama"] = "Siti Aminah"
	rec = doRequest(t, srv, http.MethodPost, "/pegawai", secondPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, body = %s", rec.Code, rec.Body.String())
	}
	second := decodeResponse(t, rec).Data

	updatePayload := validPayload(created.BidangID)
	updatePayload["nip"] = created.NIP
	rec = doRequest(t, srv, http.MethodPut, "/pegawai/"+itoa(second.ID), updatePayload)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate update status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPegawaiNIPReusableAfterDelete(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)

	rec := doRequest(t, srv, http.MethodDelete, "/pegawai/"+itoa(created.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, srv, http.MethodPost, "/pegawai", validPayload(created.BidangID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("recreate status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPegawaiNotFound(t *testing.T) {
	srv := newTestServer(t)
	bidang := createBidang(t, srv)

	rec := doRequest(t, srv, http.MethodGet, "/pegawai/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get status = %d, want 404", rec.Code)
	}

	rec = doRequest(t, srv, http.MethodPut, "/pegawai/999", validPayload(bidang.ID))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("update status = %d, want 404", rec.Code)
	}

	rec = doRequest(t, srv, http.MethodDelete, "/pegawai/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete status = %d, want 404", rec.Code)
	}
}

func TestPegawaiInvalidID(t *testing.T) {
	srv := newTestServer(t)

	rec := doRequest(t, srv, http.MethodGet, "/pegawai/abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPegawaiInvalidJSON(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/pegawai", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func listPegawai(t *testing.T, srv http.Handler, query string) []pegawaiData {
	t.Helper()

	path := "/pegawai"
	if query != "" {
		path += "?q=" + url.QueryEscape(query)
	}

	rec := doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %q status = %d, body = %s", query, rec.Code, rec.Body.String())
	}

	var list struct {
		Data []pegawaiData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return list.Data
}

func TestPegawaiSearch(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)

	second := validPayload(created.BidangID)
	second["nama"] = "Siti Aminah"
	second["nip"] = "199001012020011002"
	second["tanggal_lahir"] = "1985-05-17"
	second["alamat"] = "Jl. Melati No. 2, Bandung"
	second["jenis_kelamin"] = "wanita"
	rec := doRequest(t, srv, http.MethodPost, "/pegawai", second)
	if rec.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, body = %s", rec.Code, rec.Body.String())
	}

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{name: "matches nama", query: "budi", want: 1},
		{name: "matches partial alamat", query: "merdeka", want: 1},
		{name: "matches tanggal lahir", query: "17-05-1985", want: 1},
		{name: "matches partial tanggal lahir", query: "05-1985", want: 1},
		{name: "matches pria", query: "pria", want: 1},
		{name: "matches wanita", query: "wanita", want: 1},
		{name: "case insensitive", query: "BUDI", want: 1},
		{name: "no match", query: "zzz", want: 0},
		{name: "fts operators are literal", query: "budi AND siti", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(listPegawai(t, srv, tt.query)); got != tt.want {
				t.Fatalf("search %q returned %d rows, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestPegawaiSearchHidesInternalColumns(t *testing.T) {
	srv := newTestServer(t)
	created := createPegawai(t, srv)

	rec := doRequest(t, srv, http.MethodGet, "/pegawai?q=budi", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(list.Data))
	}
	assertInternalColumnsHidden(t, list.Data[0])

	rec = doRequest(t, srv, http.MethodGet, "/pegawai/"+itoa(created.ID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	assertInternalColumnsHidden(t, detail.Data)
}

// assertInternalColumnsHidden guards the DTO: search_text, the stored boolean,
// and audit columns must never reach the client.
func assertInternalColumnsHidden(t *testing.T, row map[string]any) {
	t.Helper()

	hidden := []string{
		"search_text",
		"is_male",
		"created_at", "created_by",
		"updated_at", "updated_by",
		"deleted_at", "deleted_by",
	}
	for _, column := range hidden {
		if _, exposed := row[column]; exposed {
			t.Fatalf("column %q must not be exposed in the response", column)
		}
	}
	if got := row["jenis_kelamin"]; got != "Pria" {
		t.Fatalf("jenis_kelamin = %v, want Pria", got)
	}
}

func TestPegawaiSearchReindexedOnUpdate(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)

	updatePayload := validPayload(created.BidangID)
	updatePayload["alamat"] = "Jl. Kenanga No. 9, Bogor"
	rec := doRequest(t, srv, http.MethodPut, "/pegawai/"+itoa(created.ID), updatePayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := len(listPegawai(t, srv, "kenanga")); got != 1 {
		t.Fatalf("search new alamat returned %d rows, want 1", got)
	}
	if got := len(listPegawai(t, srv, "merdeka")); got != 0 {
		t.Fatalf("search old alamat returned %d rows, want 0", got)
	}
}

func TestPegawaiSearchExcludesDeleted(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)

	rec := doRequest(t, srv, http.MethodDelete, "/pegawai/"+itoa(created.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := len(listPegawai(t, srv, "budi")); got != 0 {
		t.Fatalf("search after delete returned %d rows, want 0", got)
	}
}

func TestPegawaiSearchTooShort(t *testing.T) {
	srv := newTestServer(t)
	createPegawai(t, srv)

	rec := doRequest(t, srv, http.MethodGet, "/pegawai?q="+url.QueryEscape("si"), nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body = %s", rec.Code, rec.Body.String())
	}
	if decodeResponse(t, rec).Error == "" {
		t.Fatal("error message is empty")
	}

	// Blank input means "no filter" and must still list everything.
	if got := len(listPegawai(t, srv, "   ")); got != 1 {
		t.Fatalf("blank query returned %d rows, want 1", got)
	}
}

func TestPegawaiSearchTooLong(t *testing.T) {
	srv := newTestServer(t)

	rec := doRequest(t, srv, http.MethodGet, "/pegawai?q="+url.QueryEscape(strings.Repeat("a", 101)), nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body = %s", rec.Code, rec.Body.String())
	}
	if decodeResponse(t, rec).Error == "" {
		t.Fatal("error message is empty")
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

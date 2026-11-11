package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"simple-go-crud-app/internal/database"
	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/internal/service"
)

type pegawaiData struct {
	ID           int64  `json:"id"`
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

	repo := repository.NewPegawaiRepository(db)
	svc := service.NewPegawaiService(repo)
	return router.New(handler.NewPegawaiHandler(svc))
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

func validPayload() map[string]any {
	return map[string]any{
		"nama":          "Budi Santoso",
		"nip":           "199001012020011001",
		"tanggal_lahir": "1990-01-01",
		"alamat":        "Jl. Merdeka No. 1, Jakarta",
		"jenis_kelamin": "L",
	}
}

func createPegawai(t *testing.T, srv http.Handler) pegawaiData {
	t.Helper()

	rec := doRequest(t, srv, http.MethodPost, "/pegawai", validPayload())
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
	if created.TanggalLahir != "1990-01-01" {
		t.Fatalf("tanggal_lahir = %q, want 1990-01-01", created.TanggalLahir)
	}
	if created.JenisKelamin != "L" {
		t.Fatalf("jenis_kelamin = %q, want L", created.JenisKelamin)
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

	path := "/pegawai/" + itoa(created.ID)

	rec = doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := decodeResponse(t, rec).Data; got.ID != created.ID {
		t.Fatalf("get id = %d, want %d", got.ID, created.ID)
	}

	updatePayload := validPayload()
	updatePayload["nama"] = "Budi Santoso Baru"
	updatePayload["tanggal_lahir"] = "1991-02-03"
	updatePayload["jenis_kelamin"] = "perempuan"
	rec = doRequest(t, srv, http.MethodPut, path, updatePayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated := decodeResponse(t, rec).Data
	if updated.Nama != "Budi Santoso Baru" || updated.TanggalLahir != "1991-02-03" {
		t.Fatalf("unexpected updated pegawai: %+v", updated)
	}
	if updated.JenisKelamin != "P" {
		t.Fatalf("jenis_kelamin = %q, want P", updated.JenisKelamin)
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
			mutate:     func(p map[string]any) { p["tanggal_lahir"] = "01-01-1990" },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "future tanggal lahir",
			mutate:     func(p map[string]any) { p["tanggal_lahir"] = "2999-01-01" },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid jenis kelamin",
			mutate:     func(p map[string]any) { p["jenis_kelamin"] = "X" },
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
			payload := validPayload()
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

func TestPegawaiNIPConflict(t *testing.T) {
	srv := newTestServer(t)

	created := createPegawai(t, srv)

	rec := doRequest(t, srv, http.MethodPost, "/pegawai", validPayload())
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}

	secondPayload := validPayload()
	secondPayload["nip"] = "199001012020011002"
	secondPayload["nama"] = "Siti Aminah"
	rec = doRequest(t, srv, http.MethodPost, "/pegawai", secondPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, body = %s", rec.Code, rec.Body.String())
	}
	second := decodeResponse(t, rec).Data

	updatePayload := validPayload()
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

	rec = doRequest(t, srv, http.MethodPost, "/pegawai", validPayload())
	if rec.Code != http.StatusCreated {
		t.Fatalf("recreate status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPegawaiNotFound(t *testing.T) {
	srv := newTestServer(t)

	rec := doRequest(t, srv, http.MethodGet, "/pegawai/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get status = %d, want 404", rec.Code)
	}

	rec = doRequest(t, srv, http.MethodPut, "/pegawai/999", validPayload())
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

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

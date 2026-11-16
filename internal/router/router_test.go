package router_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/dto"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/internal/testutil"
)

type testEnv struct {
	t       *testing.T
	db      *bun.DB
	handler http.Handler
}

const testOrigin = "http://127.0.0.1:8082"

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testutil.OpenDB(t)
	return &testEnv{t: t, db: db, handler: router.New(db, []string{testOrigin})}
}

func (e *testEnv) do(method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.doWithHeaders(method, path, body, nil)
}

func (e *testEnv) doWithHeaders(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) createBidang(name string) int64 {
	e.t.Helper()

	rec := e.do(http.MethodPost, "/bidang", fmt.Sprintf(`{"bidang":%q}`, name))
	requireStatus(e.t, rec, http.StatusCreated)

	rec = e.do(http.MethodGet, "/bidang", "")
	requireStatus(e.t, rec, http.StatusOK)

	list := decodeJSON[dto.BidangListResponse](e.t, rec)
	for _, item := range list.Data {
		if item.Bidang == name {
			return item.ID
		}
	}
	e.t.Fatalf("created bidang %q not found", name)
	return 0
}

func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}

func requireEmptyBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return value
}

func decodeValidations(t *testing.T, rec *httptest.ResponseRecorder) map[string][]string {
	t.Helper()

	body := decodeJSON[dto.ValidationErrorResponse](t, rec)
	return body.Error.Validations
}

func requireValidationFields(t *testing.T, rec *httptest.ResponseRecorder, fields ...string) map[string][]string {
	t.Helper()

	requireStatus(t, rec, http.StatusUnprocessableEntity)
	validations := decodeValidations(t, rec)
	for _, field := range fields {
		messages, ok := validations[field]
		if !ok {
			t.Fatalf("field %q missing from validations: %v", field, validations)
		}
		if len(messages) == 0 {
			t.Fatalf("field %q has no messages", field)
		}
		for _, message := range messages {
			if message == "" {
				t.Fatalf("field %q has an empty message", field)
			}
		}
	}
	return validations
}

func TestCORS(t *testing.T) {
	env := newTestEnv(t)

	t.Run("actual request from docs origin", func(t *testing.T) {
		rec := env.doWithHeaders(http.MethodGet, "/bidang", "", map[string]string{
			"Origin": testOrigin,
		})
		requireStatus(t, rec, http.StatusOK)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, testOrigin)
		}
		if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
			t.Errorf("Vary = %q, want it to contain Origin", got)
		}
	})

	t.Run("error response keeps CORS headers", func(t *testing.T) {
		rec := env.doWithHeaders(http.MethodGet, "/bidang/9999", "", map[string]string{
			"Origin": testOrigin,
		})
		requireStatus(t, rec, http.StatusNotFound)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, testOrigin)
		}
	})

	t.Run("preflight", func(t *testing.T) {
		rec := env.doWithHeaders(http.MethodOptions, "/bidang", "", map[string]string{
			"Origin":                         testOrigin,
			"Access-Control-Request-Method":  http.MethodPost,
			"Access-Control-Request-Headers": "content-type",
		})
		requireStatus(t, rec, http.StatusNoContent)
		requireEmptyBody(t, rec)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, testOrigin)
		}
		if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
			t.Errorf("Access-Control-Allow-Methods = %q, want it to contain POST", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
			t.Errorf("Access-Control-Allow-Headers = %q, want it to contain Content-Type", got)
		}
	})

	t.Run("disallowed origin", func(t *testing.T) {
		rec := env.doWithHeaders(http.MethodGet, "/bidang", "", map[string]string{
			"Origin": "http://evil.example",
		})
		requireStatus(t, rec, http.StatusOK)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
		}
	})

	t.Run("no origin", func(t *testing.T) {
		rec := env.do(http.MethodGet, "/bidang", "")
		requireStatus(t, rec, http.StatusOK)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
		}
	})
}

func TestBidangCRUD(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(http.MethodGet, "/bidang", "")
	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("empty list body = %q, want data to marshal as []", rec.Body.String())
	}

	rec = env.do(http.MethodPost, "/bidang", `{"bidang":"  Keuangan  "}`)
	requireStatus(t, rec, http.StatusCreated)
	requireEmptyBody(t, rec)

	rec = env.do(http.MethodGet, "/bidang", "")
	requireStatus(t, rec, http.StatusOK)
	list := decodeJSON[dto.BidangListResponse](t, rec)
	if len(list.Data) != 1 {
		t.Fatalf("list length = %d, want 1", len(list.Data))
	}
	if list.Data[0].Bidang != "Keuangan" {
		t.Errorf("bidang = %q, want trimmed %q", list.Data[0].Bidang, "Keuangan")
	}
	id := list.Data[0].ID

	rec = env.do(http.MethodGet, fmt.Sprintf("/bidang/%d", id), "")
	requireStatus(t, rec, http.StatusOK)
	detail := decodeJSON[dto.BidangResponse](t, rec)
	if detail.Data.ID != id || detail.Data.Bidang != "Keuangan" {
		t.Errorf("detail = %+v, want id %d and bidang Keuangan", detail.Data, id)
	}

	rec = env.do(http.MethodPut, fmt.Sprintf("/bidang/%d", id), `{"bidang":"Keuangan Pusat"}`)
	requireStatus(t, rec, http.StatusOK)
	requireEmptyBody(t, rec)

	rec = env.do(http.MethodGet, fmt.Sprintf("/bidang/%d", id), "")
	requireStatus(t, rec, http.StatusOK)
	detail = decodeJSON[dto.BidangResponse](t, rec)
	if detail.Data.Bidang != "Keuangan Pusat" {
		t.Errorf("bidang after update = %q, want %q", detail.Data.Bidang, "Keuangan Pusat")
	}

	rec = env.do(http.MethodGet, "/bidang?q=Keuangan", "")
	requireStatus(t, rec, http.StatusOK)
	if got := len(decodeJSON[dto.BidangListResponse](t, rec).Data); got != 1 {
		t.Errorf("search hits = %d, want 1", got)
	}

	rec = env.do(http.MethodGet, "/bidang?q=tidakada", "")
	requireStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("search miss body = %q, want data to marshal as []", rec.Body.String())
	}

	rec = env.do(http.MethodDelete, fmt.Sprintf("/bidang/%d", id), "")
	requireStatus(t, rec, http.StatusNoContent)
	requireEmptyBody(t, rec)

	rec = env.do(http.MethodGet, fmt.Sprintf("/bidang/%d", id), "")
	requireStatus(t, rec, http.StatusNotFound)
	notFound := decodeJSON[dto.ErrorResponse](t, rec)
	if notFound.Error == "" {
		t.Error("404 response has an empty error message")
	}

	rec = env.do(http.MethodDelete, fmt.Sprintf("/bidang/%d", id), "")
	requireStatus(t, rec, http.StatusNotFound)

	// The partial unique index only covers active rows, so the same name can be
	// created again after a soft delete.
	rec = env.do(http.MethodPost, "/bidang", `{"bidang":"Keuangan Pusat"}`)
	requireStatus(t, rec, http.StatusCreated)
}

func TestBidangBadRequestBody(t *testing.T) {
	env := newTestEnv(t)

	tests := []struct {
		name string
		body string
	}{
		{name: "empty", body: ""},
		{name: "malformed", body: "{"},
		{name: "array", body: "[]"},
		{name: "null", body: "null"},
		{name: "scalar", body: `"teks"`},
		{name: "two values", body: `{} {}`},
		{name: "trailing garbage", body: `{"bidang":"ok"} x`},
		{name: "over one mebibyte", body: `{"bidang":"` + strings.Repeat("a", 1<<20) + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := env.do(http.MethodPost, "/bidang", test.body)
			requireStatus(t, rec, http.StatusBadRequest)
			body := decodeJSON[dto.ErrorResponse](t, rec)
			if body.Error == "" {
				t.Error("400 response has an empty error message")
			}
		})
	}
}

func TestBidangValidation(t *testing.T) {
	env := newTestEnv(t)

	t.Run("missing", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{}`)
		requireValidationFields(t, rec, "bidang")
	})

	t.Run("whitespace only", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{"bidang":"   "}`)
		requireValidationFields(t, rec, "bidang")
	})

	t.Run("too long", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{"bidang":"`+strings.Repeat("a", 151)+`"}`)
		requireValidationFields(t, rec, "bidang")
	})

	t.Run("unknown field", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{"bidang":"ok","lain":"x"}`)
		requireValidationFields(t, rec, "lain")
	})

	t.Run("wrong type", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{"bidang":123}`)
		validations := requireValidationFields(t, rec, "bidang")
		if len(validations["bidang"]) != 1 {
			t.Errorf("bidang messages = %v, want exactly one structural message", validations["bidang"])
		}
	})

	t.Run("collects unknown and semantic", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{"lain":"x"}`)
		requireValidationFields(t, rec, "bidang", "lain")
	})
}

func TestBidangDatabaseErrors(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(http.MethodPost, "/bidang", `{"bidang":"SDM"}`)
	requireStatus(t, rec, http.StatusCreated)

	// Duplicate active name is a post-validation error, so the contract maps it
	// to a generic 500 rather than 409.
	rec = env.do(http.MethodPost, "/bidang", `{"bidang":"SDM"}`)
	requireStatus(t, rec, http.StatusInternalServerError)
	body := decodeJSON[dto.ErrorResponse](t, rec)
	if body.Error == "" {
		t.Error("500 response has an empty error message")
	}
}

func TestBidangInvalidID(t *testing.T) {
	env := newTestEnv(t)

	for _, path := range []string{"/bidang/abc", "/bidang/0", "/bidang/-1", "/bidang/1.5"} {
		t.Run(path, func(t *testing.T) {
			rec := env.do(http.MethodGet, path, "")
			requireValidationFields(t, rec, "id")

			rec = env.do(http.MethodPut, path, `{"bidang":"ok"}`)
			requireValidationFields(t, rec, "id")

			rec = env.do(http.MethodDelete, path, "")
			requireValidationFields(t, rec, "id")
		})
	}
}

func TestBidangSearchLength(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(http.MethodGet, "/bidang?q=ke", "")
	requireValidationFields(t, rec, "q")

	rec = env.do(http.MethodGet, "/bidang?q="+strings.Repeat("a", 101), "")
	requireValidationFields(t, rec, "q")

	rec = env.do(http.MethodGet, "/bidang?q="+strings.Repeat("a", 100), "")
	requireStatus(t, rec, http.StatusOK)

	rec = env.do(http.MethodGet, "/bidang?q="+strings.Repeat("a", 3), "")
	requireStatus(t, rec, http.StatusOK)
}

func TestBidangNotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(http.MethodGet, "/bidang/9999", "")
	requireStatus(t, rec, http.StatusNotFound)

	rec = env.do(http.MethodPut, "/bidang/9999", `{"bidang":"ok"}`)
	requireStatus(t, rec, http.StatusNotFound)

	rec = env.do(http.MethodDelete, "/bidang/9999", "")
	requireStatus(t, rec, http.StatusNotFound)
}

func pegawaiBody(bidangID int64, nama, nip, tanggalLahir, alamat, jenisKelamin string) string {
	return fmt.Sprintf(
		`{"bidang_id":%d,"nama":%q,"nip":%q,"tanggal_lahir":%q,"alamat":%q,"jenis_kelamin":%q}`,
		bidangID, nama, nip, tanggalLahir, alamat, jenisKelamin,
	)
}

func TestPegawaiCRUD(t *testing.T) {
	env := newTestEnv(t)
	bidangID := env.createBidang("Keuangan")

	rec := env.do(http.MethodPost, "/pegawai", pegawaiBody(
		bidangID, "Budi Santoso", "1987654321", "17-08-1990", "Jl. Merdeka 1", "pria",
	))
	requireStatus(t, rec, http.StatusCreated)
	requireEmptyBody(t, rec)

	rec = env.do(http.MethodGet, "/pegawai", "")
	requireStatus(t, rec, http.StatusOK)
	list := decodeJSON[dto.PegawaiListResponse](t, rec)
	if len(list.Data) != 1 {
		t.Fatalf("list length = %d, want 1", len(list.Data))
	}
	item := list.Data[0]
	if item.TanggalLahir != "17-08-1990" {
		t.Errorf("tanggal_lahir = %q, want %q", item.TanggalLahir, "17-08-1990")
	}
	if item.JenisKelamin != dto.Pria {
		t.Errorf("jenis_kelamin = %q, want %q", item.JenisKelamin, dto.Pria)
	}
	if item.BidangID != bidangID || item.Nama != "Budi Santoso" || item.NIP != "1987654321" {
		t.Errorf("unexpected pegawai: %+v", item)
	}
	id := item.ID

	rec = env.do(http.MethodGet, fmt.Sprintf("/pegawai/%d", id), "")
	requireStatus(t, rec, http.StatusOK)
	detail := decodeJSON[dto.PegawaiResponse](t, rec)
	if detail.Data.ID != id {
		t.Errorf("detail id = %d, want %d", detail.Data.ID, id)
	}

	for _, q := range []string{"Budi", "1987654321", "17-08-1990", "pria"} {
		rec = env.do(http.MethodGet, "/pegawai?q="+q, "")
		requireStatus(t, rec, http.StatusOK)
		hits := decodeJSON[dto.PegawaiListResponse](t, rec)
		if len(hits.Data) != 1 {
			t.Errorf("search %q hits = %d, want 1", q, len(hits.Data))
		}
	}

	rec = env.do(http.MethodPut, fmt.Sprintf("/pegawai/%d", id), pegawaiBody(
		bidangID, "Budi Santoso", "1987654321", "1990-08-17", "Jl. Merdeka 2", "WANITA",
	))
	requireStatus(t, rec, http.StatusOK)
	requireEmptyBody(t, rec)

	rec = env.do(http.MethodGet, fmt.Sprintf("/pegawai/%d", id), "")
	requireStatus(t, rec, http.StatusOK)
	detail = decodeJSON[dto.PegawaiResponse](t, rec)
	if detail.Data.TanggalLahir != "17-08-1990" {
		t.Errorf("tanggal_lahir after update = %q, want DD-MM-YYYY", detail.Data.TanggalLahir)
	}
	if detail.Data.JenisKelamin != dto.Wanita {
		t.Errorf("jenis_kelamin after update = %q, want %q", detail.Data.JenisKelamin, dto.Wanita)
	}
	if detail.Data.Alamat != "Jl. Merdeka 2" {
		t.Errorf("alamat after update = %q", detail.Data.Alamat)
	}

	rec = env.do(http.MethodDelete, fmt.Sprintf("/pegawai/%d", id), "")
	requireStatus(t, rec, http.StatusNoContent)
	requireEmptyBody(t, rec)

	rec = env.do(http.MethodGet, fmt.Sprintf("/pegawai/%d", id), "")
	requireStatus(t, rec, http.StatusNotFound)

	rec = env.do(http.MethodDelete, fmt.Sprintf("/pegawai/%d", id), "")
	requireStatus(t, rec, http.StatusNotFound)
}

func TestPegawaiValidation(t *testing.T) {
	env := newTestEnv(t)
	bidangID := env.createBidang("Keuangan")

	future := time.Now().AddDate(0, 0, 1).Format("02-01-2006")

	tests := []struct {
		name   string
		body   string
		fields []string
	}{
		{
			name:   "all required missing",
			body:   `{}`,
			fields: []string{"bidang_id", "nama", "nip", "tanggal_lahir", "alamat", "jenis_kelamin"},
		},
		{
			name:   "all empty strings",
			body:   `{"bidang_id":0,"nama":"","nip":"","tanggal_lahir":"","alamat":"","jenis_kelamin":""}`,
			fields: []string{"bidang_id", "nama", "nip", "tanggal_lahir", "alamat", "jenis_kelamin"},
		},
		{
			name:   "negative bidang",
			body:   pegawaiBody(-1, "Budi", "1", "17-08-1990", "Jl. Merdeka", "pria"),
			fields: []string{"bidang_id"},
		},
		{
			name:   "impossible date",
			body:   pegawaiBody(bidangID, "Budi", "1", "31-02-2020", "Jl. Merdeka", "pria"),
			fields: []string{"tanggal_lahir"},
		},
		{
			name:   "bad date format",
			body:   pegawaiBody(bidangID, "Budi", "1", "2020/02/02", "Jl. Merdeka", "pria"),
			fields: []string{"tanggal_lahir"},
		},
		{
			name:   "future date",
			body:   pegawaiBody(bidangID, "Budi", "1", future, "Jl. Merdeka", "pria"),
			fields: []string{"tanggal_lahir"},
		},
		{
			name:   "bad gender",
			body:   pegawaiBody(bidangID, "Budi", "1", "17-08-1990", "Jl. Merdeka", "lainnya"),
			fields: []string{"jenis_kelamin"},
		},
		{
			name:   "nama too long",
			body:   pegawaiBody(bidangID, strings.Repeat("a", 151), "1", "17-08-1990", "Jl. Merdeka", "pria"),
			fields: []string{"nama"},
		},
		{
			name:   "nip too long",
			body:   pegawaiBody(bidangID, "Budi", strings.Repeat("1", 51), "17-08-1990", "Jl. Merdeka", "pria"),
			fields: []string{"nip"},
		},
		{
			name:   "alamat too long",
			body:   pegawaiBody(bidangID, "Budi", "1", "17-08-1990", strings.Repeat("a", 501), "pria"),
			fields: []string{"alamat"},
		},
		{
			name:   "unknown field",
			body:   `{"bidang_id":1,"nama":"Budi","nip":"1","tanggal_lahir":"17-08-1990","alamat":"Jl. Merdeka","jenis_kelamin":"pria","lain":"x"}`,
			fields: []string{"lain"},
		},
		{
			name:   "wrong types",
			body:   `{"bidang_id":"1","nama":12,"nip":false,"tanggal_lahir":[],"alamat":{},"jenis_kelamin":1}`,
			fields: []string{"bidang_id", "nama", "nip", "tanggal_lahir", "alamat", "jenis_kelamin"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := env.do(http.MethodPost, "/pegawai", test.body)
			requireValidationFields(t, rec, test.fields...)

			// PUT must validate the same way before checking existence.
			rec = env.do(http.MethodPut, "/pegawai/9999", test.body)
			requireValidationFields(t, rec, test.fields...)
		})
	}

	t.Run("wrong type has no duplicate required message", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/pegawai", `{"bidang_id":"1"}`)
		validations := requireValidationFields(t, rec, "bidang_id")
		if len(validations["bidang_id"]) != 1 {
			t.Errorf("bidang_id messages = %v, want exactly one structural message", validations["bidang_id"])
		}
	})
}

func TestPegawaiDatabaseErrors(t *testing.T) {
	env := newTestEnv(t)
	bidangID := env.createBidang("Keuangan")

	rec := env.do(http.MethodPost, "/pegawai", pegawaiBody(
		999999, "Budi", "1", "17-08-1990", "Jl. Merdeka", "pria",
	))
	requireStatus(t, rec, http.StatusInternalServerError)

	rec = env.do(http.MethodPost, "/pegawai", pegawaiBody(
		bidangID, "Budi", "1", "17-08-1990", "Jl. Merdeka", "pria",
	))
	requireStatus(t, rec, http.StatusCreated)

	// Duplicate NIP is a post-validation error, so it maps to 500.
	rec = env.do(http.MethodPost, "/pegawai", pegawaiBody(
		bidangID, "Ani", "1", "17-08-1990", "Jl. Merdeka", "wanita",
	))
	requireStatus(t, rec, http.StatusInternalServerError)
}

func TestPegawaiSearchLength(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(http.MethodGet, "/pegawai?q=ab", "")
	requireValidationFields(t, rec, "q")

	rec = env.do(http.MethodGet, "/pegawai?q="+strings.Repeat("a", 101), "")
	requireValidationFields(t, rec, "q")
}

func TestPegawaiNotFound(t *testing.T) {
	env := newTestEnv(t)
	bidangID := env.createBidang("Keuangan")

	rec := env.do(http.MethodGet, "/pegawai/9999", "")
	requireStatus(t, rec, http.StatusNotFound)

	rec = env.do(http.MethodPut, "/pegawai/9999", pegawaiBody(
		bidangID, "Budi", "1", "17-08-1990", "Jl. Merdeka", "pria",
	))
	requireStatus(t, rec, http.StatusNotFound)

	rec = env.do(http.MethodDelete, "/pegawai/9999", "")
	requireStatus(t, rec, http.StatusNotFound)
}

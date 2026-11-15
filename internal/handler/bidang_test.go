package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type bidangData struct {
	ID     int64  `json:"id"`
	Bidang string `json:"bidang"`
}

func createBidangNamed(t *testing.T, srv http.Handler, name string) bidangData {
	t.Helper()

	rec := doRequest(t, srv, http.MethodPost, "/bidang", map[string]any{"bidang": name})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create bidang status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Data bidangData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode bidang: %v", err)
	}
	return res.Data
}

// createBidang creates a bidang with a unique name derived from the test name.
func createBidang(t *testing.T, srv http.Handler) bidangData {
	t.Helper()
	return createBidangNamed(t, srv, "Bidang "+t.Name())
}

func listBidang(t *testing.T, srv http.Handler, query string) []bidangData {
	t.Helper()

	path := "/bidang"
	if query != "" {
		path += "?q=" + url.QueryEscape(query)
	}

	rec := doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list bidang %q status = %d, body = %s", query, rec.Code, rec.Body.String())
	}

	var list struct {
		Data []bidangData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode bidang list: %v", err)
	}
	return list.Data
}

func TestBidangCRUD(t *testing.T) {
	srv := newTestServer(t)

	created := createBidangNamed(t, srv, "Keuangan")
	if created.ID <= 0 {
		t.Fatalf("created id = %d, want > 0", created.ID)
	}
	if created.Bidang != "Keuangan" {
		t.Fatalf("created bidang = %q, want Keuangan", created.Bidang)
	}

	if got := listBidang(t, srv, ""); len(got) != 1 {
		t.Fatalf("list length = %d, want 1", len(got))
	}

	path := "/bidang/" + itoa(created.ID)

	rec := doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, srv, http.MethodPut, path, map[string]any{"bidang": "Keuangan dan Aset"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Data bidangData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated bidang: %v", err)
	}
	if updated.Data.Bidang != "Keuangan dan Aset" {
		t.Fatalf("updated bidang = %q, want Keuangan dan Aset", updated.Data.Bidang)
	}

	rec = doRequest(t, srv, http.MethodDelete, path, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, srv, http.MethodGet, path, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", rec.Code)
	}
}

func TestBidangValidation(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "blank bidang",
			body:       map[string]any{"bidang": "   "},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "bidang too long",
			body:       map[string]any{"bidang": strings.Repeat("a", 151)},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "unknown field",
			body:       map[string]any{"bidang": "Keuangan", "created_at": 1},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, srv, http.MethodPost, "/bidang", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if decodeResponse(t, rec).Error == "" {
				t.Fatal("error message is empty")
			}
		})
	}
}

func TestBidangInvalidJSON(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/bidang", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBidangConflict(t *testing.T) {
	srv := newTestServer(t)

	created := createBidangNamed(t, srv, "Keuangan")

	rec := doRequest(t, srv, http.MethodPost, "/bidang", map[string]any{"bidang": "Keuangan"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}

	second := createBidangNamed(t, srv, "Sekretariat")
	rec = doRequest(t, srv, http.MethodPut, "/bidang/"+itoa(second.ID), map[string]any{"bidang": "Keuangan"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate update status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}

	if rec := doRequest(t, srv, http.MethodDelete, "/bidang/"+itoa(created.ID), nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec = doRequest(t, srv, http.MethodPost, "/bidang", map[string]any{"bidang": "Keuangan"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("recreate status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}
}

func TestBidangNotFound(t *testing.T) {
	srv := newTestServer(t)

	rec := doRequest(t, srv, http.MethodGet, "/bidang/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get status = %d, want 404", rec.Code)
	}

	rec = doRequest(t, srv, http.MethodPut, "/bidang/999", map[string]any{"bidang": "Keuangan"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("update status = %d, want 404", rec.Code)
	}

	rec = doRequest(t, srv, http.MethodDelete, "/bidang/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete status = %d, want 404", rec.Code)
	}
}

func TestBidangSearch(t *testing.T) {
	srv := newTestServer(t)

	createBidangNamed(t, srv, "Keuangan")
	createBidangNamed(t, srv, "Sekretariat")

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{name: "matches bidang", query: "keuangan", want: 1},
		{name: "case insensitive", query: "KEUANGAN", want: 1},
		{name: "matches partial", query: "sekret", want: 1},
		{name: "no match", query: "zzz", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(listBidang(t, srv, tt.query)); got != tt.want {
				t.Fatalf("search %q returned %d rows, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestBidangSearchValidation(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name  string
		query string
	}{
		{name: "too short", query: "ke"},
		{name: "too long", query: strings.Repeat("a", 101)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, srv, http.MethodGet, "/bidang?q="+url.QueryEscape(tt.query), nil)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBidangSearchReindexedOnUpdate(t *testing.T) {
	srv := newTestServer(t)

	created := createBidangNamed(t, srv, "Keuangan")

	rec := doRequest(t, srv, http.MethodPut, "/bidang/"+itoa(created.ID), map[string]any{"bidang": "Sekretariat"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := len(listBidang(t, srv, "sekret")); got != 1 {
		t.Fatalf("search new bidang returned %d rows, want 1", got)
	}
	if got := len(listBidang(t, srv, "keuangan")); got != 0 {
		t.Fatalf("search old bidang returned %d rows, want 0", got)
	}
}

func TestBidangSearchExcludesDeleted(t *testing.T) {
	srv := newTestServer(t)

	created := createBidangNamed(t, srv, "Keuangan")

	rec := doRequest(t, srv, http.MethodDelete, "/bidang/"+itoa(created.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := len(listBidang(t, srv, "keuangan")); got != 0 {
		t.Fatalf("search after delete returned %d rows, want 0", got)
	}
}

func TestBidangHidesInternalColumns(t *testing.T) {
	srv := newTestServer(t)
	created := createBidangNamed(t, srv, "Keuangan")

	rec := doRequest(t, srv, http.MethodGet, "/bidang/"+itoa(created.ID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	for _, column := range []string{
		"search_text",
		"created_at", "created_by",
		"updated_at", "updated_by",
		"deleted_at", "deleted_by",
	} {
		if _, exposed := detail.Data[column]; exposed {
			t.Fatalf("column %q must not be exposed in the response", column)
		}
	}
}

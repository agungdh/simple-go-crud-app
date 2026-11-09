package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"simple-go-crud-app/internal/database"
	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/web"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()

	renderer, err := render.New(web.FS)
	if err != nil {
		t.Fatalf("render.New: %v", err)
	}

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("database.Migrate: %v", err)
	}

	pegawai := handler.NewPegawai(repository.NewPegawai(db), renderer, pages.Pegawai)
	bidang := handler.NewBidang(repository.NewBidang(db), renderer, pages.Bidang)

	handler, err := router.New(pages.Pages, renderer, web.FS, pegawai, bidang)
	if err != nil {
		t.Fatalf("router.New: %v", err)
	}
	return handler
}

func get(handler http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func send(handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			panic(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type bidangResponse struct {
	Data struct {
		ID   int64  `json:"id"`
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"data"`
}

func seedBidang(t *testing.T, handler http.Handler) int64 {
	t.Helper()

	rec := send(handler, http.MethodPost, "/bidang", map[string]string{
		"kode": "BID-01",
		"nama": "Umum",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed bidang status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var created bidangResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode seed bidang: %v", err)
	}
	return created.Data.ID
}

func TestAllPagesRender(t *testing.T) {
	handler := newHandler(t)

	for _, page := range pages.Pages {
		rec := get(handler, page.Route)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", page.Route, rec.Code)
			continue
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<!DOCTYPE html>") || !strings.Contains(body, "</html>") {
			t.Errorf("%s: response is not a complete HTML document", page.Route)
		}
		if !strings.Contains(body, "<title>"+page.Title+"</title>") {
			t.Errorf("%s: title %q not found", page.Route, page.Title)
		}
	}
}

func TestRootRedirectsToDashboard(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, "/")
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != pages.Dashboard.Route {
		t.Fatalf("location = %q, want %q", loc, pages.Dashboard.Route)
	}

	if rec := get(handler, "/starter.html"); rec.Code != http.StatusNotFound {
		t.Fatalf("starter status = %d, want 404", rec.Code)
	}
}

func TestDashboardPageRenders(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, pages.Dashboard.Route)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Dashboard", `href="/dashboard"`, `href="/pegawai"`, `href="/bidang"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
}

func TestUnknownRouteReturns404Page(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, "/route-that-does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "404 Error") {
		t.Fatal("404 page content not rendered")
	}
}

func TestEmbeddedAssetsAreServed(t *testing.T) {
	handler := newHandler(t)

	// _all-skins.min.css is a regression guard for go:embed's default rule
	// that skips files whose names start with an underscore.
	required := []string{
		"/dist/css/AdminLTE.min.css",
		"/dist/css/skins/_all-skins.min.css",
		"/bower_components/jquery/dist/jquery.min.js",
		"/plugins/datatables/jquery.dataTables.min.js",
		"/plugins/datatables/dataTables.bootstrap.min.js",
		"/plugins/datatables/dataTables.bootstrap.min.css",
		"/plugins/select2/select2.min.js",
		"/plugins/select2/select2.min.css",
	}
	for _, asset := range required {
		if rec := get(handler, asset); rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", asset, rec.Code)
		}
	}

	assetRE := regexp.MustCompile(`\b(?:href|src)="(/(?:dist|plugins|bower_components)/[^"]+)"`)
	seen := make(map[string]bool)
	for _, page := range pages.Pages {
		body := get(handler, page.Route).Body.String()
		for _, match := range assetRE.FindAllStringSubmatch(body, -1) {
			seen[match[1]] = true
		}
	}
	if len(seen) == 0 {
		t.Fatal("no local assets referenced by the rendered pages")
	}

	for asset := range seen {
		path := strings.SplitN(asset, "?", 2)[0]
		if rec := get(handler, path); rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, rec.Code)
		}
	}
}

type pegawaiResponse struct {
	Data struct {
		ID         int64  `json:"id"`
		NIP        string `json:"nip"`
		Nama       string `json:"nama"`
		Jabatan    string `json:"jabatan"`
		BidangID   int64  `json:"bidang_id"`
		BidangNama string `json:"bidang_nama"`
		Email      string `json:"email"`
		Telepon    string `json:"telepon"`
		Alamat     string `json:"alamat"`
	} `json:"data"`
}

func TestPegawaiPageRenders(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, pages.Pegawai.Route)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Data Pegawai", "Tambah Pegawai", "/pegawai", `id="tabel-pegawai"`, "/plugins/datatables/jquery.dataTables.min.js", "serverSide: true", "/pegawai/datatable", `href="/bidang"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
	if strings.Contains(body, "/starter.html") {
		t.Error("body still references the removed starter page")
	}
}

func TestPegawaiFormPages(t *testing.T) {
	handler := newHandler(t)
	bidangID := seedBidang(t, handler)

	create := get(handler, pages.PegawaiCreate.Route)
	if create.Code != http.StatusOK {
		t.Fatalf("create form status = %d, want 200", create.Code)
	}
	createBody := create.Body.String()
	for _, want := range []string{`id="form-pegawai"`, `data-action="/pegawai"`, `data-method="POST"`, `name="bidang_id"`, "/plugins/select2/select2.min.js", "/bidang/search"} {
		if !strings.Contains(createBody, want) {
			t.Errorf("create form does not contain %q", want)
		}
	}

	created := send(handler, http.MethodPost, "/pegawai", map[string]any{
		"nip":       "198001012005011001",
		"nama":      "Budi Santoso",
		"bidang_id": bidangID,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}

	edit := get(handler, "/pegawai/1/edit")
	if edit.Code != http.StatusOK {
		t.Fatalf("edit form status = %d, want 200", edit.Code)
	}
	editBody := edit.Body.String()
	for _, want := range []string{`data-action="/pegawai/1"`, `data-method="PUT"`} {
		if !strings.Contains(editBody, want) {
			t.Errorf("edit form does not contain %q", want)
		}
	}
	if strings.Contains(editBody, `value="Budi Santoso"`) {
		t.Error("edit form must not render pegawai data server-side")
	}

	show := get(handler, "/pegawai/1")
	if show.Code != http.StatusOK {
		t.Fatalf("show status = %d, want 200", show.Code)
	}
	var shown pegawaiResponse
	if err := json.Unmarshal(show.Body.Bytes(), &shown); err != nil {
		t.Fatalf("decode show: %v", err)
	}
	if shown.Data.Nama != "Budi Santoso" || shown.Data.BidangNama != "Umum" {
		t.Fatalf("unexpected show payload: %+v", shown.Data)
	}

	// The edit page is a static shell, so a missing row still renders; the
	// client discovers the 404 when it fetches Show.
	if rec := get(handler, "/pegawai/999/edit"); rec.Code != http.StatusOK {
		t.Fatalf("missing edit form status = %d, want 200", rec.Code)
	}
	if rec := get(handler, "/pegawai/999"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing show status = %d, want 404", rec.Code)
	}
	if rec := get(handler, "/pegawai/abc"); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid show status = %d, want 400", rec.Code)
	}
}

func TestPegawaiCRUDFlow(t *testing.T) {
	handler := newHandler(t)
	bidangID := seedBidang(t, handler)

	create := send(handler, http.MethodPost, "/pegawai", map[string]any{
		"nip":       "198001012005011001",
		"nama":      "Budi Santoso",
		"jabatan":   "Staf",
		"bidang_id": bidangID,
		"email":     "budi@example.com",
		"telepon":   "08123456789",
		"alamat":    "Jakarta",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}

	var created pegawaiResponse
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data.ID == 0 || created.Data.Nama != "Budi Santoso" {
		t.Fatalf("unexpected created row: %+v", created.Data)
	}

	// The same NIP must be rejected.
	duplicate := send(handler, http.MethodPost, "/pegawai", map[string]any{
		"nip":       "198001012005011001",
		"nama":      "Pegawai Lain",
		"bidang_id": bidangID,
	})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", duplicate.Code)
	}

	listRec := get(handler, "/pegawai/datatable?start=0&length=10")
	if listRec.Code != http.StatusOK {
		t.Fatalf("datatable status = %d", listRec.Code)
	}
	var listed datatableResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode datatable: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].NIP != "198001012005011001" {
		t.Fatalf("unexpected list: %+v", listed.Data)
	}

	update := send(handler, http.MethodPut, "/pegawai/1", map[string]any{
		"nip":       "198001012005011001",
		"nama":      "Budi Santoso, S.Kom",
		"jabatan":   "Kepala Seksi",
		"bidang_id": bidangID,
		"email":     "budi@example.com",
		"telepon":   "08123456789",
		"alamat":    "Bandung",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", update.Code, update.Body.String())
	}

	var updated pegawaiResponse
	if err := json.Unmarshal(update.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Data.Nama != "Budi Santoso, S.Kom" || updated.Data.Jabatan != "Kepala Seksi" {
		t.Fatalf("update not persisted: %+v", updated.Data)
	}
	if updated.Data.BidangID != bidangID || updated.Data.BidangNama != "Umum" {
		t.Fatalf("bidang not persisted: %+v", updated.Data)
	}

	if rec := send(handler, http.MethodDelete, "/pegawai/1", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if rec := get(handler, "/pegawai/1"); rec.Code != http.StatusNotFound {
		t.Fatalf("show after delete status = %d, want 404", rec.Code)
	}
}

func TestPegawaiValidation(t *testing.T) {
	handler := newHandler(t)
	bidangID := seedBidang(t, handler)

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing nip", map[string]any{"nama": "Tanpa NIP", "bidang_id": bidangID}, http.StatusUnprocessableEntity},
		{"missing nama", map[string]any{"nip": "123", "bidang_id": bidangID}, http.StatusUnprocessableEntity},
		{"missing bidang", map[string]any{"nip": "123", "nama": "Budi"}, http.StatusUnprocessableEntity},
		{"invalid email", map[string]any{"nip": "123", "nama": "Budi", "bidang_id": bidangID, "email": "bukan-email"}, http.StatusUnprocessableEntity},
		{"invalid json", nil, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tc.body == nil {
				req := httptest.NewRequest(http.MethodPost, "/pegawai", strings.NewReader("{not json"))
				rec = httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			} else {
				rec = send(handler, http.MethodPost, "/pegawai", tc.body)
			}
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	if rec := send(handler, http.MethodPut, "/pegawai/abc", map[string]any{"nip": "123", "nama": "Budi", "bidang_id": bidangID}); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d, want 400", rec.Code)
	}
}

type datatableResponse struct {
	Draw            int   `json:"draw"`
	RecordsTotal    int64 `json:"recordsTotal"`
	RecordsFiltered int64 `json:"recordsFiltered"`
	Data            []struct {
		ID   int64  `json:"id"`
		NIP  string `json:"nip"`
		Nama string `json:"nama"`
	} `json:"data"`
}

func TestPegawaiDatatable(t *testing.T) {
	handler := newHandler(t)
	bidangID := seedBidang(t, handler)

	for i := 1; i <= 12; i++ {
		rec := send(handler, http.MethodPost, "/pegawai", map[string]any{
			"nip":       fmt.Sprintf("nip-%02d", i),
			"nama":      fmt.Sprintf("Pegawai %02d", i),
			"bidang_id": bidangID,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed %d: status = %d, body = %s", i, rec.Code, rec.Body.String())
		}
	}

	request := func(params url.Values) datatableResponse {
		t.Helper()
		rec := get(handler, "/pegawai/datatable?"+params.Encode())
		if rec.Code != http.StatusOK {
			t.Fatalf("datatable status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var out datatableResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode datatable: %v", err)
		}
		return out
	}

	first := request(url.Values{
		"draw":             {"1"},
		"start":            {"0"},
		"length":           {"5"},
		"order[0][column]": {"0"},
		"order[0][dir]":    {"asc"},
	})
	if first.Draw != 1 || first.RecordsTotal != 12 || first.RecordsFiltered != 12 {
		t.Fatalf("unexpected counters: %+v", first)
	}
	if len(first.Data) != 5 || first.Data[0].NIP != "nip-01" {
		t.Fatalf("unexpected first page: %+v", first.Data)
	}

	third := request(url.Values{
		"draw":             {"2"},
		"start":            {"10"},
		"length":           {"5"},
		"order[0][column]": {"0"},
		"order[0][dir]":    {"asc"},
	})
	if len(third.Data) != 2 || third.Data[0].ID != 11 {
		t.Fatalf("unexpected third page: %+v", third.Data)
	}

	desc := request(url.Values{
		"start":            {"0"},
		"length":           {"5"},
		"order[0][column]": {"2"},
		"order[0][dir]":    {"desc"},
	})
	if len(desc.Data) == 0 || desc.Data[0].Nama != "Pegawai 12" {
		t.Fatalf("unexpected desc order: %+v", desc.Data)
	}

	search := request(url.Values{
		"start":         {"0"},
		"length":        {"5"},
		"search[value]": {"Pegawai 0"},
	})
	if search.RecordsTotal != 12 || search.RecordsFiltered != 9 || len(search.Data) != 5 {
		t.Fatalf("unexpected search result: %+v", search)
	}

	lowerSearch := request(url.Values{
		"start":         {"0"},
		"length":        {"5"},
		"search[value]": {"pegawai 0"},
	})
	if lowerSearch.RecordsFiltered != 9 {
		t.Fatalf("case-insensitive search failed: %+v", lowerSearch)
	}

	none := request(url.Values{
		"start":         {"0"},
		"length":        {"5"},
		"search[value]": {"tidak-ada"},
	})
	if none.RecordsFiltered != 0 || len(none.Data) != 0 {
		t.Fatalf("unexpected empty search result: %+v", none)
	}

	// Unknown order column must fall back safely instead of reaching SQL.
	fallback := request(url.Values{
		"start":            {"0"},
		"length":           {"5"},
		"order[0][column]": {"99"},
		"order[0][dir]":    {"drop table"},
	})
	if len(fallback.Data) != 5 || fallback.Data[0].ID != 1 {
		t.Fatalf("unexpected fallback order: %+v", fallback.Data)
	}
}

type bidangListResponse struct {
	Data []struct {
		ID   int64  `json:"id"`
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"data"`
}

type bidangDatatableResponse struct {
	Draw            int   `json:"draw"`
	RecordsTotal    int64 `json:"recordsTotal"`
	RecordsFiltered int64 `json:"recordsFiltered"`
	Data            []struct {
		ID   int64  `json:"id"`
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"data"`
}

func TestBidangPageRenders(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, pages.Bidang.Route)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Data Bidang", "Tambah Bidang", "/bidang/create", `id="tabel-bidang"`, "serverSide: true", "/bidang/datatable"} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
}

func TestBidangFormPages(t *testing.T) {
	handler := newHandler(t)

	create := get(handler, pages.BidangCreate.Route)
	if create.Code != http.StatusOK {
		t.Fatalf("create form status = %d, want 200", create.Code)
	}
	createBody := create.Body.String()
	for _, want := range []string{`id="form-bidang"`, `data-action="/bidang"`, `data-method="POST"`, `name="kode"`, `name="nama"`} {
		if !strings.Contains(createBody, want) {
			t.Errorf("create form does not contain %q", want)
		}
	}

	created := send(handler, http.MethodPost, "/bidang", map[string]string{
		"kode": "BID-01",
		"nama": "Umum",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}

	edit := get(handler, "/bidang/1/edit")
	if edit.Code != http.StatusOK {
		t.Fatalf("edit form status = %d, want 200", edit.Code)
	}
	editBody := edit.Body.String()
	for _, want := range []string{`data-action="/bidang/1"`, `data-method="PUT"`} {
		if !strings.Contains(editBody, want) {
			t.Errorf("edit form does not contain %q", want)
		}
	}
	if strings.Contains(editBody, `value="BID-01"`) {
		t.Error("edit form must not render bidang data server-side")
	}

	show := get(handler, "/bidang/1")
	if show.Code != http.StatusOK {
		t.Fatalf("show status = %d, want 200", show.Code)
	}
	var shown bidangResponse
	if err := json.Unmarshal(show.Body.Bytes(), &shown); err != nil {
		t.Fatalf("decode show: %v", err)
	}
	if shown.Data.Kode != "BID-01" || shown.Data.Nama != "Umum" {
		t.Fatalf("unexpected show payload: %+v", shown.Data)
	}

	// The edit page is a static shell, so a missing row still renders; the
	// client discovers the 404 when it fetches Show.
	if rec := get(handler, "/bidang/999/edit"); rec.Code != http.StatusOK {
		t.Fatalf("missing edit form status = %d, want 200", rec.Code)
	}
	if rec := get(handler, "/bidang/999"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing show status = %d, want 404", rec.Code)
	}
	if rec := get(handler, "/bidang/abc"); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid show status = %d, want 400", rec.Code)
	}
}

func TestBidangCRUDFlow(t *testing.T) {
	handler := newHandler(t)

	create := send(handler, http.MethodPost, "/bidang", map[string]string{
		"kode": "BID-01",
		"nama": "Umum",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}

	var created bidangResponse
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data.ID == 0 || created.Data.Kode != "BID-01" {
		t.Fatalf("unexpected created row: %+v", created.Data)
	}

	duplicate := send(handler, http.MethodPost, "/bidang", map[string]string{
		"kode": "BID-01",
		"nama": "Bidang Lain",
	})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", duplicate.Code)
	}

	listRec := get(handler, "/bidang/datatable?start=0&length=10")
	if listRec.Code != http.StatusOK {
		t.Fatalf("datatable status = %d", listRec.Code)
	}
	var listed bidangDatatableResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode datatable: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Kode != "BID-01" {
		t.Fatalf("unexpected list: %+v", listed.Data)
	}

	update := send(handler, http.MethodPut, "/bidang/1", map[string]string{
		"kode": "BID-01",
		"nama": "Umum dan Kepegawaian",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", update.Code, update.Body.String())
	}

	var updated bidangResponse
	if err := json.Unmarshal(update.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Data.Nama != "Umum dan Kepegawaian" {
		t.Fatalf("update not persisted: %+v", updated.Data)
	}

	if rec := send(handler, http.MethodDelete, "/bidang/1", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if rec := get(handler, "/bidang/1"); rec.Code != http.StatusNotFound {
		t.Fatalf("show after delete status = %d, want 404", rec.Code)
	}
}

func TestBidangValidation(t *testing.T) {
	handler := newHandler(t)

	cases := []struct {
		name string
		body map[string]string
		want int
	}{
		{"missing kode", map[string]string{"nama": "Umum"}, http.StatusUnprocessableEntity},
		{"missing nama", map[string]string{"kode": "BID-01"}, http.StatusUnprocessableEntity},
		{"invalid json", nil, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tc.body == nil {
				req := httptest.NewRequest(http.MethodPost, "/bidang", strings.NewReader("{not json"))
				rec = httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			} else {
				rec = send(handler, http.MethodPost, "/bidang", tc.body)
			}
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	if rec := send(handler, http.MethodPut, "/bidang/abc", map[string]string{"kode": "X", "nama": "Y"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d, want 400", rec.Code)
	}
}

func TestBidangDatatable(t *testing.T) {
	handler := newHandler(t)

	for i := 1; i <= 12; i++ {
		rec := send(handler, http.MethodPost, "/bidang", map[string]string{
			"kode": fmt.Sprintf("BID-%02d", i),
			"nama": fmt.Sprintf("Bidang %02d", i),
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed %d: status = %d, body = %s", i, rec.Code, rec.Body.String())
		}
	}

	request := func(params url.Values) bidangDatatableResponse {
		t.Helper()
		rec := get(handler, "/bidang/datatable?"+params.Encode())
		if rec.Code != http.StatusOK {
			t.Fatalf("datatable status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var out bidangDatatableResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode datatable: %v", err)
		}
		return out
	}

	first := request(url.Values{
		"draw":             {"1"},
		"start":            {"0"},
		"length":           {"5"},
		"order[0][column]": {"0"},
		"order[0][dir]":    {"asc"},
	})
	if first.Draw != 1 || first.RecordsTotal != 12 || first.RecordsFiltered != 12 {
		t.Fatalf("unexpected counters: %+v", first)
	}
	if len(first.Data) != 5 || first.Data[0].Kode != "BID-01" {
		t.Fatalf("unexpected first page: %+v", first.Data)
	}

	desc := request(url.Values{
		"start":            {"0"},
		"length":           {"5"},
		"order[0][column]": {"2"},
		"order[0][dir]":    {"desc"},
	})
	if len(desc.Data) == 0 || desc.Data[0].Nama != "Bidang 12" {
		t.Fatalf("unexpected desc order: %+v", desc.Data)
	}

	search := request(url.Values{
		"start":         {"0"},
		"length":        {"5"},
		"search[value]": {"Bidang 0"},
	})
	if search.RecordsTotal != 12 || search.RecordsFiltered != 9 || len(search.Data) != 5 {
		t.Fatalf("unexpected search result: %+v", search)
	}

	fallback := request(url.Values{
		"start":            {"0"},
		"length":           {"5"},
		"order[0][column]": {"99"},
		"order[0][dir]":    {"drop table"},
	})
	if len(fallback.Data) != 5 || fallback.Data[0].ID != 1 {
		t.Fatalf("unexpected fallback order: %+v", fallback.Data)
	}
}

func TestBidangSearch(t *testing.T) {
	handler := newHandler(t)

	created := send(handler, http.MethodPost, "/bidang", map[string]string{
		"kode": "BID-01",
		"nama": "Kepegawaian",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}

	search := func(q string) bidangListResponse {
		t.Helper()
		rec := get(handler, "/bidang/search?q="+url.QueryEscape(q))
		if rec.Code != http.StatusOK {
			t.Fatalf("search status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var out bidangListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode search: %v", err)
		}
		return out
	}

	if got := search("kepeg"); len(got.Data) != 1 || got.Data[0].Kode != "BID-01" {
		t.Fatalf("unexpected name search: %+v", got.Data)
	}
	if got := search("KEPEG"); len(got.Data) != 1 {
		t.Fatalf("case-insensitive name search failed: %+v", got.Data)
	}
	if got := search("bid-01"); len(got.Data) != 1 {
		t.Fatalf("unexpected kode search: %+v", got.Data)
	}
	if got := search("BID-01"); len(got.Data) != 1 {
		t.Fatalf("case-insensitive kode search failed: %+v", got.Data)
	}
	if got := search("tidak-ada"); len(got.Data) != 0 {
		t.Fatalf("expected empty result, got %+v", got.Data)
	}
	if got := search(""); len(got.Data) != 0 {
		t.Fatalf("expected empty keyword to return nothing, got %+v", got.Data)
	}
}

func TestBidangDeleteInUse(t *testing.T) {
	handler := newHandler(t)
	bidangID := seedBidang(t, handler)

	created := send(handler, http.MethodPost, "/pegawai", map[string]any{
		"nip":       "198001012005011001",
		"nama":      "Budi Santoso",
		"bidang_id": bidangID,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create pegawai status = %d, body = %s", created.Code, created.Body.String())
	}

	if rec := send(handler, http.MethodDelete, fmt.Sprintf("/bidang/%d", bidangID), nil); rec.Code != http.StatusConflict {
		t.Fatalf("delete in-use status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}

	if rec := send(handler, http.MethodDelete, "/pegawai/1", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete pegawai status = %d, want 204", rec.Code)
	}
	if rec := send(handler, http.MethodDelete, fmt.Sprintf("/bidang/%d", bidangID), nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete bidang status = %d, want 204", rec.Code)
	}
}

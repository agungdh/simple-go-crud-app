package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	handler, err := router.New(pages.Pages, renderer, web.FS, pegawai)
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

func TestRootRendersStarter(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>AdminLTE 2 | Starter</title>") {
		t.Fatal("root did not render the starter page")
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
		ID      int64  `json:"id"`
		NIP     string `json:"nip"`
		Nama    string `json:"nama"`
		Jabatan string `json:"jabatan"`
		Email   string `json:"email"`
		Telepon string `json:"telepon"`
		Alamat  string `json:"alamat"`
	} `json:"data"`
}

type pegawaiListResponse struct {
	Data []struct {
		ID   int64  `json:"id"`
		NIP  string `json:"nip"`
		Nama string `json:"nama"`
	} `json:"data"`
}

func TestPegawaiPageRenders(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, pages.Pegawai.Route)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Data Pegawai", "Tambah Pegawai", "/pegawai", `id="tabel-pegawai"`, "/plugins/datatables/jquery.dataTables.min.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
}

func TestPegawaiFormPages(t *testing.T) {
	handler := newHandler(t)

	create := get(handler, pages.PegawaiCreate.Route)
	if create.Code != http.StatusOK {
		t.Fatalf("create form status = %d, want 200", create.Code)
	}
	createBody := create.Body.String()
	for _, want := range []string{`id="form-pegawai"`, `data-action="/api/pegawai"`, `data-method="POST"`} {
		if !strings.Contains(createBody, want) {
			t.Errorf("create form does not contain %q", want)
		}
	}

	created := send(handler, http.MethodPost, "/api/pegawai", map[string]string{
		"nip":  "198001012005011001",
		"nama": "Budi Santoso",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}

	edit := get(handler, "/pegawai/1/edit")
	if edit.Code != http.StatusOK {
		t.Fatalf("edit form status = %d, want 200", edit.Code)
	}
	editBody := edit.Body.String()
	for _, want := range []string{`data-action="/api/pegawai/1"`, `data-method="PUT"`, `value="Budi Santoso"`} {
		if !strings.Contains(editBody, want) {
			t.Errorf("edit form does not contain %q", want)
		}
	}

	if rec := get(handler, "/pegawai/999/edit"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing edit form status = %d, want 404", rec.Code)
	}
}

func TestPegawaiCRUDFlow(t *testing.T) {
	handler := newHandler(t)

	create := send(handler, http.MethodPost, "/api/pegawai", map[string]string{
		"nip":     "198001012005011001",
		"nama":    "Budi Santoso",
		"jabatan": "Staf",
		"email":   "budi@example.com",
		"telepon": "08123456789",
		"alamat":  "Jakarta",
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
	duplicate := send(handler, http.MethodPost, "/api/pegawai", map[string]string{
		"nip":  "198001012005011001",
		"nama": "Pegawai Lain",
	})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", duplicate.Code)
	}

	list := get(handler, "/api/pegawai")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}
	var listed pegawaiListResponse
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].NIP != "198001012005011001" {
		t.Fatalf("unexpected list: %+v", listed.Data)
	}

	update := send(handler, http.MethodPut, "/api/pegawai/1", map[string]string{
		"nip":     "198001012005011001",
		"nama":    "Budi Santoso, S.Kom",
		"jabatan": "Kepala Seksi",
		"email":   "budi@example.com",
		"telepon": "08123456789",
		"alamat":  "Bandung",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", update.Code, update.Body.String())
	}

	single := get(handler, "/api/pegawai/1")
	if single.Code != http.StatusOK {
		t.Fatalf("get status = %d", single.Code)
	}
	var fetched pegawaiResponse
	if err := json.Unmarshal(single.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if fetched.Data.Nama != "Budi Santoso, S.Kom" || fetched.Data.Jabatan != "Kepala Seksi" {
		t.Fatalf("update not persisted: %+v", fetched.Data)
	}

	if rec := send(handler, http.MethodDelete, "/api/pegawai/1", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if rec := get(handler, "/api/pegawai/1"); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", rec.Code)
	}
}

func TestPegawaiValidation(t *testing.T) {
	handler := newHandler(t)

	cases := []struct {
		name string
		body map[string]string
		want int
	}{
		{"missing nip", map[string]string{"nama": "Tanpa NIP"}, http.StatusUnprocessableEntity},
		{"missing nama", map[string]string{"nip": "123"}, http.StatusUnprocessableEntity},
		{"invalid email", map[string]string{"nip": "123", "nama": "Budi", "email": "bukan-email"}, http.StatusUnprocessableEntity},
		{"invalid json", nil, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tc.body == nil {
				req := httptest.NewRequest(http.MethodPost, "/api/pegawai", strings.NewReader("{not json"))
				rec = httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			} else {
				rec = send(handler, http.MethodPost, "/api/pegawai", tc.body)
			}
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	if rec := get(handler, "/api/pegawai/abc"); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d, want 400", rec.Code)
	}
}

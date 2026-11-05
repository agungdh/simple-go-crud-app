package router_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/web"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()

	renderer, err := render.New(web.FS)
	if err != nil {
		t.Fatalf("render.New: %v", err)
	}

	handler, err := router.New(pages.Pages, renderer, web.FS)
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

func TestRootRendersDashboard(t *testing.T) {
	handler := newHandler(t)

	rec := get(handler, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>AdminLTE 2 | Dashboard</title>") {
		t.Fatal("root did not render the dashboard page")
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

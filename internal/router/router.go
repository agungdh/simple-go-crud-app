package router

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
)

// New builds the HTTP handler for every AdminLTE page plus the embedded static
// assets found in the "static" directory of webFS.
//
// pegawai may be nil, in which case the CRUD page and API routes are not
// mounted.
func New(registry []pages.Page, renderer *render.Renderer, webFS fs.FS, pegawai *handler.Pegawai) (http.Handler, error) {
	staticFS, err := fs.Sub(webFS, "static")
	if err != nil {
		return nil, fmt.Errorf("static: %w", err)
	}

	mux := chi.NewRouter()

	for _, dir := range []string{"dist", "bower_components", "plugins"} {
		sub, err := fs.Sub(staticFS, dir)
		if err != nil {
			return nil, fmt.Errorf("static %s: %w", dir, err)
		}
		prefix := "/" + dir
		mux.Handle(prefix+"/*", http.StripPrefix(prefix+"/", http.FileServer(http.FS(sub))))
	}

	renderPage := func(w http.ResponseWriter, p pages.Page) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := renderer.Render(w, p); err != nil {
			log.Printf("render %s: %v", p.Route, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}

	byRoute := make(map[string]pages.Page, len(registry))
	for _, p := range registry {
		byRoute[p.Route] = p

		page := p
		mux.Get(page.Route, func(w http.ResponseWriter, r *http.Request) {
			renderPage(w, page)
		})
	}

	if starter, ok := byRoute["/starter.html"]; ok {
		mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
			renderPage(w, starter)
		})
	}

	if pegawai != nil {
		mux.Get(pages.Pegawai.Route, pegawai.Page)
		mux.Get(pages.PegawaiCreate.Route, pegawai.CreateForm)
		mux.Get(pages.PegawaiEdit.Route, pegawai.EditForm)
		mux.Get("/api/pegawai", pegawai.List)
		mux.Post("/api/pegawai", pegawai.Create)
		mux.Get("/api/pegawai/{id}", pegawai.Get)
		mux.Put("/api/pegawai/{id}", pegawai.Update)
		mux.Delete("/api/pegawai/{id}", pegawai.Delete)
	}

	if notFound, ok := byRoute["/pages/examples/404.html"]; ok {
		mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			if err := renderer.Render(w, notFound); err != nil {
				log.Printf("render 404: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		})
	}

	return mux, nil
}

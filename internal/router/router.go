package router

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
)

// New builds the HTTP handler for every AdminLTE page plus the static assets.
func New(registry []pages.Page, renderer *render.Renderer, staticDir string) http.Handler {
	mux := chi.NewRouter()

	mount := func(prefix, dir string) {
		mux.Handle(prefix+"/*", http.StripPrefix(prefix+"/", http.FileServer(http.Dir(filepath.Join(staticDir, dir)))))
	}
	mount("/dist", "dist")
	mount("/plugins", "plugins")
	mount("/bower_components", "bower_components")

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

	if index, ok := byRoute["/index.html"]; ok {
		mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
			renderPage(w, index)
		})
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

	return mux
}

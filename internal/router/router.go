package router

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/service"
)

// New wires repositories, services, and handlers into the HTTP router. Routes
// mirror api/openapi.yaml exactly, without trailing slashes. allowedOrigins is
// the exact list of browser origins permitted by CORS.
func New(db *bun.DB, allowedOrigins []string) http.Handler {
	bidangHandler := handler.NewBidang(service.NewBidang(repository.NewBidangRepository(db)))
	pegawaiHandler := handler.NewPegawai(service.NewPegawai(repository.NewPegawaiRepository(db)))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(recoverer)
	r.Use(corsMiddleware(allowedOrigins))

	r.Get("/bidang", bidangHandler.List)
	r.Post("/bidang", bidangHandler.Create)
	r.Get("/bidang/{id}", bidangHandler.Get)
	r.Put("/bidang/{id}", bidangHandler.Update)
	r.Delete("/bidang/{id}", bidangHandler.Delete)

	r.Get("/pegawai", pegawaiHandler.List)
	r.Post("/pegawai", pegawaiHandler.Create)
	r.Get("/pegawai/{id}", pegawaiHandler.Get)
	r.Put("/pegawai/{id}", pegawaiHandler.Update)
	r.Delete("/pegawai/{id}", pegawaiHandler.Delete)

	return r
}

// recoverer converts panics into the same JSON 500 envelope used by handlers,
// so clients never receive a plain-text error page.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			log.Printf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
			if wrapped.Status() != 0 {
				return
			}
			body, err := json.Marshal(map[string]string{
				"error": "Terjadi kesalahan di server.",
			})
			if err != nil {
				return
			}
			wrapped.Header().Set("Content-Type", "application/json")
			wrapped.WriteHeader(http.StatusInternalServerError)
			if _, err := wrapped.Write(body); err != nil {
				log.Printf("write panic response: %v", err)
			}
		}()

		next.ServeHTTP(wrapped, r)
	})
}

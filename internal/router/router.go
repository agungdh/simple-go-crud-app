package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/service"
)

// Config holds the settings required to build the HTTP router.
type Config struct {
	AllowedOrigins  []string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// New wires repositories, services, and handlers into the HTTP router. Routes
// mirror api/openapi.yaml exactly, without trailing slashes. allowedOrigins is
// the exact list of browser origins permitted by CORS.
func New(db *bun.DB, cfg Config) (http.Handler, error) {
	tokens, err := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("configure auth: %w", err)
	}

	bidangRepo := repository.NewBidangRepository(db)
	pegawaiRepo := repository.NewPegawaiRepository(db)
	userRepo := repository.NewUserRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)

	bidangHandler := handler.NewBidang(service.NewBidang(bidangRepo))
	pegawaiHandler := handler.NewPegawai(service.NewPegawai(pegawaiRepo, bidangRepo))
	authService := service.NewAuth(userRepo, refreshRepo, tokens)
	authHandler := handler.NewAuth(authService)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(recoverer)
	r.Use(corsMiddleware(cfg.AllowedOrigins))

	// Public endpoints: login and refresh only. Logout needs a valid access
	// token so a user can only revoke their own sessions.
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Group(func(r chi.Router) {
			r.Use(handler.RequireAuth(authService))
			r.Post("/logout", authHandler.Logout)
		})
	})

	// Authenticated reads are available to both roles.
	r.Group(func(r chi.Router) {
		r.Use(handler.RequireAuth(authService))

		r.Get("/bidang", bidangHandler.List)
		r.Get("/bidang/{id}", bidangHandler.Get)
		r.Get("/pegawai", pegawaiHandler.List)
		r.Get("/pegawai/{id}", pegawaiHandler.Get)

		// Writes require the admin role.
		r.Group(func(r chi.Router) {
			r.Use(handler.RequireAdmin)

			r.Post("/bidang", bidangHandler.Create)
			r.Put("/bidang/{id}", bidangHandler.Update)
			r.Delete("/bidang/{id}", bidangHandler.Delete)

			r.Post("/pegawai", pegawaiHandler.Create)
			r.Put("/pegawai/{id}", pegawaiHandler.Update)
			r.Delete("/pegawai/{id}", pegawaiHandler.Delete)
		})
	})

	return r, nil
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

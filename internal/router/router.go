package router

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/bidang"
	"simple-go-crud-app/internal/pegawai"
	"simple-go-crud-app/internal/shared/httpx"
)

// Config holds the settings required to build the HTTP router.
type Config struct {
	AllowedOrigins  []string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// New wires the domain packages into the HTTP router. Routes mirror
// api/openapi.yaml exactly, without trailing slashes. allowedOrigins is the
// exact list of browser origins permitted by CORS.
func New(db *bun.DB, cfg Config) (http.Handler, error) {
	tokens, err := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("configure auth: %w", err)
	}

	bidangRepo := bidang.NewRepository(db)
	pegawaiRepo := pegawai.NewRepository(db)
	userRepo := auth.NewUserRepository(db)
	refreshRepo := auth.NewRefreshTokenRepository(db)

	bidangHandler := bidang.NewHandler(bidang.NewService(bidangRepo))
	pegawaiHandler := pegawai.NewHandler(pegawai.NewService(pegawaiRepo, bidangRepo))
	authService := auth.NewService(userRepo, refreshRepo, tokens)
	authHandler := auth.NewHandler(authService)

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
			r.Use(auth.RequireAuth(authService))
			r.Post("/logout", authHandler.Logout)
		})
	})

	// Authenticated reads are available to both roles.
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(authService))

		r.Get("/bidang", bidangHandler.List)
		r.Get("/bidang/{id}", bidangHandler.Get)
		r.Get("/pegawai", pegawaiHandler.List)
		r.Get("/pegawai/{id}", pegawaiHandler.Get)

		// Writes require the admin role.
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin)

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
			httpx.WriteJSON(wrapped, http.StatusInternalServerError, httpx.ErrorResponse{
				Error: httpx.MessageInternal,
			})
		}()

		next.ServeHTTP(wrapped, r)
	})
}

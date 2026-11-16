package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/internal/shared/database"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func run() error {
	dbPath := envOr("DB_PATH", "persistence/db/app.db")
	addr := envOr("ADDR", ":8080")
	corsOrigins := splitList(envOr("CORS_ORIGINS", "http://localhost:8082"))
	jwtSecret := os.Getenv("JWT_SECRET")
	accessTTL, err := durationOr("ACCESS_TOKEN_TTL", "15m")
	if err != nil {
		return err
	}
	refreshTTL, err := durationOr("REFRESH_TOKEN_TTL", "168h")
	if err != nil {
		return err
	}

	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := database.Migrate(context.Background(), db, migrationsFS, "migrations"); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	httpHandler, err := router.New(db, router.Config{
		AllowedOrigins:  corsOrigins,
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	})
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}

	adminUsername := envOr("ADMIN_USERNAME", "admin")
	seeded, err := auth.EnsureAdmin(context.Background(), auth.NewUserRepository(db), auth.AdminSeed{
		Username: adminUsername,
		Password: os.Getenv("ADMIN_PASSWORD"),
		Name:     envOr("ADMIN_NAME", "Administrator"),
	})
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if seeded {
		log.Printf("seeded admin user %q", adminUsername)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s (database %s)", addr, dbPath)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Printf("shutting down http server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// durationOr parses a duration env value, using fallback when it is empty.
func durationOr(key, fallback string) (time.Duration, error) {
	parsed, err := time.ParseDuration(envOr(key, fallback))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

// splitList turns a comma-separated env value into a trimmed list, dropping
// empty entries.
func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

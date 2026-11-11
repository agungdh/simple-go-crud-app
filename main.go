package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simple-go-crud-app/internal/database"
	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/internal/service"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	addr := envOr("ADDR", ":8080")
	dbPath := envOr("DB_PATH", "persistence/db/app.db")

	ctx := context.Background()

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db, migrationsFS, "migrations"); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	pegawaiRepo := repository.NewPegawaiRepository(db)
	pegawaiSvc := service.NewPegawaiService(pegawaiRepo)
	pegawaiHandler := handler.NewPegawaiHandler(pegawaiSvc)

	srv := &http.Server{
		Addr:              addr,
		Handler:           router.New(pegawaiHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

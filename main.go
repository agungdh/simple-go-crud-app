package main

import (
	"context"
	"embed"
	"log"
	"os"

	"simple-go-crud-app/internal/database"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
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

	// The HTTP layer is not wired yet; the contract lives in api/openapi.yaml.
	log.Printf("database ready: %s", dbPath)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

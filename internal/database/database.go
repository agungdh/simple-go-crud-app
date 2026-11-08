// Package database opens the SQLite connection and runs goose migrations.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	// Register the pure-Go "sqlite" driver used by sqlitedialect.
	_ "github.com/uptrace/bun/driver/sqliteshim"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open connects to the SQLite file at path, creating it when needed.
//
// The connection pool is limited to a single connection because SQLite only
// allows one writer at a time; serializing access avoids "database is locked"
// errors under concurrent HTTP requests.
func Open(path string) (*bun.DB, error) {
	dsn := path
	if !strings.HasPrefix(path, "file:") {
		if path == ":memory:" {
			dsn = "file::memory:"
		} else {
			dsn = "file:" + path
		}
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn += sep + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

// Migrate applies every embedded goose migration that has not run yet.
func Migrate(ctx context.Context, db *bun.DB) error {
	goose.SetBaseFS(migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db.DB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

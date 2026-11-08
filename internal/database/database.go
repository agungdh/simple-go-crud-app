// Package database opens the SQLite connection and runs goose migrations.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	// Register the pure-Go "sqlite" driver used by sqlitedialect.
	_ "github.com/uptrace/bun/driver/sqliteshim"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Connection pool sizing for a small single-vCPU instance (512 MiB RAM).
//
// SQLite serializes writers, but WAL mode lets readers run while a write
// transaction is open, so a few connections are useful. The cap stays low
// because each connection keeps its own page cache and 1 vCPU cannot run many
// parallel queries anyway; busy_timeout absorbs the occasional writer race.
const (
	maxOpenConns    = 4
	maxIdleConns    = 4
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute
)

// Open connects to the SQLite file at path, creating it when needed.
//
// The pool is deliberately small: WAL mode allows concurrent readers, while
// writes remain serialized by SQLite itself and back off via busy_timeout.
func Open(path string) (*bun.DB, error) {
	dsn := path
	if !strings.HasPrefix(path, "file:") {
		dsn = "file:" + path
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
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxIdleConns)
	sqldb.SetConnMaxLifetime(connMaxLifetime)
	sqldb.SetConnMaxIdleTime(connMaxIdleTime)

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

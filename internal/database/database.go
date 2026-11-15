package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

const (
	// busyTimeoutMS is how long a connection waits for a lock held by another
	// connection before SQLite returns SQLITE_BUSY.
	busyTimeoutMS = 5000
	// maxOpenConns allows concurrent readers under WAL. Writes are still
	// serialized by SQLite, so the pool only needs to stay small.
	maxOpenConns = 10
)

// Open opens the SQLite database with the pragmas required by the project and
// wraps it in a Bun DB.
//
// The pragmas are passed through the DSN so that every pooled connection gets
// them, not just the first one. WAL plus synchronous=NORMAL keeps commits cheap
// while remaining crash-safe (a power loss can lose the last transactions, but
// the database is not corrupted).
func Open(path string) (*bun.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		path, busyTimeoutMS,
	)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite has a single writer, but WAL lets readers run alongside it, so the
	// pool does not need to be pinned to one connection. busy_timeout makes a
	// waiting writer retry instead of failing with "database is locked".
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxOpenConns)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return bun.NewDB(sqlDB, sqlitedialect.New()), nil
}

// Migrate runs all pending goose migrations from dir inside fsys.
func Migrate(ctx context.Context, db *bun.DB, fsys fs.FS, dir string) error {
	goose.SetBaseFS(fsys)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db.DB, dir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

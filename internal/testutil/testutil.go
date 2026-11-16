// Package testutil provides shared test helpers. It is only imported by tests.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/database"
)

// OpenDB opens a temporary SQLite database and runs every migration. The
// database is closed and deleted automatically when the test finishes.
func OpenDB(t *testing.T) *bun.DB {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	root := repoRoot(t)
	if err := database.Migrate(context.Background(), db, os.DirFS(root), "migrations"); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

// repoRoot locates the module root from this file's path.
func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve testutil path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

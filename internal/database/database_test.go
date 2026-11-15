package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"simple-go-crud-app/internal/database"
)

// TestOpenConfiguresEveryConnection verifies that the DSN pragmas are applied
// to each pooled connection, not only the first one.
func TestOpenConfiguresEveryConnection(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	if got := db.Stats().MaxOpenConnections; got < 2 {
		t.Fatalf("MaxOpenConnections = %d, want more than 1", got)
	}

	ctx := context.Background()

	// Hold several connections at once so the checks below cover connections
	// created after the first one by the pool.
	const conns = 3
	held := make([]*sql.Conn, 0, conns)
	for i := 0; i < conns; i++ {
		conn, err := db.DB.Conn(ctx)
		if err != nil {
			t.Fatalf("acquire connection %d: %v", i, err)
		}
		held = append(held, conn)
	}
	t.Cleanup(func() {
		for _, conn := range held {
			_ = conn.Close()
		}
	})

	for i, conn := range held {
		pragmas := map[string]string{
			"journal_mode": "wal",
			"synchronous":  "1", // NORMAL
			"foreign_keys": "1",
			"busy_timeout": "5000",
		}
		for name, want := range pragmas {
			var got string
			if err := conn.QueryRowContext(ctx, "PRAGMA "+name).Scan(&got); err != nil {
				t.Fatalf("connection %d: read pragma %s: %v", i, name, err)
			}
			if got != want {
				t.Errorf("connection %d: pragma %s = %q, want %q", i, name, got, want)
			}
		}
	}
}

// TestOpenSerializesConcurrentWrites checks that a multi-connection pool does
// not surface "database is locked" when several goroutines write at once; the
// busy_timeout pragma should make them wait instead of failing.
func TestOpenSerializesConcurrentWrites(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, "CREATE TABLE counter (id INTEGER PRIMARY KEY AUTOINCREMENT, value INTEGER NOT NULL)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	const (
		writers   = 4
		perWriter = 10
	)

	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if _, err := db.ExecContext(ctx, "INSERT INTO counter (value) VALUES (?)", i); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent write: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM counter").Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if want := writers * perWriter; count != want {
		t.Errorf("rows = %d, want %d", count, want)
	}
}

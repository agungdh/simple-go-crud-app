package upload

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simple-go-crud-app/internal/shared/apperr"
)

func newTestService(t *testing.T, maxBytes int64) (*Service, string) {
	t.Helper()

	dir := t.TempDir()
	svc, err := NewService(Config{TempDir: dir, MaxBytes: maxBytes})
	if err != nil {
		t.Fatalf("create upload service: %v", err)
	}
	return svc, dir
}

func requireDirEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read upload dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("upload dir has %d files, want 0", len(entries))
	}
}

func TestStoreWritesFile(t *testing.T) {
	svc, dir := newTestService(t, 1024)

	stored, err := svc.Store(context.Background(), strings.NewReader("hello"), 5)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}
	if stored.Size != 5 {
		t.Errorf("size = %d, want 5", stored.Size)
	}

	content, err := os.ReadFile(filepath.Join(dir, stored.Key))
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("content = %q, want hello", content)
	}
}

func TestStoreRejectsInvalidDeclaredLength(t *testing.T) {
	svc, dir := newTestService(t, 1024)

	if _, err := svc.Store(context.Background(), strings.NewReader("x"), -1); !errors.Is(err, apperr.ErrLengthRequired) {
		t.Errorf("missing length error = %v, want ErrLengthRequired", err)
	}
	if _, err := svc.Store(context.Background(), strings.NewReader(""), 0); !errors.Is(err, apperr.ErrIncompleteBody) {
		t.Errorf("empty body error = %v, want ErrIncompleteBody", err)
	}
	if _, err := svc.Store(context.Background(), strings.NewReader("x"), 1025); !errors.Is(err, apperr.ErrBodyTooLarge) {
		t.Errorf("oversized body error = %v, want ErrBodyTooLarge", err)
	}
	requireDirEmpty(t, dir)
}

func TestStoreRemovesTruncatedFile(t *testing.T) {
	svc, dir := newTestService(t, 1024)

	_, err := svc.Store(context.Background(), strings.NewReader("short"), 100)
	if !errors.Is(err, apperr.ErrIncompleteBody) {
		t.Fatalf("Store error = %v, want ErrIncompleteBody", err)
	}
	requireDirEmpty(t, dir)
}

type failingReader struct {
	sent bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, io.ErrUnexpectedEOF
	}
	r.sent = true
	return copy(p, "part"), nil
}

func TestStoreRemovesPartialFileOnReadError(t *testing.T) {
	svc, dir := newTestService(t, 1024)

	_, err := svc.Store(context.Background(), &failingReader{}, 100)
	if !errors.Is(err, apperr.ErrIncompleteBody) {
		t.Fatalf("Store error = %v, want ErrIncompleteBody", err)
	}
	requireDirEmpty(t, dir)
}

type tooLargeReader struct{}

func (tooLargeReader) Read(p []byte) (int, error) {
	return 0, &http.MaxBytesError{Limit: 4}
}

func TestStoreMapsMaxBytesError(t *testing.T) {
	svc, dir := newTestService(t, 1024)

	_, err := svc.Store(context.Background(), tooLargeReader{}, 4)
	if !errors.Is(err, apperr.ErrBodyTooLarge) {
		t.Fatalf("Store error = %v, want ErrBodyTooLarge", err)
	}
	requireDirEmpty(t, dir)
}

func TestNewServiceRejectsInvalidConfig(t *testing.T) {
	if _, err := NewService(Config{TempDir: "", MaxBytes: 1}); err == nil {
		t.Error("empty temp dir accepted")
	}
	if _, err := NewService(Config{TempDir: t.TempDir(), MaxBytes: 0}); err == nil {
		t.Error("zero max bytes accepted")
	}
}

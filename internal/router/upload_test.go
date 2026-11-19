package router_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/internal/shared/httpx"
	"simple-go-crud-app/internal/upload"
)

const (
	testUploadGlobalLimit   = int64(1 << 20)
	testUploadEndpointLimit = int64(4 << 10)
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func newUploadEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWithConfig(t, func(cfg *router.Config) {
		cfg.UploadMaxBytes = testUploadGlobalLimit
		cfg.UploadEndpointMaxBytes = testUploadEndpointLimit
	})
}

func (e *testEnv) serve(req *http.Request) *httptest.ResponseRecorder {
	e.t.Helper()
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) doUpload(token string, body []byte) *httptest.ResponseRecorder {
	e.t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.serve(req)
}

func requireUploadDirEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read upload dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("upload dir has %d leftover files, want 0", len(entries))
	}
}

func requireErrorBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Body.Len() == 0 {
		t.Fatal("error response has an empty body")
	}
	if decodeJSON[httpx.ErrorResponse](t, rec).Error == "" {
		t.Fatal("error response has an empty message")
	}
}

func TestUploadRequiresAuth(t *testing.T) {
	env := newUploadEnv(t)

	rec := env.doUpload("", []byte("hello"))
	requireStatus(t, rec, http.StatusUnauthorized)
	requireErrorBody(t, rec)
	requireUploadDirEmpty(t, env.uploadDir)
}

func TestUploadAllowsUserRole(t *testing.T) {
	env := newUploadEnv(t)
	payload := []byte("hello upload")

	rec := env.doUpload(env.userToken, payload)
	requireStatus(t, rec, http.StatusCreated)

	body := decodeJSON[httpx.Envelope[upload.Response]](t, rec)
	if !uuidV4Pattern.MatchString(body.Data.FileKey) {
		t.Errorf("file_key = %q, want UUID v4", body.Data.FileKey)
	}
	if body.Data.Size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", body.Data.Size, len(payload))
	}

	path := filepath.Join(env.uploadDir, body.Data.FileKey)
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if !bytes.Equal(stored, payload) {
		t.Errorf("stored file = %q, want %q", stored, payload)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat stored file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("stored file mode = %o, want 600", mode)
	}

	// The response must stay minimal; content type belongs to the later move.
	var raw map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	if _, exists := raw["data"]["content_type"]; exists {
		t.Error("response exposes content_type, want it omitted")
	}
}

func TestUploadRejectsMissingContentLength(t *testing.T) {
	env := newUploadEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("hello"))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}

	rec := env.serve(req)
	requireStatus(t, rec, http.StatusLengthRequired)
	requireErrorBody(t, rec)
	requireUploadDirEmpty(t, env.uploadDir)
}

func TestUploadRejectsEmptyBody(t *testing.T) {
	env := newUploadEnv(t)

	rec := env.doUpload(env.adminToken, nil)
	requireStatus(t, rec, http.StatusBadRequest)
	requireErrorBody(t, rec)
	requireUploadDirEmpty(t, env.uploadDir)
}

// trackingReader records whether the handler touched the body at all.
type trackingReader struct {
	data []byte
	read bool
}

func (r *trackingReader) Read(p []byte) (int, error) {
	r.read = true
	return 0, io.EOF
}

func TestUploadRejectsContentLengthOverLimitWithoutReading(t *testing.T) {
	env := newUploadEnv(t)
	reader := &trackingReader{data: []byte("ignored")}

	req := httptest.NewRequest(http.MethodPost, "/upload", reader)
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.ContentLength = testUploadEndpointLimit + 1

	rec := env.serve(req)
	requireStatus(t, rec, http.StatusRequestEntityTooLarge)
	requireErrorBody(t, rec)
	if reader.read {
		t.Error("handler read the body before rejecting the oversized request")
	}
	requireUploadDirEmpty(t, env.uploadDir)
}

func TestUploadUsesLowerGlobalLimit(t *testing.T) {
	env := newTestEnvWithConfig(t, func(cfg *router.Config) {
		cfg.UploadMaxBytes = 100
		cfg.UploadEndpointMaxBytes = 4 << 10
	})

	rec := env.doUpload(env.adminToken, bytes.Repeat([]byte("a"), 101))
	requireStatus(t, rec, http.StatusRequestEntityTooLarge)
	requireErrorBody(t, rec)
	requireUploadDirEmpty(t, env.uploadDir)
}

func TestUploadAcceptsExactLimit(t *testing.T) {
	env := newUploadEnv(t)
	payload := bytes.Repeat([]byte("a"), int(testUploadEndpointLimit))

	rec := env.doUpload(env.adminToken, payload)
	requireStatus(t, rec, http.StatusCreated)

	body := decodeJSON[httpx.Envelope[upload.Response]](t, rec)
	if body.Data.Size != testUploadEndpointLimit {
		t.Errorf("size = %d, want %d", body.Data.Size, testUploadEndpointLimit)
	}
}

func TestUploadRejectsBodyOverLimitDespiteHeader(t *testing.T) {
	env := newUploadEnv(t)
	payload := bytes.Repeat([]byte("a"), int(testUploadEndpointLimit)+100)

	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	// The header claims a small body; MaxBytesReader must still stop the stream.
	req.ContentLength = 10

	rec := env.serve(req)
	requireStatus(t, rec, http.StatusRequestEntityTooLarge)
	requireErrorBody(t, rec)
	requireUploadDirEmpty(t, env.uploadDir)
}

func TestUploadRejectsTruncatedBody(t *testing.T) {
	env := newUploadEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("short"))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.ContentLength = 100

	rec := env.serve(req)
	requireStatus(t, rec, http.StatusBadRequest)
	requireErrorBody(t, rec)
	requireUploadDirEmpty(t, env.uploadDir)
}

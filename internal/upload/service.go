package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"simple-go-crud-app/internal/shared/apperr"
)

// errWriteUpload marks failures while writing to disk so they can be mapped to
// a server error instead of an incomplete client body.
var errWriteUpload = errors.New("write upload file")

// Config configures the upload service. MaxBytes is already the effective
// limit, i.e. the lower of the global hard limit and EndpointMaxBytes.
type Config struct {
	TempDir  string
	MaxBytes int64
}

// Service stores uploaded bytes in a temporary directory.
type Service struct {
	tempDir  string
	maxBytes int64
}

// NewService validates the configuration and makes sure the temporary
// directory exists.
func NewService(cfg Config) (*Service, error) {
	if cfg.TempDir == "" {
		return nil, errors.New("upload temp dir is required")
	}
	if cfg.MaxBytes <= 0 {
		return nil, errors.New("upload max bytes must be positive")
	}
	if err := os.MkdirAll(cfg.TempDir, 0o700); err != nil {
		return nil, fmt.Errorf("create upload temp dir: %w", err)
	}
	return &Service{tempDir: cfg.TempDir, maxBytes: cfg.MaxBytes}, nil
}

// Limit returns the effective upload size limit in bytes.
func (s *Service) Limit() int64 {
	return s.maxBytes
}

// Store streams body into a new temporary file. The returned key is the
// generated file name; the caller must eventually move or delete the file.
func (s *Service) Store(ctx context.Context, body io.Reader, declared int64) (Stored, error) {
	switch {
	case declared < 0:
		return Stored{}, apperr.ErrLengthRequired
	case declared == 0:
		return Stored{}, apperr.ErrIncompleteBody
	case declared > s.maxBytes:
		return Stored{}, apperr.ErrBodyTooLarge
	}

	key := uuid.NewString()
	path := filepath.Join(s.tempDir, key)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Stored{}, fmt.Errorf("create upload file: %w", err)
	}

	written, err := s.copy(ctx, file, body)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		if errors.Is(err, errWriteUpload) || errors.Is(err, apperr.ErrBodyTooLarge) {
			return Stored{}, err
		}
		if written < declared {
			return Stored{}, apperr.ErrIncompleteBody
		}
		return Stored{}, err
	}
	if written != declared {
		_ = file.Close()
		_ = os.Remove(path)
		return Stored{}, apperr.ErrIncompleteBody
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return Stored{}, fmt.Errorf("close upload file: %w", err)
	}
	return Stored{Key: key, Size: written}, nil
}

// Stored describes a temporary file written by Store.
type Stored struct {
	Key  string
	Size int64
}

// copy streams src to dst with a small fixed buffer. Read errors from a
// truncated body are reported as incomplete; write errors stay server errors.
func (s *Service) copy(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buffer := make([]byte, copyBufferSize)
	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return written, fmt.Errorf("upload canceled: %w", err)
		}

		n, readErr := src.Read(buffer)
		if n > 0 {
			wn, writeErr := dst.Write(buffer[:n])
			written += int64(wn)
			if writeErr != nil {
				return written, fmt.Errorf("%w: %w", errWriteUpload, writeErr)
			}
			if wn != n {
				return written, fmt.Errorf("%w: %w", errWriteUpload, io.ErrShortWrite)
			}
		}

		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			var maxErr *http.MaxBytesError
			if errors.As(readErr, &maxErr) {
				return written, fmt.Errorf("upload exceeds limit: %w", apperr.ErrBodyTooLarge)
			}
			return written, fmt.Errorf("read upload body: %w", readErr)
		}
	}
}

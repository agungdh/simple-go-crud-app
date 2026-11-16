package apperr

import (
	"errors"
	"fmt"
)

// Sentinel errors shared between layers. The HTTP layer maps these to status
// codes; repositories and services should return them instead of writing HTTP
// responses.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// FieldErrors collects validation messages keyed by JSON field name. The key
// is the snake_case field name from the contract, never the Go field name.
//
// A field may have more than one message. Fields without problems are absent.
// An empty (or nil) value means there are no validation errors.
type FieldErrors map[string][]string

// Add appends a message for field.
func (e FieldErrors) Add(field, message string) {
	e[field] = append(e[field], message)
}

// Any reports whether at least one message was collected.
func (e FieldErrors) Any() bool {
	return len(e) > 0
}

// Error implements error so validation failures can bubble up through the
// service layer.
func (e FieldErrors) Error() string {
	return fmt.Sprintf("validation failed: %v", map[string][]string(e))
}

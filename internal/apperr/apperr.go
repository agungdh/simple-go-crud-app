package apperr

import "errors"

// Sentinel errors shared between layers. The HTTP layer maps these to status
// codes; repositories and services should return them instead of writing HTTP
// responses.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// ValidationError describes invalid input. Its message is safe to return to
// the client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Validation builds a ValidationError.
func Validation(message string) error {
	return &ValidationError{Message: message}
}

package dto

// ErrorResponse is the envelope for non-validation errors (400, 404, 500, and
// later 401/403). The message is safe to show to clients; internal error
// details must never be put here.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrorResponse is the 422 envelope defined by the contract. The
// field messages live under error.validations, keyed by JSON field name.
type ValidationErrorResponse struct {
	Error ValidationError `json:"error"`
}

// ValidationError groups per-field validation messages. Validations must never
// be nil so the envelope always marshals as an object.
type ValidationError struct {
	Validations map[string][]string `json:"validations"`
}

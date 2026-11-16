// Package httpx holds the HTTP plumbing shared by every domain: reading
// request bodies, writing the JSON envelopes, mapping errors to status codes,
// and parsing path parameters.
package httpx

// ErrorResponse is the envelope for non-validation errors (400, 401, 403, 404,
// and 500). The message is safe to show to clients; internal error details must
// never be put here.
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

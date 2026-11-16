package dto

// ErrorResponse is the envelope for non-validation errors. The message is safe
// to show to clients; internal error details must never be put here.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrorResponse is the 422 envelope defined by the contract. Errors
// maps a JSON field name to one or more messages; request-wide problems use the
// "body" key. The map must never be nil so the envelope always marshals as an
// object.
type ValidationErrorResponse struct {
	Errors map[string][]string `json:"errors"`
}

package dto

// ErrorResponse is the envelope for non-validation errors. The message is safe
// to show to clients; internal error details must never be put here.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrorResponse is the 422 envelope defined by the contract. Errors
// maps a JSON field name to one or more messages. The map must never be nil so
// the envelope always marshals as an object.
type ValidationErrorResponse struct {
	Errors map[string][]string `json:"errors"`
}

// BadRequestResponse is the 400 envelope for a body that cannot be read. Unlike
// the 422 envelope, it only carries request-wide errors under "body".
type BadRequestResponse struct {
	Errors BadRequestErrors `json:"errors"`
}

// BadRequestErrors groups request-wide errors for an unreadable body.
type BadRequestErrors struct {
	Body []string `json:"body"`
}

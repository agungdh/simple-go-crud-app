package dto

// ErrorResponse is the error envelope defined by the contract. The message is
// safe to show to clients; internal error details must never be put here.
type ErrorResponse struct {
	Error string `json:"error"`
}

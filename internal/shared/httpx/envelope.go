package httpx

// Envelope is the single-object response envelope used by GET by id.
type Envelope[T any] struct {
	Data T `json:"data"`
}

// ListEnvelope is the collection response envelope. Data must marshal as []
// when empty, never null, so callers must pass a non-nil slice.
type ListEnvelope[T any] struct {
	Data []T `json:"data"`
}

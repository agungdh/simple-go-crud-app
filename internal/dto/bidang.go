// Package dto holds the request and response payloads defined by
// api/openapi.yaml.
//
// The project has no code generation: every struct here is written by hand and
// must stay in sync with the contract. dto_test.go validates sample payloads
// against the OpenAPI schemas, so a field rename without a contract change
// fails the test suite.
package dto

// Bidang is the bidang representation returned to clients.
type Bidang struct {
	ID     int64  `json:"id"`
	Bidang string `json:"bidang"`
}

// BidangInput is the request body for creating or updating a bidang.
type BidangInput struct {
	Bidang string `json:"bidang"`
}

// BidangResponse is the single-object envelope.
type BidangResponse struct {
	Data Bidang `json:"data"`
}

// BidangListResponse is the collection envelope. Data must marshal as [] when
// empty, never null.
type BidangListResponse struct {
	Data []Bidang `json:"data"`
}

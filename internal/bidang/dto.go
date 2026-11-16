// The project has no code generation: every struct here is written by hand
// after the contract is stable and must stay in sync with api/openapi.yaml.
// The contract is validated with `make api-validate`; DTO compatibility is
// checked during review, not by Go tests.
package bidang

// Response is the bidang representation returned to clients.
type Response struct {
	ID     int64  `json:"id"`
	Bidang string `json:"bidang"`
}

// Input is the request body for creating or updating a bidang.
type Input struct {
	Bidang string `json:"bidang"`
}

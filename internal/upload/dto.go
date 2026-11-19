// The project has no code generation: every struct here is written by hand
// after the contract is stable and must stay in sync with api/openapi.yaml.
package upload

// Response is the metadata returned after a temporary upload succeeds. The
// server generates file_key; clients never choose the stored name.
type Response struct {
	FileKey string `json:"file_key"`
	Size    int64  `json:"size"`
}

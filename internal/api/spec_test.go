package api_test

import (
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// loadSpec loads and validates the API contract. Every drift in the spec fails
// here before it can reach the generated code.
func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("validate spec: %v", err)
	}
	return doc
}

func TestOpenAPIContract(t *testing.T) {
	doc := loadSpec(t)

	want := map[string]string{
		"GET /bidang":          "listBidang",
		"POST /bidang":         "createBidang",
		"GET /bidang/{id}":     "getBidang",
		"PUT /bidang/{id}":     "updateBidang",
		"DELETE /bidang/{id}":  "deleteBidang",
		"GET /pegawai":         "listPegawai",
		"POST /pegawai":        "createPegawai",
		"GET /pegawai/{id}":    "getPegawai",
		"PUT /pegawai/{id}":    "updatePegawai",
		"DELETE /pegawai/{id}": "deletePegawai",
	}

	got := make(map[string]string, len(want))
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			key := method + " " + path
			if op.OperationID == "" {
				t.Errorf("%s: operationId wajib diisi", key)
			}
			if len(op.Tags) == 0 {
				t.Errorf("%s: tag wajib diisi", key)
			}
			got[key] = op.OperationID
		}
	}

	for key, operationID := range want {
		if got[key] != operationID {
			t.Errorf("%s: operationId = %q, want %q", key, got[key], operationID)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			t.Errorf("%s: endpoint tidak ada di kontrak yang disepakati", key)
		}
	}
}

func TestOpenAPISchemasMatchConventions(t *testing.T) {
	doc := loadSpec(t)

	internalColumns := []string{
		"search_text", "is_male", "bidang_fts", "pegawai_fts",
		"created_at", "created_by",
		"updated_at", "updated_by",
		"deleted_at", "deleted_by",
	}

	for name, ref := range doc.Components.Schemas {
		if ref.Value == nil {
			t.Fatalf("schema %q kosong", name)
		}
		for _, column := range internalColumns {
			if _, exposed := ref.Value.Properties[column]; exposed {
				t.Errorf("schema %q membocorkan kolom internal %q", name, column)
			}
		}
	}
}

func TestOpenAPIDateAndGenderContract(t *testing.T) {
	doc := loadSpec(t)

	request := doc.Components.Schemas["TanggalLahirInput"]
	if request == nil || request.Value == nil || request.Value.Pattern == "" {
		t.Fatal("TanggalLahirInput harus punya pattern DD-MM-YYYY atau YYYY-MM-DD")
	}
	if request.Value.Pattern != `^(\d{2}-\d{2}-\d{4}|\d{4}-\d{2}-\d{2})$` {
		t.Errorf("TanggalLahirInput pattern = %q", request.Value.Pattern)
	}

	response := doc.Components.Schemas["TanggalLahir"]
	if response == nil || response.Value == nil || response.Value.Pattern != `^\d{2}-\d{2}-\d{4}$` {
		t.Error("TanggalLahir response harus selalu DD-MM-YYYY")
	}

	gender := doc.Components.Schemas["JenisKelamin"]
	if gender == nil || gender.Value == nil || len(gender.Value.Enum) != 2 {
		t.Fatal("JenisKelamin harus enum dengan dua nilai")
	}
	if gender.Value.Enum[0] != "Pria" || gender.Value.Enum[1] != "Wanita" {
		t.Errorf("JenisKelamin enum = %v, want [Pria Wanita]", gender.Value.Enum)
	}
}

package api

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestOpenAPIContract keeps the spec honest: it must be valid, the endpoint
// list is fixed, and internal columns must never leak into a schema.
func TestOpenAPIContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("validate spec: %v", err)
	}

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

func TestOpenAPIStatusCodes(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}

	allowed := map[int]bool{
		200: true, 201: true, 204: true,
		401: true, 403: true,
		404: true, 422: true, 500: true,
	}

	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			key := method + " " + path
			codes := make(map[int]bool)
			for code := range op.Responses.Map() {
				n, err := strconv.Atoi(code)
				if err != nil {
					t.Errorf("%s: status %q bukan angka", key, code)
					continue
				}
				if !allowed[n] {
					t.Errorf("%s: status %d tidak diizinkan oleh kontrak", key, n)
				}
				codes[n] = true
			}

			// 404 hanya untuk operasi by id.
			byID := strings.Contains(path, "{id}")
			if byID && !codes[404] {
				t.Errorf("%s: operasi by id wajib mendokumentasikan 404", key)
			}
			if !byID && codes[404] {
				t.Errorf("%s: 404 hanya boleh dipakai operasi by id", key)
			}
		}
	}
}

// TestOpenAPISpecHasNoInternalDetails keeps the contract consumer-facing:
// table names, indexes, storage format, and search internals are not the
// client's business.
func TestOpenAPISpecHasNoInternalDetails(t *testing.T) {
	raw, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	lower := strings.ToLower(string(raw))

	forbidden := []string{
		"soft delete", "search_text", "is_male", "foreign key",
		"unique index", "autoincrement", "epoch", "fts5", "trigram",
		"sqlite", "migration",
		"created_at", "created_by", "updated_at", "updated_by",
		"deleted_at", "deleted_by",
	}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Errorf("spec membocorkan detail internal: %q", word)
		}
	}
}

func TestOpenAPISchemasMatchConventions(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}

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

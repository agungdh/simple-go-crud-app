package dto_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"simple-go-crud-app/internal/dto"
)

// loadSpec loads the contract that the DTOs are written by hand against.
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

// assertMatchesSchema validates the JSON produced by a DTO against a schema in
// api/openapi.yaml. Renaming a JSON field without touching the contract fails
// here instead of silently breaking clients.
func assertMatchesSchema(t *testing.T, doc *openapi3.T, schemaName string, payload any) {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	ref, ok := doc.Components.Schemas[schemaName]
	if !ok || ref.Value == nil {
		t.Fatalf("schema %q tidak ada di kontrak", schemaName)
	}
	if err := ref.Value.VisitJSON(value); err != nil {
		t.Fatalf("%s tidak sesuai kontrak: %v\npayload: %s", schemaName, err, raw)
	}
}

func TestBidangDTOMatchesContract(t *testing.T) {
	doc := loadSpec(t)

	assertMatchesSchema(t, doc, "Bidang", dto.Bidang{ID: 1, Bidang: "Keuangan"})
	assertMatchesSchema(t, doc, "BidangInput", dto.BidangInput{Bidang: "Keuangan"})
	assertMatchesSchema(t, doc, "BidangResponse", dto.BidangResponse{
		Data: dto.Bidang{ID: 1, Bidang: "Keuangan"},
	})
	assertMatchesSchema(t, doc, "BidangListResponse", dto.BidangListResponse{
		Data: []dto.Bidang{{ID: 1, Bidang: "Keuangan"}},
	})
	assertMatchesSchema(t, doc, "BidangListResponse", dto.BidangListResponse{Data: []dto.Bidang{}})
}

func TestPegawaiDTOMatchesContract(t *testing.T) {
	doc := loadSpec(t)

	pegawai := dto.Pegawai{
		ID:           1,
		BidangID:     2,
		Nama:         "Budi Santoso",
		NIP:          "199001012020011001",
		TanggalLahir: "17-08-1990",
		Alamat:       "Jl. Merdeka No. 1, Jakarta",
		JenisKelamin: dto.Pria,
	}

	assertMatchesSchema(t, doc, "Pegawai", pegawai)
	assertMatchesSchema(t, doc, "PegawaiResponse", dto.PegawaiResponse{Data: pegawai})
	assertMatchesSchema(t, doc, "PegawaiListResponse", dto.PegawaiListResponse{
		Data: []dto.Pegawai{pegawai},
	})
	assertMatchesSchema(t, doc, "PegawaiInput", dto.PegawaiInput{
		BidangID:     2,
		Nama:         "Budi Santoso",
		NIP:          "199001012020011001",
		TanggalLahir: "1990-08-17",
		Alamat:       "Jl. Merdeka No. 1, Jakarta",
		JenisKelamin: dto.InputPria,
	})
}

func TestErrorResponseMatchesContract(t *testing.T) {
	doc := loadSpec(t)

	assertMatchesSchema(t, doc, "ErrorResponse", dto.ErrorResponse{Error: "bidang tidak ditemukan"})
}

// TestListEnvelopesNeverMarshalNull guards the contract promise that list
// responses always return an array.
func TestListEnvelopesNeverMarshalNull(t *testing.T) {
	tests := map[string]any{
		"BidangListResponse":  dto.BidangListResponse{Data: []dto.Bidang{}},
		"PegawaiListResponse": dto.PegawaiListResponse{Data: []dto.Pegawai{}},
	}

	for name, payload := range tests {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if string(raw) != `{"data":[]}` {
			t.Errorf("%s = %s, want {\"data\":[]}", name, raw)
		}
	}
}

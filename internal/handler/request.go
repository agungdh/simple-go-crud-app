package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/dto"
)

// maxBodyBytes matches the contract: bodies larger than 1 MiB are a 400.
const maxBodyBytes = 1 << 20

// readObject reads a request body and decodes exactly one JSON object. Every
// body problem (empty, malformed, not an object, more than one value, too
// large) is a 400 with the same safe message.
func readObject(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, messageBadBody)
		return nil, false
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		writeError(w, http.StatusBadRequest, messageBadBody)
		return nil, false
	}
	return fields, true
}

// rejectUnknown adds a 422 message for every key that is not in allowed.
func rejectUnknown(fields map[string]json.RawMessage, allowed map[string]struct{}, errs apperr.FieldErrors) {
	for name := range fields {
		if _, ok := allowed[name]; !ok {
			errs.Add(name, "Field tidak dikenal.")
		}
	}
}

// decodeString decodes one field as a string. A missing or null field stays
// empty so semantic validation reports it as required.
func decodeString(fields map[string]json.RawMessage, name, label string, errs apperr.FieldErrors) string {
	raw, ok := fields[name]
	if !ok {
		return ""
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		errs.Add(name, label+" harus berupa teks.")
		return ""
	}
	return value
}

// decodeInt64 decodes one field as an integer. Fractional numbers and strings
// are rejected; missing or null fields stay zero.
func decodeInt64(fields map[string]json.RawMessage, name, label string, errs apperr.FieldErrors) int64 {
	raw, ok := fields[name]
	if !ok {
		return 0
	}

	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		errs.Add(name, label+" harus berupa bilangan bulat.")
		return 0
	}
	return value
}

// decodeBidangInput maps the wire object to the DTO and collects structural
// problems (unknown fields and wrong JSON types).
func decodeBidangInput(fields map[string]json.RawMessage) (dto.BidangInput, apperr.FieldErrors) {
	errs := apperr.FieldErrors{}
	rejectUnknown(fields, map[string]struct{}{"bidang": {}}, errs)

	return dto.BidangInput{
		Bidang: decodeString(fields, "bidang", "Bidang", errs),
	}, errs
}

// decodePegawaiInput maps the wire object to the DTO and collects structural
// problems (unknown fields and wrong JSON types).
func decodePegawaiInput(fields map[string]json.RawMessage) (dto.PegawaiInput, apperr.FieldErrors) {
	errs := apperr.FieldErrors{}
	rejectUnknown(fields, map[string]struct{}{
		"bidang_id":     {},
		"nama":          {},
		"nip":           {},
		"tanggal_lahir": {},
		"alamat":        {},
		"jenis_kelamin": {},
	}, errs)

	return dto.PegawaiInput{
		BidangID:     decodeInt64(fields, "bidang_id", "Bidang ID", errs),
		Nama:         decodeString(fields, "nama", "Nama", errs),
		NIP:          decodeString(fields, "nip", "NIP", errs),
		TanggalLahir: decodeString(fields, "tanggal_lahir", "Tanggal lahir", errs),
		Alamat:       decodeString(fields, "alamat", "Alamat", errs),
		JenisKelamin: dto.JenisKelaminInput(decodeString(fields, "jenis_kelamin", "Jenis kelamin", errs)),
	}, errs
}

// combineValidation merges structural and semantic errors. Structural messages
// (wrong type, unknown field) win for the same field so a field never reports
// both "must be text" and "required".
func combineValidation(structural apperr.FieldErrors, semantic error) error {
	merged := make(apperr.FieldErrors, len(structural))
	for field, messages := range structural {
		merged[field] = append([]string(nil), messages...)
	}

	if semantic != nil {
		var fields apperr.FieldErrors
		if errors.As(semantic, &fields) {
			for field, messages := range fields {
				if _, exists := merged[field]; exists {
					continue
				}
				merged[field] = append([]string(nil), messages...)
			}
		}
	}

	if merged.Any() {
		return merged
	}
	return nil
}

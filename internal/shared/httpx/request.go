package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"simple-go-crud-app/internal/shared/apperr"
)

// maxBodyBytes matches the contract: bodies larger than 1 MiB are a 400.
const maxBodyBytes = 1 << 20

// ReadObject reads a request body and decodes exactly one JSON object. Every
// body problem (empty, malformed, not an object, more than one value, too
// large) is a 400 with the same safe message.
func ReadObject(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, MessageBadBody)
		return nil, false
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		WriteError(w, http.StatusBadRequest, MessageBadBody)
		return nil, false
	}
	return fields, true
}

// RejectUnknown adds a 422 message for every key that is not in allowed.
func RejectUnknown(fields map[string]json.RawMessage, allowed map[string]struct{}, errs apperr.FieldErrors) {
	for name := range fields {
		if _, ok := allowed[name]; !ok {
			errs.Add(name, "Field tidak dikenal.")
		}
	}
}

// DecodeString decodes one field as a string. A missing or null field stays
// empty so semantic validation reports it as required.
func DecodeString(fields map[string]json.RawMessage, name, label string, errs apperr.FieldErrors) string {
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

// DecodeInt64 decodes one field as an integer. Fractional numbers and strings
// are rejected; missing or null fields stay zero.
func DecodeInt64(fields map[string]json.RawMessage, name, label string, errs apperr.FieldErrors) int64 {
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

// CombineValidation merges structural and semantic errors. Structural messages
// (wrong type, unknown field) win for the same field so a field never reports
// both "must be text" and "required".
func CombineValidation(structural apperr.FieldErrors, semantic error) error {
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

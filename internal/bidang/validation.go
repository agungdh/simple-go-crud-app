package bidang

import (
	"strings"
	"unicode/utf8"

	"simple-go-crud-app/internal/shared/apperr"
)

const maxBidangLength = 150

// validateInput trims the name and returns the value to persist. All
// validation messages are collected before returning.
func validateInput(input Input) (string, error) {
	value := strings.TrimSpace(input.Bidang)

	errs := apperr.FieldErrors{}
	switch {
	case value == "":
		errs.Add("bidang", "Bidang wajib diisi.")
	case utf8.RuneCountInString(value) > maxBidangLength:
		errs.Add("bidang", "Bidang maksimal 150 karakter.")
	}
	if errs.Any() {
		return value, errs
	}
	return value, nil
}

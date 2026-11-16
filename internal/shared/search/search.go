// Package search holds the helpers shared by every domain that supports
// trigram search: keyword normalization and FTS5 phrase quoting.
package search

import (
	"strings"
	"unicode/utf8"

	"simple-go-crud-app/internal/shared/apperr"
)

const (
	minLength = 3
	maxLength = 100
)

// Normalize trims the search keyword and enforces the 3-100 character range
// required by the trigram index. An empty keyword means "no filter".
func Normalize(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", nil
	}

	errs := apperr.FieldErrors{}
	length := utf8.RuneCountInString(q)
	switch {
	case length < minLength:
		errs.Add("q", "Kata kunci pencarian minimal 3 karakter.")
	case length > maxLength:
		errs.Add("q", "Kata kunci pencarian maksimal 100 karakter.")
	}
	if errs.Any() {
		return q, errs
	}
	return q, nil
}

// Phrase wraps user input in a quoted FTS5 phrase so that operators such as
// AND, OR, NOT, and * are matched as literal text.
func Phrase(query string) string {
	return `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
}

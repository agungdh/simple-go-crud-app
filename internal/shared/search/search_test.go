package search

import (
	"errors"
	"strings"
	"testing"

	"simple-go-crud-app/internal/shared/apperr"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "empty", value: "", want: ""},
		{name: "whitespace only", value: "   ", want: ""},
		{name: "trims", value: "  keuangan  ", want: "keuangan"},
		{name: "two characters", value: "ke", wantErr: true},
		{name: "three characters", value: "keu", want: "keu"},
		{name: "hundred characters", value: strings.Repeat("a", 100), want: strings.Repeat("a", 100)},
		{name: "hundred and one characters", value: strings.Repeat("a", 101), wantErr: true},
		{name: "rune count not byte count", value: strings.Repeat("é", 100), want: strings.Repeat("é", 100)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Normalize(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) error = nil, want validation error", test.value)
				}
				var fields apperr.FieldErrors
				if !errors.As(err, &fields) {
					t.Fatalf("Normalize(%q) error type = %T, want apperr.FieldErrors", test.value, err)
				}
				if len(fields["q"]) == 0 {
					t.Errorf("Normalize(%q) has no message for q", test.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) unexpected error: %v", test.value, err)
			}
			if got != test.want {
				t.Errorf("Normalize(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestPhrase(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "budi", want: `"budi"`},
		{value: `a"b`, want: `"a""b"`},
		{value: "AND OR NOT *", want: `"AND OR NOT *"`},
	}

	for _, test := range tests {
		if got := Phrase(test.value); got != test.want {
			t.Errorf("Phrase(%q) = %q, want %q", test.value, got, test.want)
		}
	}
}

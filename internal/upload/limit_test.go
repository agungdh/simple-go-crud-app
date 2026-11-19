package upload

import (
	"math"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{input: "10MB", want: 10 << 20},
		{input: "2MiB", want: 2 << 20},
		{input: "2097152", want: 2 << 20},
		{input: " 1 kb ", want: 1 << 10},
		{input: "10 MB", want: 10 << 20},
		{input: "1GB", want: 1 << 30},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseSize(test.input)
			if err != nil {
				t.Fatalf("ParseSize(%q) error: %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("ParseSize(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestParseSizeRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"",
		"0",
		"-1MB",
		"10XB",
		"1.5MB",
		"MB",
		"9223372036854775808",
		"9999999999999999999GB",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseSize(input); err == nil {
				t.Errorf("ParseSize(%q) = nil error, want failure", input)
			}
		})
	}
}

func TestParseSizeAcceptsMaxInt64(t *testing.T) {
	got, err := ParseSize("9223372036854775807")
	if err != nil {
		t.Fatalf("ParseSize(max int64) error: %v", err)
	}
	if got != math.MaxInt64 {
		t.Errorf("ParseSize(max int64) = %d, want %d", got, int64(math.MaxInt64))
	}
}

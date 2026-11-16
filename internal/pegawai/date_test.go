package pegawai

import (
	"testing"
	"time"
)

func TestParseTanggalLahir(t *testing.T) {
	now := time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{name: "DD-MM-YYYY", value: "17-08-1990", want: "1990-08-17"},
		{name: "YYYY-MM-DD", value: "1990-08-17", want: "1990-08-17"},
		{name: "leap day", value: "29-02-2024", want: "2024-02-29"},
		{name: "today", value: "16-11-2026", want: "2026-11-16"},
		{name: "tomorrow", value: "17-11-2026", wantErr: errFutureDate},
		{name: "impossible date", value: "31-02-2020", wantErr: errInvalidDate},
		{name: "non-leap day", value: "29-02-2023", wantErr: errInvalidDate},
		{name: "wrong separator", value: "2020/02/02", wantErr: errInvalidDate},
		{name: "text", value: "kemarin", wantErr: errInvalidDate},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseTanggalLahir(test.value, now)
			if test.wantErr != nil {
				if err != test.wantErr {
					t.Fatalf("parseTanggalLahir(%q) error = %v, want %v", test.value, err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTanggalLahir(%q) unexpected error: %v", test.value, err)
			}
			if got != test.want {
				t.Errorf("parseTanggalLahir(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

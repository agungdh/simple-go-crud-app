package pegawai

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"simple-go-crud-app/internal/shared/apperr"
)

const (
	maxNamaLength   = 150
	maxNIPLength    = 50
	maxAlamatLength = 500
)

// pegawaiFields holds the normalized values ready to be persisted.
type pegawaiFields struct {
	BidangID     int64
	Nama         string
	NIP          string
	TanggalLahir string
	Alamat       string
	IsMale       bool
}

// validateInput validates every field and returns normalized values. Business
// dates are converted to ISO-8601 before storage.
func validateInput(input Input) (pegawaiFields, error) {
	result := pegawaiFields{}
	errs := apperr.FieldErrors{}

	switch {
	case input.BidangID == 0:
		errs.Add("bidang_id", "Bidang wajib diisi.")
	case input.BidangID < 0:
		errs.Add("bidang_id", "Bidang tidak valid.")
	default:
		result.BidangID = input.BidangID
	}

	result.Nama = strings.TrimSpace(input.Nama)
	switch {
	case result.Nama == "":
		errs.Add("nama", "Nama wajib diisi.")
	case utf8.RuneCountInString(result.Nama) > maxNamaLength:
		errs.Add("nama", "Nama maksimal 150 karakter.")
	}

	result.NIP = strings.TrimSpace(input.NIP)
	switch {
	case result.NIP == "":
		errs.Add("nip", "NIP wajib diisi.")
	case utf8.RuneCountInString(result.NIP) > maxNIPLength:
		errs.Add("nip", "NIP maksimal 50 karakter.")
	}

	tanggalLahir := strings.TrimSpace(input.TanggalLahir)
	switch {
	case tanggalLahir == "":
		errs.Add("tanggal_lahir", "Tanggal lahir wajib diisi.")
	default:
		iso, err := parseTanggalLahir(tanggalLahir, time.Now().UTC())
		switch {
		case errors.Is(err, errFutureDate):
			errs.Add("tanggal_lahir", "Tanggal lahir tidak boleh di masa depan.")
		case err != nil:
			errs.Add("tanggal_lahir", "Tanggal lahir tidak valid.")
		default:
			result.TanggalLahir = iso
		}
	}

	result.Alamat = strings.TrimSpace(input.Alamat)
	switch {
	case result.Alamat == "":
		errs.Add("alamat", "Alamat wajib diisi.")
	case utf8.RuneCountInString(result.Alamat) > maxAlamatLength:
		errs.Add("alamat", "Alamat maksimal 500 karakter.")
	}

	jenisKelamin := strings.ToLower(strings.TrimSpace(string(input.JenisKelamin)))
	switch jenisKelamin {
	case string(InputPria):
		result.IsMale = true
	case string(InputWanita):
		result.IsMale = false
	case "":
		errs.Add("jenis_kelamin", "Jenis kelamin wajib diisi.")
	default:
		errs.Add("jenis_kelamin", "Jenis kelamin harus pria atau wanita.")
	}

	if errs.Any() {
		return result, errs
	}
	return result, nil
}

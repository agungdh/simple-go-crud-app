package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/dto"
)

const (
	maxBidangLength   = 150
	maxNamaLength     = 150
	maxNIPLength      = 50
	maxAlamatLength   = 500
	maxUsernameLength = 50

	maxRefreshTokenLength = 512

	minSearchLength = 3
	maxSearchLength = 100
)

// isoDateLayout is how business dates are stored in SQLite.
const isoDateLayout = "2006-01-02"

var (
	errInvalidDate = errors.New("invalid date")
	errFutureDate  = errors.New("future date")
)

// normalizeQuery trims the search keyword and enforces the 3-100 character
// range required by the trigram index. An empty keyword means "no filter".
func normalizeQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", nil
	}

	errs := apperr.FieldErrors{}
	length := utf8.RuneCountInString(q)
	switch {
	case length < minSearchLength:
		errs.Add("q", "Kata kunci pencarian minimal 3 karakter.")
	case length > maxSearchLength:
		errs.Add("q", "Kata kunci pencarian maksimal 100 karakter.")
	}
	if errs.Any() {
		return q, errs
	}
	return q, nil
}

// validateBidangInput trims the name and returns the value to persist. All
// validation messages are collected before returning.
func validateBidangInput(input dto.BidangInput) (string, error) {
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

// pegawaiFields holds the normalized values ready to be persisted.
type pegawaiFields struct {
	BidangID     int64
	Nama         string
	NIP          string
	TanggalLahir string
	Alamat       string
	IsMale       bool
}

// validatePegawaiInput validates every field and returns normalized values.
// Business dates are converted to ISO-8601 before storage.
func validatePegawaiInput(input dto.PegawaiInput) (pegawaiFields, error) {
	fields := pegawaiFields{}
	errs := apperr.FieldErrors{}

	switch {
	case input.BidangID == 0:
		errs.Add("bidang_id", "Bidang wajib diisi.")
	case input.BidangID < 0:
		errs.Add("bidang_id", "Bidang tidak valid.")
	default:
		fields.BidangID = input.BidangID
	}

	fields.Nama = strings.TrimSpace(input.Nama)
	switch {
	case fields.Nama == "":
		errs.Add("nama", "Nama wajib diisi.")
	case utf8.RuneCountInString(fields.Nama) > maxNamaLength:
		errs.Add("nama", "Nama maksimal 150 karakter.")
	}

	fields.NIP = strings.TrimSpace(input.NIP)
	switch {
	case fields.NIP == "":
		errs.Add("nip", "NIP wajib diisi.")
	case utf8.RuneCountInString(fields.NIP) > maxNIPLength:
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
			fields.TanggalLahir = iso
		}
	}

	fields.Alamat = strings.TrimSpace(input.Alamat)
	switch {
	case fields.Alamat == "":
		errs.Add("alamat", "Alamat wajib diisi.")
	case utf8.RuneCountInString(fields.Alamat) > maxAlamatLength:
		errs.Add("alamat", "Alamat maksimal 500 karakter.")
	}

	jenisKelamin := strings.ToLower(strings.TrimSpace(string(input.JenisKelamin)))
	switch jenisKelamin {
	case string(dto.InputPria):
		fields.IsMale = true
	case string(dto.InputWanita):
		fields.IsMale = false
	case "":
		errs.Add("jenis_kelamin", "Jenis kelamin wajib diisi.")
	default:
		errs.Add("jenis_kelamin", "Jenis kelamin harus pria atau wanita.")
	}

	if errs.Any() {
		return fields, errs
	}
	return fields, nil
}

// parseTanggalLahir accepts DD-MM-YYYY or YYYY-MM-DD and returns the ISO-8601
// form used for storage. Dates after now.UTC's day are rejected.
func parseTanggalLahir(value string, now time.Time) (string, error) {
	var parsed time.Time
	var err error
	for _, layout := range []string{"02-01-2006", isoDateLayout} {
		parsed, err = time.ParseInLocation(layout, value, time.UTC)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", errInvalidDate
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if parsed.After(today) {
		return "", errFutureDate
	}
	return parsed.Format(isoDateLayout), nil
}

// formatTanggalLahir converts the stored ISO-8601 date to DD-MM-YYYY. Malformed
// storage values are returned unchanged so they remain visible during
// troubleshooting instead of silently disappearing.
func formatTanggalLahir(iso string) string {
	parsed, err := time.Parse(isoDateLayout, iso)
	if err != nil {
		return iso
	}
	return parsed.Format("02-01-2006")
}

// ValidateBidangInput validates a bidang payload without touching storage. The
// handler calls it before Create/Update so a structurally invalid payload (for
// example an unknown field) can collect semantic errors without writing.
func ValidateBidangInput(input dto.BidangInput) error {
	_, err := validateBidangInput(input)
	return err
}

// ValidatePegawaiInput validates a pegawai payload without touching storage.
func ValidatePegawaiInput(input dto.PegawaiInput) error {
	_, err := validatePegawaiInput(input)
	return err
}

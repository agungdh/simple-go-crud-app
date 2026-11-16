package pegawai

import (
	"errors"
	"time"
)

// isoDateLayout is how business dates are stored in SQLite.
const isoDateLayout = "2006-01-02"

var (
	errInvalidDate = errors.New("invalid date")
	errFutureDate  = errors.New("future date")
)

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

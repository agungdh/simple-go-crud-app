package auth

import (
	"strings"
	"unicode/utf8"

	"simple-go-crud-app/internal/shared/apperr"
)

const (
	maxUsernameLength = 50
	maxNameLength     = 150

	maxRefreshTokenLength = 512
)

// validateLoginInput trims the username and collects missing/oversized fields.
// Passwords are never trimmed.
func validateLoginInput(input LoginInput) (string, error) {
	username := strings.TrimSpace(input.Username)
	errs := apperr.FieldErrors{}

	switch {
	case username == "":
		errs.Add("username", "Username wajib diisi.")
	case utf8.RuneCountInString(username) > maxUsernameLength:
		errs.Add("username", "Username maksimal 50 karakter.")
	}

	switch {
	case input.Password == "":
		errs.Add("password", "Password wajib diisi.")
	case len(input.Password) > MaxPasswordLength:
		errs.Add("password", "Password maksimal 72 karakter.")
	}

	if errs.Any() {
		return username, errs
	}
	return username, nil
}

// validateRefreshTokenInput trims the opaque token and enforces the length
// documented by the contract.
func validateRefreshTokenInput(input RefreshTokenInput) (string, error) {
	token := strings.TrimSpace(input.RefreshToken)
	errs := apperr.FieldErrors{}

	switch {
	case token == "":
		errs.Add("refresh_token", "Refresh token wajib diisi.")
	case len(token) > maxRefreshTokenLength:
		errs.Add("refresh_token", "Refresh token maksimal 512 karakter.")
	}

	if errs.Any() {
		return token, errs
	}
	return token, nil
}

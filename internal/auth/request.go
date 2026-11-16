package auth

import (
	"encoding/json"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/httpx"
)

// decodeLoginInput maps the login body to the DTO and collects structural
// problems (unknown fields and wrong JSON types).
func decodeLoginInput(fields map[string]json.RawMessage) (LoginInput, apperr.FieldErrors) {
	errs := apperr.FieldErrors{}
	httpx.RejectUnknown(fields, map[string]struct{}{
		"username": {},
		"password": {},
	}, errs)

	return LoginInput{
		Username: httpx.DecodeString(fields, "username", "Username", errs),
		Password: httpx.DecodeString(fields, "password", "Password", errs),
	}, errs
}

// decodeRefreshTokenInput maps the refresh/logout body to the DTO and collects
// structural problems (unknown fields and wrong JSON types).
func decodeRefreshTokenInput(fields map[string]json.RawMessage) (RefreshTokenInput, apperr.FieldErrors) {
	errs := apperr.FieldErrors{}
	httpx.RejectUnknown(fields, map[string]struct{}{
		"refresh_token": {},
	}, errs)

	return RefreshTokenInput{
		RefreshToken: httpx.DecodeString(fields, "refresh_token", "Refresh token", errs),
	}, errs
}

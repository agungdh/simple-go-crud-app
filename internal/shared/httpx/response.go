package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/shared/apperr"
)

// Message constants are the safe, user-facing error messages used by every
// domain handler.
const (
	MessageBadBody            = "Body request tidak terbaca."
	MessageNotFound           = "Data dengan id tersebut tidak ditemukan."
	MessageInternal           = "Terjadi kesalahan di server."
	MessageInvalidCredentials = "Username atau password salah."
	MessageInvalidToken       = "Token tidak valid atau sudah kedaluwarsa."
	MessageAuthRequired       = "Autentikasi diperlukan."
	MessageForbidden          = "Akses ditolak."
	MessageLengthRequired     = "Header Content-Length wajib diisi."
	MessageTooLarge           = "Ukuran file melebihi batas."
)

// WriteJSON writes a JSON payload with the given status.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("marshal response: %v", err)
		WriteError(w, http.StatusInternalServerError, MessageInternal)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// WriteError writes the non-validation error envelope.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// WriteValidation writes the 422 envelope with per-field messages.
func WriteValidation(w http.ResponseWriter, validations apperr.FieldErrors) {
	if len(validations) == 0 {
		WriteError(w, http.StatusInternalServerError, MessageInternal)
		return
	}
	WriteJSON(w, http.StatusUnprocessableEntity, ValidationErrorResponse{
		Error: ValidationError{Validations: validations},
	})
}

// WriteServiceError maps domain errors to the status codes allowed by the
// contract. Anything that is not a validation or not-found error is logged and
// reported as a generic 500 with no internal details.
func WriteServiceError(w http.ResponseWriter, operation string, err error) {
	var validations apperr.FieldErrors
	if errors.As(err, &validations) && len(validations) > 0 {
		WriteValidation(w, validations)
		return
	}
	if errors.Is(err, apperr.ErrInvalidCredentials) {
		WriteError(w, http.StatusUnauthorized, MessageInvalidCredentials)
		return
	}
	if errors.Is(err, apperr.ErrInvalidToken) {
		WriteError(w, http.StatusUnauthorized, MessageInvalidToken)
		return
	}
	if errors.Is(err, apperr.ErrLengthRequired) {
		WriteError(w, http.StatusLengthRequired, MessageLengthRequired)
		return
	}
	if errors.Is(err, apperr.ErrBodyTooLarge) {
		WriteError(w, http.StatusRequestEntityTooLarge, MessageTooLarge)
		return
	}
	if errors.Is(err, apperr.ErrIncompleteBody) {
		WriteError(w, http.StatusBadRequest, MessageBadBody)
		return
	}
	if errors.Is(err, apperr.ErrNotFound) {
		WriteError(w, http.StatusNotFound, MessageNotFound)
		return
	}

	log.Printf("%s: %v", operation, err)
	WriteError(w, http.StatusInternalServerError, MessageInternal)
}

// ParseID reads the {id} path parameter. A non-positive or non-integer value is
// a 422, as documented by the contract.
func ParseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		WriteValidation(w, apperr.FieldErrors{
			"id": {"ID harus berupa bilangan bulat positif."},
		})
		return 0, false
	}
	return id, true
}

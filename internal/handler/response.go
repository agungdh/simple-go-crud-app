package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/dto"
)

const (
	messageBadBody  = "Body request tidak terbaca."
	messageNotFound = "Data dengan id tersebut tidak ditemukan."
	messageInternal = "Terjadi kesalahan di server."
)

// writeJSON writes a JSON payload with the given status.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("marshal response: %v", err)
		writeError(w, http.StatusInternalServerError, messageInternal)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// writeError writes the non-validation error envelope.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}

// writeValidation writes the 422 envelope with per-field messages.
func writeValidation(w http.ResponseWriter, validations apperr.FieldErrors) {
	if len(validations) == 0 {
		writeError(w, http.StatusInternalServerError, messageInternal)
		return
	}
	writeJSON(w, http.StatusUnprocessableEntity, dto.ValidationErrorResponse{
		Error: dto.ValidationError{Validations: validations},
	})
}

// writeServiceError maps layer errors to the status codes allowed by the
// contract. Anything that is not a validation or not-found error is logged and
// reported as a generic 500 with no internal details.
func writeServiceError(w http.ResponseWriter, operation string, err error) {
	var validations apperr.FieldErrors
	if errors.As(err, &validations) && len(validations) > 0 {
		writeValidation(w, validations)
		return
	}
	if errors.Is(err, apperr.ErrNotFound) {
		writeError(w, http.StatusNotFound, messageNotFound)
		return
	}

	log.Printf("%s: %v", operation, err)
	writeError(w, http.StatusInternalServerError, messageInternal)
}

// parseID reads the {id} path parameter. A non-positive or non-integer value is
// a 422, as documented by the contract.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeValidation(w, apperr.FieldErrors{
			"id": {"ID harus berupa bilangan bulat positif."},
		})
		return 0, false
	}
	return id, true
}

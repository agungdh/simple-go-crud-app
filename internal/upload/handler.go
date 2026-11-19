package upload

import (
	"net/http"

	"simple-go-crud-app/internal/shared/apperr"
	"simple-go-crud-app/internal/shared/httpx"
)

// Handler handles the upload endpoint.
type Handler struct {
	service *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// Create handles POST /upload. Content-Length is checked before the body is
// touched so an oversized upload is rejected without waiting for the transfer.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	limit := h.service.Limit()
	if err := checkContentLength(r, limit); err != nil {
		httpx.WriteServiceError(w, "upload file", err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, limit)

	stored, err := h.service.Store(r.Context(), r.Body, r.ContentLength)
	if err != nil {
		httpx.WriteServiceError(w, "upload file", err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, httpx.Envelope[Response]{
		Data: Response{FileKey: stored.Key, Size: stored.Size},
	})
}

// checkContentLength validates the declared size before any bytes are read.
func checkContentLength(r *http.Request, limit int64) error {
	switch {
	case r.ContentLength < 0:
		return apperr.ErrLengthRequired
	case r.ContentLength == 0:
		return apperr.ErrIncompleteBody
	case r.ContentLength > limit:
		return apperr.ErrBodyTooLarge
	default:
		return nil
	}
}

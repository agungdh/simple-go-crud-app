package upload

import (
	"io"
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

// Create handles POST /upload. The declared size is checked before the body is
// stored, so an oversized upload never touches the disk.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	limit := h.service.Limit()

	switch {
	case r.ContentLength < 0:
		httpx.WriteServiceError(w, "upload file", apperr.ErrLengthRequired)
		return
	case r.ContentLength == 0:
		httpx.WriteServiceError(w, "upload file", apperr.ErrIncompleteBody)
		return
	case r.ContentLength > limit:
		// Send the rejection before reading, then consume the rest of the body.
		// Closing the connection while a browser is still uploading makes the
		// request look pending, or end as a connection reset, instead of
		// delivering the 413 response.
		httpx.WriteServiceError(w, "upload file", apperr.ErrBodyTooLarge)
		flush(w)
		drain(r.Body)
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

// flush writes the response headers and body to the client immediately so the
// client can stop or finish the upload without waiting for the handler.
func flush(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// drain reads and discards the remaining request body. The whole declared body
// is consumed: some browsers only deliver the 413 after they finish sending, so
// stopping early would reset the connection before they see the response. The
// route read deadline (upload.ReadTimeout) bounds how long this can take.
// Errors are ignored on purpose: a client that already received the response
// often closes the connection, which makes the read fail.
func drain(body io.Reader) {
	_, _ = io.Copy(io.Discard, body)
}

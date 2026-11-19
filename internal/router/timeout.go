package router

import (
	"errors"
	"log"
	"net/http"
	"time"
)

// readTimeout extends the connection read deadline for a single route. The
// server keeps its default ReadTimeout for every other route.
func readTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller := http.NewResponseController(w)
			if err := controller.SetReadDeadline(time.Now().Add(timeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
				log.Printf("set read deadline: %v", err)
			}
			next.ServeHTTP(w, r)
		})
	}
}

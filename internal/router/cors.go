package router

import (
	"net/http"
	"strings"
)

// corsMiddleware allows browser clients served from the configured origins
// (Scalar docs by default) to call the API. Origins are matched exactly; no
// wildcard is used because the API does not enable credentials.
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The response body does not depend on Origin, but the CORS headers
			// do, so shared caches must key on it.
			w.Header().Add("Vary", "Origin")

			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok && origin != "" {
				header := w.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				header.Set("Access-Control-Allow-Headers", "Content-Type")
				header.Set("Access-Control-Max-Age", "300")

				// Answer the preflight before chi routing; OPTIONS is not part of
				// the documented contract.
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

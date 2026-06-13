package middleware

import "net/http"

// Auth is a no-op placeholder for future OAuth/JWT validation.
// All requests pass through until real authentication is implemented.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: validate bearer token / OAuth session before serving protected routes.
		next.ServeHTTP(w, r)
	})
}

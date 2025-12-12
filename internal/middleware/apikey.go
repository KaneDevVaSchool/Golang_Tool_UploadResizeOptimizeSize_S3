package middleware

import (
	"crypto/subtle"
	"net/http"
)

// APIKeyAuth middleware validates API key from header or query parameter
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				// No API key configured, allow all requests
				next.ServeHTTP(w, r)
				return
			}

			// Check X-API-Key header only (security: never accept API key from query parameters)
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"MISSING_API_KEY","message":"API key is required. Provide it via X-API-Key header."}}`))
				return
			}

			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(apiKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"success":false,"error":{"code":"INVALID_API_KEY","message":"Invalid API key."}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIKeyAuth middleware validate API key từ header
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				// * Không có API key configured, allow tất cả requests
				next.ServeHTTP(w, r)
				return
			}

			// Health probes must work without credentials (load balancer / k8s).
			if r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Trang public (khách ẩn danh xem triển lãm) không được có API key —
			// yêu cầu key ở đây coi như khoá cả trang public ra khỏi Internet.
			if strings.HasPrefix(r.URL.Path, "/api/v1/public/") {
				next.ServeHTTP(w, r)
				return
			}

			// ! Chỉ check X-API-Key header (không bao giờ accept từ query parameters)
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"MISSING_API_KEY","message":"API key is required. Provide it via X-API-Key header."}}`))
				return
			}

			// ! Dùng constant-time comparison để tránh timing attacks
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

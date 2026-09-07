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

			if isAPIKeyExempt(r.URL.Path) {
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

// isAPIKeyExempt: các đường không dùng X-API-Key.
//
// API key chỉ bảo vệ công cụ upload cũ (/api/v1/upload, /api/v1/metrics).
// Trang public, bộ lọc metadata, và khu admin (session cookie Google OAuth)
// nếu bị đòi key thì trình duyệt không có header đó → 401 hàng loạt, admin
// không vào được dù đã đăng nhập, phòng triển lãm mất khối lớp/chủ đề.
func isAPIKeyExempt(path string) bool {
	if path == "/api/v1/health" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/public/") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/admin/") {
		return true
	}
	switch path {
	case "/api/v1/schools", "/api/v1/grade-levels", "/api/v1/awards", "/api/v1/topic-categories":
		return true
	}
	return false
}

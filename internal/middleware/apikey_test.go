package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuth(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		apiKey     string // key cấu hình cho middleware, rỗng nghĩa là không bật
		path       string
		headerKey  string
		wantStatus int
	}{
		{
			name:       "không cấu hình API key thì cho qua mọi request",
			apiKey:     "",
			path:       "/api/v1/upload",
			headerKey:  "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "health luôn miễn key",
			apiKey:     "secret",
			path:       "/api/v1/health",
			headerKey:  "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "trang public miễn key — đây là chỗ P0.1 khoá lại, khách ẩn danh phải xem được",
			apiKey:     "secret",
			path:       "/api/v1/public/artworks",
			headerKey:  "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "trang public con (kèm id) cũng miễn key",
			apiKey:     "secret",
			path:       "/api/v1/public/artworks/123/reactions",
			headerKey:  "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "endpoint admin thiếu key thì 401",
			apiKey:     "secret",
			path:       "/api/v1/upload",
			headerKey:  "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "endpoint admin sai key thì 403",
			apiKey:     "secret",
			path:       "/api/v1/upload",
			headerKey:  "wrong-key",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "endpoint admin đúng key thì cho qua",
			apiKey:     "secret",
			path:       "/api/v1/upload",
			headerKey:  "secret",
			wantStatus: http.StatusOK,
		},
		{
			name:       "path chỉ chứa tiền tố 'public' nhưng không đúng /api/v1/public/ thì vẫn phải yêu cầu key",
			apiKey:     "secret",
			path:       "/api/v1/publicity",
			headerKey:  "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := APIKeyAuth(tt.apiKey)(okHandler)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.headerKey != "" {
				req.Header.Set("X-API-Key", tt.headerKey)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("path %s: got status %d, want %d (body: %s)", tt.path, rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

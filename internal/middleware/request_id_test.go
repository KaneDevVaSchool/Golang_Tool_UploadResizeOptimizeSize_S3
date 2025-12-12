package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Request ID should not be empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check header
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header should be set")
	}
}

func TestRequestIDFromHeader(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID != "custom-id-123" {
			t.Errorf("Expected 'custom-id-123', got '%s'", requestID)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		want      string
		wantEmpty bool
	}{
		{
			name: "with request ID",
			ctx:  context.WithValue(context.Background(), RequestIDKey, "test-id"),
			want: "test-id",
		},
		{
			name:      "without request ID",
			ctx:       context.Background(),
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRequestID(tt.ctx)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("GetRequestID() = %v, want empty", got)
				}
			} else {
				if got != tt.want {
					t.Errorf("GetRequestID() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

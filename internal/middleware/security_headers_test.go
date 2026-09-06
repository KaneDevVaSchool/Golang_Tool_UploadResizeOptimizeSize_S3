package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders_DatHeaderCoBan(t *testing.T) {
	h := SecurityHeadersMiddleware(SecurityHeadersConfig{})(okHandler())
	w := doRequest(h, "/phong-trien-lam", "Chrome", "203.0.113.20:1")

	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "SAMEORIGIN",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q, mong đợi %q", k, got, v)
		}
	}
}

func TestSecurityHeaders_CSPChoPhepAnhS3(t *testing.T) {
	// Quên khai domain S3 thì CSP chặn đúng ảnh tác phẩm - lỗi chỉ lộ trên
	// trình duyệt người dùng cuối, không thấy trong log server.
	h := SecurityHeadersMiddleware(SecurityHeadersConfig{
		ExtraImageSources: []string{"https://vas-pictures.s3.ap-southeast-1.amazonaws.com"},
	})(okHandler())
	w := doRequest(h, "/", "Chrome", "203.0.113.21:1")

	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "vas-pictures.s3.ap-southeast-1.amazonaws.com") {
		t.Fatalf("CSP phải cho phép domain S3, nhận được %q", csp)
	}
	if !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("script-src phải giữ nghiêm ngặt 'self', nhận được %q", csp)
	}
	// 'unsafe-inline' cho script là hướng tấn công XSS chính - không được nới.
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Fatal("script-src không được nới 'unsafe-inline'")
	}
}

func TestSecurityHeaders_KhongDatCSPTrenAPI(t *testing.T) {
	h := SecurityHeadersMiddleware(SecurityHeadersConfig{})(okHandler())
	w := doRequest(h, "/api/v1/public/artworks", "Chrome", "203.0.113.22:1")

	if csp := w.Header().Get("Content-Security-Policy"); csp != "" {
		t.Fatalf("response JSON không cần CSP, nhận được %q", csp)
	}
}

func TestSecurityHeaders_LocGiaTriCSPHong(t *testing.T) {
	// Một giá trị chứa dấu chấm phẩy sẽ tách directive và phá vỡ cả chính sách.
	h := SecurityHeadersMiddleware(SecurityHeadersConfig{
		ExtraImageSources: []string{"https://ok.example.com", "https://xau.example.com; script-src *"},
	})(okHandler())
	w := doRequest(h, "/", "Chrome", "203.0.113.23:1")

	csp := w.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "script-src *") {
		t.Fatalf("giá trị cấu hình hỏng phải bị loại, nhận được %q", csp)
	}
	if !strings.Contains(csp, "https://ok.example.com") {
		t.Fatalf("giá trị hợp lệ vẫn phải giữ, nhận được %q", csp)
	}
}

func TestSecurityHeaders_HSTSChiKhiHTTPS(t *testing.T) {
	withTrustedProxies(t, []string{"127.0.0.0/8"})
	h := SecurityHeadersMiddleware(SecurityHeadersConfig{EnableHSTS: true})(okHandler())

	// Request HTTP thuần: bật HSTS ở đây sẽ khoá trình duyệt khỏi site.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.24:1"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HTTP thuần không được đặt HSTS, nhận được %q", got)
	}

	// Qua Nginx với X-Forwarded-Proto: https thì mới đặt.
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:8080"
	r.Header.Set("X-Forwarded-Proto", "https")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("request HTTPS qua proxy tin cậy phải có HSTS")
	}
}

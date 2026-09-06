package middleware

import (
	"net/http"
	"strings"
)

// Security headers đặt ở tầng ứng dụng thay vì chỉ ở Nginx.
//
// Vì sao không để Nginx lo hết: vhost mẫu trong repo chỉ là điểm khởi đầu,
// còn file thật trên VPS do Certbot sửa và người vận hành chỉnh tay - repo
// không kiểm soát được nội dung đó. Đặt ở đây thì mọi lần triển khai đều có,
// kể cả khi ai đó dựng vhost mới hoặc chạy app trực tiếp không qua proxy.
//
// Nginx hiện đặt X-Frame-Options/X-Content-Type-Options/Referrer-Policy bằng
// add_header. Trùng header không gây lỗi (giá trị giống nhau), nhưng để tránh
// gửi hai lần, phần dưới chỉ đặt khi header chưa tồn tại.

// SecurityHeadersConfig cấu hình các header phụ thuộc môi trường triển khai.
type SecurityHeadersConfig struct {
	// EnableHSTS chỉ nên bật khi site đã chạy HTTPS hoàn toàn. Bật khi còn
	// phục vụ HTTP sẽ khoá trình duyệt khỏi site trong suốt max-age.
	EnableHSTS bool
	// ExtraImageSources là các origin ngoài được phép tải ảnh - bắt buộc khai
	// báo domain S3/CDN, nếu không CSP sẽ chặn chính ảnh tác phẩm.
	ExtraImageSources []string
	// ExtraConnectSources là origin được phép gọi XHR/fetch tới.
	ExtraConnectSources []string
}

// SecurityHeadersMiddleware đặt các header phòng thủ cho mọi response.
func SecurityHeadersMiddleware(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	csp := buildCSP(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			setIfAbsent(h, "X-Content-Type-Options", "nosniff")
			setIfAbsent(h, "X-Frame-Options", "SAMEORIGIN")
			setIfAbsent(h, "Referrer-Policy", "strict-origin-when-cross-origin")

			// Khoá sẵn các API trình duyệt mà trang này không bao giờ dùng, để
			// một đoạn script chèn được vào cũng không xin được quyền.
			setIfAbsent(h, "Permissions-Policy",
				"camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()")

			// Chặn trình duyệt cũ đọc tài nguyên của site này từ ngữ cảnh khác.
			setIfAbsent(h, "Cross-Origin-Resource-Policy", "same-origin")
			setIfAbsent(h, "Cross-Origin-Opener-Policy", "same-origin")

			if cfg.EnableHSTS && isHTTPSRequest(r) {
				setIfAbsent(h, "Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// CSP chỉ áp cho tài liệu HTML. Đặt lên response JSON/ảnh là vô
			// nghĩa và chỉ tốn băng thông ở mọi request.
			if csp != "" && isHTMLRoute(r.URL.Path) {
				setIfAbsent(h, "Content-Security-Policy", csp)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// buildCSP dựng chính sách cho SPA React + ảnh trên S3.
//
// 'unsafe-inline' cho style là bắt buộc: React đặt style nội tuyến qua thuộc
// tính style, và trang chia sẻ render bằng html/template cũng có style nội
// tuyến. Với script thì KHÔNG nới - Vite sinh file .js riêng, không cần inline,
// nên giữ được 'self' nghiêm ngặt, và đó mới là hướng tấn công XSS đáng lo.
func buildCSP(cfg SecurityHeadersConfig) string {
	imgSrc := []string{"'self'", "data:", "blob:"}
	imgSrc = append(imgSrc, sanitizeSources(cfg.ExtraImageSources)...)

	connectSrc := []string{"'self'"}
	connectSrc = append(connectSrc, sanitizeSources(cfg.ExtraConnectSources)...)

	directives := []string{
		"default-src 'self'",
		"base-uri 'self'",
		"object-src 'none'",
		"frame-ancestors 'self'",
		"form-action 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"font-src 'self' data:",
		"img-src " + strings.Join(imgSrc, " "),
		"connect-src " + strings.Join(connectSrc, " "),
	}

	return strings.Join(directives, "; ")
}

func sanitizeSources(sources []string) []string {
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		v := strings.TrimSpace(s)
		// Dấu chấm phẩy và khoảng trắng sẽ tách directive - loại bỏ để một
		// giá trị cấu hình sai không phá vỡ toàn bộ chính sách.
		if v == "" || strings.ContainsAny(v, "; \t\n\r") {
			continue
		}
		out = append(out, v)
	}
	return out
}

func setIfAbsent(h http.Header, key, value string) {
	if h.Get(key) == "" {
		h.Set(key, value)
	}
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// Sau Nginx, TLS kết thúc ở proxy nên r.TLS luôn nil. Header này chỉ đáng
	// tin khi request đến từ proxy tin cậy - cùng lý do đã nêu ở clientip.go.
	if isTrustedProxy(remoteIP(r)) {
		return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	}
	return false
}

// isHTMLRoute đoán response có phải tài liệu HTML không, dựa trên đường dẫn.
// SPA trả index.html cho mọi route không khớp file, nên mặc định là có, trừ
// các nhánh chắc chắn không phải HTML.
func isHTMLRoute(path string) bool {
	if strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/assets/") ||
		strings.HasPrefix(path, "/images/") ||
		strings.HasPrefix(path, "/fonts/") {
		return false
	}
	switch path {
	case "/robots.txt", "/sitemap.xml", "/favicon.ico":
		return false
	}
	return true
}

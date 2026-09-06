package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

const (
	csrfTokenLength   = 32
	csrfCookieName    = "csrf_token"
	csrfHeaderName    = "X-CSRF-Token"
	csrfFormFieldName = "csrf_token"
	csrfCookieMaxAge  = 3600
)

// csrfContextKey là kiểu riêng cho khoá context, tránh va chạm với khoá do
// package khác đặt vào cùng một context.
type csrfContextKey struct{}

var csrfTokenContextKey = csrfContextKey{}

// CSRFTokenFromContext đọc lại token đã sinh ở nhánh GET.
func CSRFTokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(csrfTokenContextKey).(string); ok {
		return token
	}
	return ""
}

// CSRFProtection cung cấp CSRF protection dùng double-submit cookie pattern
type CSRFProtection struct {
	cookieName    string
	headerName    string
	formFieldName string
	cookieMaxAge  int
	secureCookie  bool
}

// NewCSRFProtection tạo CSRF protection middleware mới
func NewCSRFProtection(secureCookie bool) *CSRFProtection {
	return &CSRFProtection{
		cookieName:    csrfCookieName,
		headerName:    csrfHeaderName,
		formFieldName: csrfFormFieldName,
		cookieMaxAge:  csrfCookieMaxAge,
		secureCookie:  secureCookie,
	}
}

// generateToken tạo cryptographically secure random token
func generateToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// setCSRFCookie set CSRF token cookie
func (c *CSRFProtection) setCSRFCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     c.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   c.cookieMaxAge,
		HttpOnly: false, // * Phải readable bởi JavaScript cho double-submit pattern
		SameSite: http.SameSiteStrictMode,
		Secure:   c.secureCookie,
	}
	http.SetCookie(w, cookie)
}

// getCSRFCookie lấy CSRF token từ cookie
func (c *CSRFProtection) getCSRFCookie(r *http.Request) string {
	cookie, err := r.Cookie(c.cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getCSRFToken lấy CSRF token từ request (header hoặc form)
func (c *CSRFProtection) getCSRFToken(r *http.Request) string {
	// * Kiểm tra header trước (cho AJAX requests)
	if token := r.Header.Get(c.headerName); token != "" {
		return token
	}

	// Chỉ đọc form khi body là form thường. Với multipart (đường upload ảnh),
	// r.FormValue sẽ parse toàn bộ body - nghĩa là một file 200MB được đọc và
	// ghi ra đĩa tạm TRƯỚC khi biết token có hợp lệ hay không, biến chính lớp
	// chống CSRF thành đường làm cạn tài nguyên. Upload của dự án gửi token
	// qua header nên nhánh này không cần thiết cho multipart.
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if token := r.PostFormValue(c.formFieldName); token != "" {
			return token
		}
	}

	return ""
}

// validateCSRF validate CSRF token
func (c *CSRFProtection) validateCSRF(r *http.Request) bool {
	cookieToken := c.getCSRFCookie(r)
	requestToken := c.getCSRFToken(r)

	if cookieToken == "" || requestToken == "" {
		return false
	}

	// So sánh constant-time: so sánh chuỗi thường thoát ra ở ký tự lệch đầu
	// tiên, để lộ độ dài tiền tố đúng qua thời gian phản hồi và cho phép dò
	// dần từng ký tự của token.
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) == 1
}

// CSRFMiddleware tạo middleware bảo vệ chống CSRF attacks
func (c *CSRFProtection) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// * Skip CSRF cho GET, HEAD, OPTIONS requests
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			token := c.getCSRFCookie(r)
			if token == "" {
				var err error
				token, err = generateToken()
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"success":false,"error":{"code":"CSRF_INIT_FAILED","message":"Unable to initialize security token."}}`))
					return
				}
				c.setCSRFCookie(w, token)
			}
			// * Add token vào context để template có thể access.
			// Key dùng kiểu riêng (csrfContextKey) chứ không phải string trần:
			// string trần có thể trùng key do package khác đặt vào cùng context.
			ctx := context.WithValue(r.Context(), csrfTokenContextKey, token)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		// ! Với state-changing methods, phải validate CSRF token
		if !c.validateCSRF(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"INVALID_CSRF","message":"Invalid CSRF token. Refresh the page and try again."}}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}


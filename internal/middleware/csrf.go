package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

const (
	csrfTokenLength   = 32
	csrfCookieName    = "csrf_token"
	csrfHeaderName    = "X-CSRF-Token"
	csrfFormFieldName = "csrf_token"
	csrfCookieMaxAge  = 3600
)

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

	// * Kiểm tra form field (cho form submissions)
	if token := r.FormValue(c.formFieldName); token != "" {
		return token
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

	// * Tokens phải match chính xác (double-submit cookie pattern)
	return cookieToken == requestToken
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
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				c.setCSRFCookie(w, token)
			}
			// * Add token vào context để template có thể access
			ctx := r.Context()
			ctx = context.WithValue(ctx, "csrf_token", token)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		// ! Với state-changing methods, phải validate CSRF token
		if !c.validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetCSRFToken lấy CSRF token để dùng trong templates
func (c *CSRFProtection) GetCSRFToken(r *http.Request) string {
	token := c.getCSRFCookie(r)
	if token == "" {
		newToken, err := generateToken()
		if err != nil {
			return ""
		}
		token = newToken
	}
	return token
}

// Package auth chứa luồng đăng nhập admin qua Google OAuth và quản lý
// session (DB-backed, không in-memory) dùng chung cho toàn bộ admin panel.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"s3-upload-tool/internal/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// oauthStateCookieName lưu state CSRF tạm thời (5 phút) trước khi redirect
// sang Google, so khớp lại khi Google gọi về /auth/google/callback.
const (
	oauthStateCookieName = "vas_oauth_state"
	oauthStateTTL        = 5 * time.Minute
)

// NewGoogleOAuthConfig dựng *oauth2.Config từ AuthConfig. Trả về config dù
// ClientID/Secret rỗng - IsGoogleOAuthConfigured dùng để handler tự kiểm tra
// và trả 503 rõ ràng thay vì gọi Google với credentials rỗng.
func NewGoogleOAuthConfig(cfg config.AuthConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// IsGoogleOAuthConfigured báo server đã có đủ Client ID/Secret để dùng
// OAuth thật hay chưa.
func IsGoogleOAuthConfigured(oc *oauth2.Config) bool {
	return oc != nil && oc.ClientID != "" && oc.ClientSecret != ""
}

// GenerateState sinh 1 chuỗi ngẫu nhiên base64 url-safe dùng làm state CSRF
// cho luồng OAuth (chống cross-site request forgery cho callback).
func GenerateState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// SetOAuthStateCookie đặt cookie state tạm thời trước khi redirect sang Google.
func SetOAuthStateCookie(w http.ResponseWriter, state string, secureCookie bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode, // Lax vì đây là redirect flow xuyên site (Google -> app)
	})
}

// ReadAndClearOAuthStateCookie đọc cookie state để so khớp với query param
// "state" Google gửi về, đồng thời xoá cookie ngay sau khi đọc (dùng 1 lần).
func ReadAndClearOAuthStateCookie(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(oauthStateCookieName)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

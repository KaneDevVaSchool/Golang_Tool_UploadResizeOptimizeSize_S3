package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"s3-upload-tool/internal/config"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
)

// SessionManager tạo/kiểm tra/xoá admin session, lưu DB (không in-memory)
// để sống sót qua restart server - khác với ChunkUploadService (in-memory
// có chủ đích vì chunk session chỉ tồn tại trong 1 lần upload ngắn hạn).
type SessionManager struct {
	repo repository.SessionRepository
	cfg  config.AuthConfig
}

func NewSessionManager(repo repository.SessionRepository, cfg config.AuthConfig) *SessionManager {
	return &SessionManager{repo: repo, cfg: cfg}
}

// CreateSession sinh 1 token ngẫu nhiên 32-byte (base64 url-safe), lưu vào
// admin_sessions với TTL từ AuthConfig.SessionTTL.
func (m *SessionManager) CreateSession(ctx context.Context, adminUserID int64) (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	expiresAt := time.Now().Add(m.cfg.SessionTTL)

	session := &models.AdminSession{
		ID:          token,
		AdminUserID: adminUserID,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
	if err := m.repo.Create(ctx, session); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// ValidateSession trả về session + admin user tương ứng nếu token hợp lệ và
// chưa hết hạn. Trả (nil, nil, nil) nếu token không tồn tại/đã hết hạn -
// caller (middleware) tự quyết định trả 401.
func (m *SessionManager) ValidateSession(ctx context.Context, token string) (*models.AdminSession, *models.AdminUser, error) {
	if token == "" {
		return nil, nil, nil
	}
	session, user, err := m.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	if session == nil || user == nil {
		return nil, nil, nil
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, nil, nil
	}
	if !user.IsActive {
		return nil, nil, nil
	}
	return session, user, nil
}

// DeleteSession xoá session khỏi DB - dùng cho logout.
func (m *SessionManager) DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return m.repo.Delete(ctx, token)
}

// SetSessionCookie đặt cookie HttpOnly chứa token session.
func (m *SessionManager) SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   m.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode, // Lax vì callback đến từ redirect Google (cross-site GET)
	})
}

// ClearSessionCookie xoá cookie session phía client - dùng khi logout hoặc
// session không còn hợp lệ.
func (m *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadSessionCookie đọc token session từ cookie request, trả "" nếu không có.
func (m *SessionManager) ReadSessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(m.cfg.SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

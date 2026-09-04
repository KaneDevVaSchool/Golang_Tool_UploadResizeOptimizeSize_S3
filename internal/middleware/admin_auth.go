package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"s3-upload-tool/internal/auth"
	"s3-upload-tool/internal/models"
)

// AdminUserContextKey lưu *models.AdminUser đang đăng nhập vào request
// context - đọc lại bằng GetAdminUser(ctx), theo đúng pattern GetRequestID.
const AdminUserContextKey = "admin_user"

// AdminAuthMiddleware đọc cookie session, xác thực qua SessionManager, và
// set AdminUser vào context nếu hợp lệ. Endpoint dưới /api/v1/admin/* luôn
// trả 401 JSON khi thiếu/hết hạn session (không redirect browser - đây là
// API JSON, FE tự điều hướng qua React Router khi nhận 401).
func AdminAuthMiddleware(sessionMgr *auth.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := sessionMgr.ReadSessionCookie(r)
			_, user, err := sessionMgr.ValidateSession(r.Context(), token)
			if err != nil || user == nil {
				writeUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), AdminUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAdminUser lấy admin user đang đăng nhập từ context - trả nil nếu
// request không đi qua AdminAuthMiddleware hoặc chưa xác thực.
func GetAdminUser(ctx context.Context) *models.AdminUser {
	if user, ok := ctx.Value(AdminUserContextKey).(*models.AdminUser); ok {
		return user
	}
	return nil
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]string{
			"code":    "UNAUTHORIZED",
			"message": "Phiên đăng nhập không hợp lệ hoặc đã hết hạn",
		},
	})
}

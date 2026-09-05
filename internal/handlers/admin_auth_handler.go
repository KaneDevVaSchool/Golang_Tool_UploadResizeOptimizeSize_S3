package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"s3-upload-tool/internal/auth"
	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"

	"golang.org/x/oauth2"
)

// googleUserInfoURL trả về sub/email/name/picture cho access token vừa đổi được.
const googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// googleUserInfo là phần response cần dùng từ Google UserInfo endpoint.
type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// AdminAuthHandler xử lý luồng Google OAuth (redirect-based, mount dưới
// /auth/*) và các endpoint JSON liên quan tới phiên đăng nhập admin (mount
// dưới /api/v1/admin/auth/*).
type AdminAuthHandler struct {
	BaseHandler
	oauthConfig      *oauth2.Config
	sessionMgr       *auth.SessionManager
	adminUserRepo    repository.AdminUserRepository
	allowedDomains   []string
	allowedEmails    map[string]bool
	secureCookie     bool
	frontendAdminURL string
	frontendLoginURL string
}

func NewAdminAuthHandler(
	oauthConfig *oauth2.Config,
	sessionMgr *auth.SessionManager,
	adminUserRepo repository.AdminUserRepository,
	allowedDomains []string,
	allowedEmails []string,
	secureCookie bool,
) *AdminAuthHandler {
	normalized := make([]string, 0, len(allowedDomains))
	for _, d := range allowedDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			normalized = append(normalized, d)
		}
	}
	emailSet := make(map[string]bool, len(allowedEmails))
	for _, e := range allowedEmails {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" {
			emailSet[e] = true
		}
	}
	return &AdminAuthHandler{
		oauthConfig:      oauthConfig,
		sessionMgr:       sessionMgr,
		adminUserRepo:    adminUserRepo,
		allowedDomains:   normalized,
		allowedEmails:    emailSet,
		secureCookie:     secureCookie,
		frontendAdminURL: "/admin",
		frontendLoginURL: "/admin/login",
	}
}

// isEmailAllowed quyết định email có được phép đăng nhập admin hay không.
//
// Whitelist email cụ thể (ADMIN_ALLOWED_EMAILS) có độ ưu tiên cao nhất: khi
// nó không rỗng thì chỉ đúng các email trong đó mới vào được, bỏ qua hoàn
// toàn kiểm tra domain. Chỉ khi whitelist rỗng mới rơi về kiểm tra domain
// (ADMIN_ALLOWED_EMAIL_DOMAIN), và domain cũng rỗng nghĩa là không giới hạn.
func (h *AdminAuthHandler) isEmailAllowed(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))

	if len(h.allowedEmails) > 0 {
		return h.allowedEmails[email]
	}

	if len(h.allowedDomains) == 0 {
		return true
	}
	for _, domain := range h.allowedDomains {
		if strings.HasSuffix(email, "@"+domain) {
			return true
		}
	}
	return false
}

// HandleGoogleLogin bắt đầu luồng OAuth: sinh state CSRF, redirect sang
// Google. Trả 503 rõ ràng nếu server chưa cấu hình Client ID/Secret (chưa
// chặn toàn bộ server khởi động - chỉ endpoint này báo lỗi).
func (h *AdminAuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !auth.IsGoogleOAuthConfigured(h.oauthConfig) {
		h.SendError(w, http.StatusServiceUnavailable, "OAUTH_NOT_CONFIGURED",
			"Đăng nhập Google chưa được cấu hình trên máy chủ. Vui lòng thiết lập GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET trong .env.")
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		log.Printf("[AdminAuth] Failed to generate oauth state: %v", err)
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không thể khởi tạo phiên đăng nhập")
		return
	}

	auth.SetOAuthStateCookie(w, state, h.secureCookie)
	http.Redirect(w, r, h.oauthConfig.AuthCodeURL(state), http.StatusFound)
}

// HandleGoogleCallback đổi code lấy token, lấy thông tin user từ Google,
// kiểm tra domain email nếu có giới hạn, upsert admin_users, tạo session,
// set cookie, rồi redirect về trang admin của FE.
func (h *AdminAuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !auth.IsGoogleOAuthConfigured(h.oauthConfig) {
		h.SendError(w, http.StatusServiceUnavailable, "OAUTH_NOT_CONFIGURED", "Đăng nhập Google chưa được cấu hình trên máy chủ.")
		return
	}

	expectedState, hadCookie := auth.ReadAndClearOAuthStateCookie(w, r)
	state := r.URL.Query().Get("state")
	if !hadCookie || state == "" || state != expectedState {
		h.redirectLoginWithError(w, r, "invalid_state")
		return
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		h.redirectLoginWithError(w, r, errParam)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectLoginWithError(w, r, "missing_code")
		return
	}

	ctx := r.Context()
	token, err := h.oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("[AdminAuth] Token exchange failed: %v", err)
		h.redirectLoginWithError(w, r, "exchange_failed")
		return
	}

	client := h.oauthConfig.Client(ctx, token)
	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		log.Printf("[AdminAuth] Failed to fetch userinfo: %v", err)
		h.redirectLoginWithError(w, r, "userinfo_failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AdminAuth] Userinfo endpoint returned status %d", resp.StatusCode)
		h.redirectLoginWithError(w, r, "userinfo_failed")
		return
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		log.Printf("[AdminAuth] Failed to decode userinfo: %v", err)
		h.redirectLoginWithError(w, r, "userinfo_failed")
		return
	}

	if info.Sub == "" || info.Email == "" {
		h.redirectLoginWithError(w, r, "incomplete_profile")
		return
	}

	// Whitelist so khớp theo email nên bắt buộc email phải được Google xác
	// thực - tránh trường hợp account đặt email trùng danh sách mà chưa verify.
	if !info.EmailVerified {
		log.Printf("[AdminAuth] Từ chối đăng nhập (email chưa xác thực): %s", info.Email)
		h.redirectLoginWithError(w, r, "email_not_verified")
		return
	}

	if !h.isEmailAllowed(info.Email) {
		log.Printf("[AdminAuth] Từ chối đăng nhập (email không nằm trong danh sách cho phép): %s", info.Email)
		h.redirectLoginWithError(w, r, "email_not_allowed")
		return
	}

	user, err := h.adminUserRepo.FindByGoogleSub(ctx, info.Sub)
	if err != nil {
		log.Printf("[AdminAuth] Failed to lookup admin user: %v", err)
		h.redirectLoginWithError(w, r, "server_error")
		return
	}

	if user == nil {
		user, err = h.adminUserRepo.Create(ctx, &models.AdminUser{
			GoogleSub: info.Sub,
			Email:     info.Email,
			Name:      info.Name,
			AvatarURL: info.Picture,
			Role:      models.AdminRoleAdmin,
		})
		if err != nil {
			log.Printf("[AdminAuth] Failed to create admin user: %v", err)
			h.redirectLoginWithError(w, r, "server_error")
			return
		}
		log.Printf("[AdminAuth] Tạo admin user mới: %s", info.Email)
	} else {
		// Chặn account bị vô hiệu hoá trước khi ghi last_login_at - lần đăng
		// nhập bị từ chối không nên được ghi nhận như đăng nhập thành công.
		if !user.IsActive {
			h.redirectLoginWithError(w, r, "account_disabled")
			return
		}
		if err := h.adminUserRepo.UpdateLastLogin(ctx, user.ID); err != nil {
			log.Printf("[AdminAuth] Failed to update last_login_at: %v", err)
		}
	}

	sessionToken, expiresAt, err := h.sessionMgr.CreateSession(ctx, user.ID)
	if err != nil {
		log.Printf("[AdminAuth] Failed to create session: %v", err)
		h.redirectLoginWithError(w, r, "server_error")
		return
	}

	h.sessionMgr.SetSessionCookie(w, sessionToken, expiresAt)
	http.Redirect(w, r, h.frontendAdminURL, http.StatusFound)
}

func (h *AdminAuthHandler) redirectLoginWithError(w http.ResponseWriter, r *http.Request, reason string) {
	http.Redirect(w, r, fmt.Sprintf("%s?error=%s", h.frontendLoginURL, reason), http.StatusFound)
}

// HandleLogout xoá session hiện tại (DB) và clear cookie. Gọi từ FE sau khi
// người dùng xác nhận trong ConfirmDialog - đây là API call thường (không
// redirect browser trực tiếp như luồng OAuth).
func (h *AdminAuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}

	token := h.sessionMgr.ReadSessionCookie(r)
	if err := h.sessionMgr.DeleteSession(r.Context(), token); err != nil {
		log.Printf("[AdminAuth] Failed to delete session: %v", err)
	}
	h.sessionMgr.ClearSessionCookie(w)
	h.SendSuccess(w, map[string]bool{"logged_out": true})
}

// HandleMe trả thông tin admin đang đăng nhập - dùng cho FE hiển thị header
// user info và tự bảo vệ route (401 nếu chưa đăng nhập, xử lý ở middleware).
func (h *AdminAuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAdminUser(r.Context())
	if user == nil {
		h.SendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Chưa đăng nhập")
		return
	}
	h.SendSuccess(w, user)
}

package models

import "time"

// AdminUser đại diện cho 1 tài khoản quản trị đăng nhập qua Google OAuth.
type AdminUser struct {
	ID          int64      `db:"id" json:"id"`
	GoogleSub   string     `db:"google_sub" json:"-"`
	Email       string     `db:"email" json:"email"`
	Name        string     `db:"name" json:"name"`
	AvatarURL   string     `db:"avatar_url" json:"avatar_url"`
	Role        string     `db:"role" json:"role"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	LastLoginAt *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// Vai trò admin - hiện chỉ dùng AdminRoleAdmin, để sẵn chỗ mở rộng sau này.
const (
	AdminRoleAdmin      = "admin"
	AdminRoleSuperAdmin = "super_admin"
)

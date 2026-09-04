package models

import "time"

// AdminSession đại diện cho 1 phiên đăng nhập admin, lưu DB (không in-memory)
// để sống sót qua restart server. ID chính là token ngẫu nhiên đặt trong cookie.
type AdminSession struct {
	ID          string    `db:"id"`
	AdminUserID int64     `db:"admin_user_id"`
	ExpiresAt   time.Time `db:"expires_at"`
	CreatedAt   time.Time `db:"created_at"`
}

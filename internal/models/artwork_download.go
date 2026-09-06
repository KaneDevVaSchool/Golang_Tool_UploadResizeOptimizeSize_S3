package models

import "time"

// Nguồn tải ảnh gốc - phân biệt vì chỉ "admin" có AdminUserID định danh được
// người tải; "public" phải dựa vào IPAddress khi cần tra cứu.
const (
	ArtworkDownloadSourceAdmin  = "admin"
	ArtworkDownloadSourcePublic = "public"
)

// ArtworkDownload ghi lại 1 lượt tải ảnh gốc (mỗi lần tải thành công = 1
// dòng) - phục vụ truy vết khi ảnh bị phát tán sai mục đích.
type ArtworkDownload struct {
	ID           int64     `db:"id" json:"id"`
	ArtworkID    int64     `db:"artwork_id" json:"artwork_id"`
	AdminUserID  *int64    `db:"admin_user_id" json:"admin_user_id,omitempty"`
	Source       string    `db:"source" json:"source"`
	IPAddress    string    `db:"ip_address" json:"-"`
	UserAgent    string    `db:"user_agent" json:"-"`
	DownloadedAt time.Time `db:"downloaded_at" json:"downloaded_at"`
}

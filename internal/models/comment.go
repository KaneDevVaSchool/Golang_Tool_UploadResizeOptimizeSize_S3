package models

import "time"

// ArtworkComment là 1 bình luận ẩn danh gắn cho 1 tác phẩm. Người xem tự
// nhập display_name khi bình luận - không xác thực danh tính thật.
//
// IsHidden xuất ra JSON (dùng cho màn hình kiểm duyệt ở admin) nhưng
// PublicHandler không bao giờ serialize struct này trực tiếp - map thủ công
// sang publicCommentResponse (xem public_handler.go) nên trường này không lộ
// ra API public.
type ArtworkComment struct {
	ID           int64     `db:"id" json:"id"`
	ArtworkID    int64     `db:"artwork_id" json:"artwork_id"`
	DisplayName  string    `db:"display_name" json:"display_name"`
	Content      string    `db:"content" json:"content"`
	VisitorToken string    `db:"visitor_token" json:"-"`
	IPAddress    string    `db:"ip_address" json:"-"`
	IsHidden     bool      `db:"is_hidden" json:"is_hidden"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

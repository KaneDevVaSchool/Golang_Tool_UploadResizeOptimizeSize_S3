package models

import "time"

// Award đại diện cho 1 loại giải thưởng cấu hình được (tên, màu, icon, thứ tự).
//
// GradeLevelID gắn giải với 1 khối lớp cụ thể - hội thi chia giải riêng theo
// khối (vd Tiểu học: mỗi khối 1-5 có 1 Nhất/1 Nhì/2 Ba, tổng 20 giải). nil =
// giải dùng chung toàn hệ thống, không tách khối (vd giải "Đặc biệt" chọn
// 20-24 tác phẩm tiêu biểu toàn hệ thống, không phân biệt khối lớp).
type Award struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Slug         string    `db:"slug" json:"slug"`
	GradeLevelID *int64    `db:"grade_level_id" json:"grade_level_id,omitempty"`
	RankOrder    int       `db:"rank_order" json:"rank_order"`
	ColorHex     string    `db:"color_hex" json:"color_hex"`
	IconKey      *string   `db:"icon_key" json:"icon_key,omitempty"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

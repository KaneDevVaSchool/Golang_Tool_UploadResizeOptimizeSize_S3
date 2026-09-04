package models

import "time"

// Award đại diện cho 1 loại giải thưởng cấu hình được (tên, màu, icon, thứ tự).
type Award struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Slug      string    `db:"slug" json:"slug"`
	RankOrder int       `db:"rank_order" json:"rank_order"`
	ColorHex  string    `db:"color_hex" json:"color_hex"`
	IconKey   *string   `db:"icon_key" json:"icon_key,omitempty"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

package models

import "time"

// ArtworkView ghi lại 1 lượt xem để chống đếm trùng - trước khi tăng
// artworks.view_count, service kiểm tra visitor_token đã xem trong 24h chưa.
type ArtworkView struct {
	ID           int64     `db:"id" json:"id"`
	ArtworkID    int64     `db:"artwork_id" json:"artwork_id"`
	VisitorToken string    `db:"visitor_token" json:"-"`
	ViewedAt     time.Time `db:"viewed_at" json:"viewed_at"`
}

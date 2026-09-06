package models

import "time"

// ArtworkView ghi lại 1 lượt xem (mỗi lần mở chi tiết tác phẩm = 1 dòng).
type ArtworkView struct {
	ID           int64     `db:"id" json:"id"`
	ArtworkID    int64     `db:"artwork_id" json:"artwork_id"`
	VisitorToken string    `db:"visitor_token" json:"-"`
	ViewedAt     time.Time `db:"viewed_at" json:"viewed_at"`
}

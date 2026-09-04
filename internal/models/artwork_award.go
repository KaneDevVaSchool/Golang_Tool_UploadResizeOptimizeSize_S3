package models

import "time"

// ArtworkAward là bảng liên kết N-N giữa artworks và awards. UI hiện tại
// chỉ cần 1 giải/tác phẩm nhưng thiết kế N-N để mở rộng về sau.
type ArtworkAward struct {
	ID        int64      `db:"id" json:"id"`
	ArtworkID int64      `db:"artwork_id" json:"artwork_id"`
	AwardID   int64      `db:"award_id" json:"award_id"`
	AwardedAt *time.Time `db:"awarded_at" json:"awarded_at,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

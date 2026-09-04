package models

import "time"

// ArtworkReaction là 1 cảm xúc ẩn danh gắn cho 1 tác phẩm. Định danh qua
// visitor_token (UUID sinh ở trình duyệt) - không phải xác thực danh tính thật.
type ArtworkReaction struct {
	ID           int64     `db:"id" json:"id"`
	ArtworkID    int64     `db:"artwork_id" json:"artwork_id"`
	ReactionType string    `db:"reaction_type" json:"reaction_type"`
	VisitorToken string    `db:"visitor_token" json:"-"`
	IPAddress    string    `db:"ip_address" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// 6 loại cảm xúc, đúng bộ Facebook-style dùng ở va-workspace.
const (
	ReactionLike  = "like"
	ReactionLove  = "love"
	ReactionHaha  = "haha"
	ReactionWow   = "wow"
	ReactionSad   = "sad"
	ReactionAngry = "angry"
)

// ValidReactionTypes liệt kê đầy đủ để validate input phía handler.
var ValidReactionTypes = map[string]bool{
	ReactionLike:  true,
	ReactionLove:  true,
	ReactionHaha:  true,
	ReactionWow:   true,
	ReactionSad:   true,
	ReactionAngry: true,
}

// ReactionEmoji map loại cảm xúc sang emoji Unicode hiển thị (dùng cả ở
// backend nếu cần build response mẫu, chủ yếu FE tự map lại theo constant này).
var ReactionEmoji = map[string]string{
	ReactionLike:  "👍",
	ReactionLove:  "❤️",
	ReactionHaha:  "😂",
	ReactionWow:   "😮",
	ReactionSad:   "😢",
	ReactionAngry: "😡",
}

package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Artwork là bảng trung tâm của hội thi vẽ tranh - 1 tác phẩm học sinh
// đã upload lên S3 và được admin gắn đầy đủ metadata.
type Artwork struct {
	ID           int64  `db:"id" json:"id"`
	Title        string `db:"title" json:"title"`
	StudentID    int64  `db:"student_id" json:"student_id"`
	SchoolID     int64  `db:"school_id" json:"school_id"`
	GradeLevelID int64  `db:"grade_level_id" json:"grade_level_id"`
	// TopicCategoryID là nhóm chủ đề sáng tạo tác phẩm thuộc về (vd "Trí
	// tưởng tượng & thế giới thần tiên"). nil với tác phẩm cũ trước khi có
	// tính năng này - không bắt buộc, trang public đã có đường lui.
	TopicCategoryID *int64  `db:"topic_category_id" json:"topic_category_id,omitempty"`
	S3Key           string  `db:"s3_key" json:"-"`
	S3URL           string  `db:"s3_url" json:"image_url"`
	ThumbnailURL    *string `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	// Variants map các cỡ ảnh dẫn xuất -> URL, khoá dạng "thumb_webp",
	// "medium_jpg"... (xem internal/service/image_variants.go). Frontend dùng
	// để dựng <picture>/srcset thay vì luôn tải ảnh gốc.
	//
	// Có thể rỗng: tác phẩm upload trước khi có tính năng này, hoặc ảnh gốc
	// vốn đã nhỏ hơn mọi cỡ đích. Frontend phải luôn có đường lui về
	// thumbnail_url rồi tới image_url.
	Variants    ArtworkVariants `db:"variants" json:"variants,omitempty"`
	FileSize    int64           `db:"file_size" json:"file_size"`
	Width       *int            `db:"width" json:"width,omitempty"`
	Height      *int            `db:"height" json:"height,omitempty"`
	IsFeatured  bool            `db:"is_featured" json:"is_featured"`
	IsPublished bool            `db:"is_published" json:"is_published"`
	ViewCount   int64           `db:"view_count" json:"view_count"`
	UploadID    *int64          `db:"upload_id" json:"-"`
	CreatedBy   *int64          `db:"created_by" json:"-"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// ArtworkFilter gom các điều kiện lọc/tìm kiếm/phân trang dùng chung cho
// cả trang quản trị lẫn API public (public sẽ luôn ép IsPublished=true).
type ArtworkFilter struct {
	Search          string // khớp title hoặc tên học sinh (LIKE, có thể đổi FULLTEXT sau)
	SchoolID        *int64
	Region          *string // saigon | cantho | vungtau — lọc qua schools.region
	GradeLevelID    *int64
	EducationLevel  string // "primary" | "secondary" | "" (không lọc)
	TopicCategoryID *int64
	AwardID         *int64
	// HasAward lọc "mọi tác phẩm có ít nhất một giải" (không quan tâm giải
	// nào). Khác AwardID ở chỗ chỉ cần một truy vấn để dựng bảng vinh danh,
	// thay vì lặp từng giải rồi truy vấn lại cho mỗi giải.
	HasAward    *bool
	IsFeatured  *bool
	IsPublished *bool
	Page        int
	PageSize    int
}

// ArtworkVariants map "<cỡ>_<định dạng>" -> URL, ví dụ:
//
//	{"thumb_webp": "https://...thumb.webp", "thumb_jpg": "https://...thumb.jpg"}
//
// Lưu ở cột JSON `artworks.variants` (migration 013).
type ArtworkVariants map[string]string

// Scan đọc cột JSON từ MySQL. NULL (tác phẩm cũ chưa sinh biến thể) trả về
// map nil chứ không phải lỗi - phía hiển thị đã có đường lui về ảnh gốc.
func (v *ArtworkVariants) Scan(src any) error {
	if src == nil {
		*v = nil
		return nil
	}

	var raw []byte
	switch s := src.(type) {
	case []byte:
		raw = s
	case string:
		raw = []byte(s)
	default:
		return fmt.Errorf("không đọc được artworks.variants từ kiểu %T", src)
	}

	if len(raw) == 0 || string(raw) == "null" {
		*v = nil
		return nil
	}

	m := make(ArtworkVariants)
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("artworks.variants không phải JSON hợp lệ: %w", err)
	}
	*v = m
	return nil
}

// Value ghi xuống cột JSON. Map rỗng lưu thành NULL để phân biệt rõ "chưa
// sinh biến thể" với "đã sinh nhưng rỗng".
func (v ArtworkVariants) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(map[string]string(v))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// ArtworkAward biểu diễn 1 tác phẩm kèm giải thưởng đã gắn - dùng khi trả
// list/detail cần hiển thị badge giải ngay trên card.
type ArtworkWithMeta struct {
	Artwork
	StudentName       string           `json:"student_name"`
	SchoolName        string           `json:"school_name"`
	Region            string           `json:"region"`
	GradeLabel        string           `json:"grade_label"`
	EducationLevel    string           `json:"education_level"`
	TopicCategoryName string           `json:"topic_category_name,omitempty"`
	ClassName         *string          `json:"class_name,omitempty"`
	CommentCount      int64            `json:"comment_count"`
	ReactionCounts    map[string]int64 `json:"reaction_counts"`
	Awards            []Award          `json:"awards,omitempty"`
}

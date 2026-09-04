package models

import "time"

// Artwork là bảng trung tâm của hội thi vẽ tranh - 1 tác phẩm học sinh
// đã upload lên S3 và được admin gắn đầy đủ metadata.
type Artwork struct {
	ID           int64     `db:"id" json:"id"`
	Title        string    `db:"title" json:"title"`
	StudentID    int64     `db:"student_id" json:"student_id"`
	SchoolID     int64     `db:"school_id" json:"school_id"`
	GradeLevelID int64     `db:"grade_level_id" json:"grade_level_id"`
	S3Key        string    `db:"s3_key" json:"-"`
	S3URL        string    `db:"s3_url" json:"image_url"`
	ThumbnailURL *string   `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	FileSize     int64     `db:"file_size" json:"file_size"`
	Width        *int      `db:"width" json:"width,omitempty"`
	Height       *int      `db:"height" json:"height,omitempty"`
	IsFeatured   bool      `db:"is_featured" json:"is_featured"`
	IsPublished  bool      `db:"is_published" json:"is_published"`
	ViewCount    int64     `db:"view_count" json:"view_count"`
	UploadID     *int64    `db:"upload_id" json:"-"`
	CreatedBy    *int64    `db:"created_by" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// ArtworkFilter gom các điều kiện lọc/tìm kiếm/phân trang dùng chung cho
// cả trang quản trị lẫn API public (public sẽ luôn ép IsPublished=true).
type ArtworkFilter struct {
	Search         string // khớp title hoặc tên học sinh (LIKE, có thể đổi FULLTEXT sau)
	SchoolID       *int64
	GradeLevelID   *int64
	EducationLevel string // "primary" | "secondary" | "" (không lọc)
	AwardID        *int64
	IsFeatured     *bool
	IsPublished    *bool
	Page           int
	PageSize       int
}

// ArtworkAward biểu diễn 1 tác phẩm kèm giải thưởng đã gắn - dùng khi trả
// list/detail cần hiển thị badge giải ngay trên card.
type ArtworkWithMeta struct {
	Artwork
	StudentName    string           `json:"student_name"`
	SchoolName     string           `json:"school_name"`
	Region         string           `json:"region"`
	GradeLabel     string           `json:"grade_label"`
	EducationLevel string           `json:"education_level"`
	ClassName      *string          `json:"class_name,omitempty"`
	CommentCount   int64            `json:"comment_count"`
	ReactionCounts map[string]int64 `json:"reaction_counts"`
	Awards         []Award          `json:"awards,omitempty"`
}

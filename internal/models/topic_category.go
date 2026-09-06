package models

import "time"

// TopicCategory là 1 nhóm chủ đề sáng tạo trong thể lệ hội thi (vd "Trí
// tưởng tượng & thế giới thần tiên", "Niềm vui, tình bạn, cảm xúc học
// đường..."). EducationLevel lọc nhóm nào áp dụng cho cấp học nào - Tiểu học
// và THCS-THPT có bộ nhóm chủ đề khác nhau; nil = dùng chung mọi cấp.
type TopicCategory struct {
	ID             int64     `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	Slug           string    `db:"slug" json:"slug"`
	EducationLevel *string   `db:"education_level" json:"education_level,omitempty"`
	DisplayOrder   int       `db:"display_order" json:"display_order"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

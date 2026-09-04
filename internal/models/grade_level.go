package models

import "time"

// GradeLevel đại diện cho 1 khối lớp, phân theo 2 cấp học: Tiểu học (1-5)
// và Trung học - THCS & THPT gộp chung (6-12).
type GradeLevel struct {
	ID             int64     `db:"id" json:"id"`
	EducationLevel string    `db:"education_level" json:"education_level"`
	GradeNumber    int       `db:"grade_number" json:"grade_number"`
	Label          string    `db:"label" json:"label"`
	DisplayOrder   int       `db:"display_order" json:"display_order"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// EducationLevel: 2 cấp học dùng để phân loại khối lớp.
const (
	EducationLevelPrimary   = "primary"   // Tiểu học - khối 1-5
	EducationLevelSecondary = "secondary" // Trung học (THCS & THPT) - khối 6-12
)

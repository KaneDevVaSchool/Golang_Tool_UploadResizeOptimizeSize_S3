package models

import "time"

// Student đại diện cho học sinh sáng tác tác phẩm. Không có hệ thống tài
// khoản học sinh đầy đủ - mỗi lần nộp tác phẩm mới sẽ tạo 1 record mới
// (không dedupe theo tên vì không có mã định danh học sinh chính thức).
type Student struct {
	ID           int64     `db:"id" json:"id"`
	FullName     string    `db:"full_name" json:"full_name"`
	SchoolID     int64     `db:"school_id" json:"school_id"`
	GradeLevelID int64     `db:"grade_level_id" json:"grade_level_id"`
	ClassName    *string   `db:"class_name" json:"class_name,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

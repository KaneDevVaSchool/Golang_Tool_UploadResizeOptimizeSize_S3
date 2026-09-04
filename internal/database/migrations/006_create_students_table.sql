-- Students: học sinh sáng tác tác phẩm. Không có hệ thống tài khoản học sinh
-- đầy đủ - chỉ lưu tên + trường + khối lớp tại thời điểm nộp tác phẩm.
CREATE TABLE IF NOT EXISTS students (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    school_id BIGINT UNSIGNED NOT NULL,
    grade_level_id BIGINT UNSIGNED NOT NULL,
    class_name VARCHAR(100) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_students_school
        FOREIGN KEY (school_id) REFERENCES schools (id),
    CONSTRAINT fk_students_grade_level
        FOREIGN KEY (grade_level_id) REFERENCES grade_levels (id),
    INDEX idx_students_school_id (school_id),
    INDEX idx_students_grade_level_id (grade_level_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

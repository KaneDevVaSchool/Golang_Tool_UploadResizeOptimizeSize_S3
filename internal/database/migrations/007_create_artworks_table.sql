-- Artworks: bảng trung tâm của hội thi vẽ tranh. school_id/grade_level_id
-- denormalize từ students để query dashboard (đếm theo khu vực/khối) nhanh,
-- không phải JOIN qua students mỗi lần thống kê.
CREATE TABLE IF NOT EXISTS artworks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    school_id BIGINT UNSIGNED NOT NULL,
    grade_level_id BIGINT UNSIGNED NOT NULL,
    s3_key VARCHAR(500) NOT NULL,
    s3_url VARCHAR(1000) NOT NULL,
    thumbnail_url VARCHAR(1000) NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    width INT NULL,
    height INT NULL,
    is_featured TINYINT(1) NOT NULL DEFAULT 0,
    is_published TINYINT(1) NOT NULL DEFAULT 1,
    view_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    upload_id BIGINT UNSIGNED NULL,
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_artworks_student
        FOREIGN KEY (student_id) REFERENCES students (id),
    CONSTRAINT fk_artworks_school
        FOREIGN KEY (school_id) REFERENCES schools (id),
    CONSTRAINT fk_artworks_grade_level
        FOREIGN KEY (grade_level_id) REFERENCES grade_levels (id),
    CONSTRAINT fk_artworks_upload
        FOREIGN KEY (upload_id) REFERENCES uploads (id) ON DELETE SET NULL,
    CONSTRAINT fk_artworks_created_by
        FOREIGN KEY (created_by) REFERENCES admin_users (id) ON DELETE SET NULL,
    INDEX idx_artworks_school_id (school_id),
    INDEX idx_artworks_grade_level_id (grade_level_id),
    INDEX idx_artworks_is_featured (is_featured),
    INDEX idx_artworks_is_published (is_published),
    FULLTEXT INDEX idx_artworks_title (title)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

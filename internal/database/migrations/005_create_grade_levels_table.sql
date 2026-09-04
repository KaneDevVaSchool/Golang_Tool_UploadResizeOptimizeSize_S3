-- Grade levels: phân loại 2 cấp theo yêu cầu -
-- Tiểu học (primary) khối 1-5, Trung học (secondary, gồm THCS & THPT) khối 6-12.
CREATE TABLE IF NOT EXISTS grade_levels (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    education_level ENUM('primary', 'secondary') NOT NULL,
    grade_number TINYINT UNSIGNED NOT NULL,
    label VARCHAR(50) NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_grade_levels_level_number (education_level, grade_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO grade_levels (education_level, grade_number, label, display_order) VALUES
    ('primary', 1, 'Khối 1', 1),
    ('primary', 2, 'Khối 2', 2),
    ('primary', 3, 'Khối 3', 3),
    ('primary', 4, 'Khối 4', 4),
    ('primary', 5, 'Khối 5', 5),
    ('secondary', 6, 'Khối 6', 6),
    ('secondary', 7, 'Khối 7', 7),
    ('secondary', 8, 'Khối 8', 8),
    ('secondary', 9, 'Khối 9', 9),
    ('secondary', 10, 'Khối 10', 10),
    ('secondary', 11, 'Khối 11', 11),
    ('secondary', 12, 'Khối 12', 12);

-- Schools: 5 cơ sở vật lý của Hệ thống Trường Việt Mỹ, gộp về 3 khu vực
-- trưng bày (region) theo yêu cầu quản trị: Sài Gòn / Cần Thơ / Vũng Tàu.
CREATE TABLE IF NOT EXISTS schools (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    region ENUM('saigon', 'cantho', 'vungtau') NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_schools_region (region)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO schools (name, region, display_order) VALUES
    ('Bình Thới - Tân Bình', 'saigon', 1),
    ('Thống Tây Hội', 'saigon', 2),
    ('Phú Định', 'saigon', 3),
    ('Vũng Tàu', 'vungtau', 4),
    ('Cần Thơ', 'cantho', 5);

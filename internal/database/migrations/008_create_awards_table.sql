-- Awards: cấu hình các loại giải thưởng (Nhất/Nhì/Ba/Khuyến khích...),
-- màu + icon hiển thị tự cấu hình được ở trang quản lý giải thưởng.
CREATE TABLE IF NOT EXISTS awards (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    rank_order INT NOT NULL DEFAULT 0,
    color_hex VARCHAR(7) NOT NULL DEFAULT '#c49c57',
    icon_key VARCHAR(50) NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_awards_slug (slug),
    INDEX idx_awards_rank_order (rank_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

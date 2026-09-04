-- Artwork comments: ẩn danh, người xem tự nhập tên hiển thị (display_name)
-- khi bình luận. is_hidden cho phép admin ẩn spam mà không xóa hẳn (moderation nhẹ).
CREATE TABLE IF NOT EXISTS artwork_comments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    artwork_id BIGINT UNSIGNED NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    content VARCHAR(1000) NOT NULL,
    visitor_token VARCHAR(64) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    is_hidden TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_artwork_comments_artwork
        FOREIGN KEY (artwork_id) REFERENCES artworks (id) ON DELETE CASCADE,
    INDEX idx_comments_artwork_id (artwork_id),
    INDEX idx_comments_created_at (created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

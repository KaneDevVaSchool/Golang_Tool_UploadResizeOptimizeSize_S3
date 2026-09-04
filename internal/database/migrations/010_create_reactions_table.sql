-- Artwork reactions: ẩn danh, định danh qua visitor_token (UUID sinh ở
-- trình duyệt, lưu localStorage) - KHÔNG phải xác thực danh tính thật.
-- ip_address chỉ dùng cho rate-limit/audit nội bộ, không hiển thị public.
CREATE TABLE IF NOT EXISTS artwork_reactions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    artwork_id BIGINT UNSIGNED NOT NULL,
    reaction_type ENUM('like', 'love', 'haha', 'wow', 'sad', 'angry') NOT NULL,
    visitor_token VARCHAR(64) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_artwork_reactions_artwork
        FOREIGN KEY (artwork_id) REFERENCES artworks (id) ON DELETE CASCADE,
    UNIQUE KEY uq_artwork_reactions_visitor (artwork_id, visitor_token, reaction_type),
    INDEX idx_reactions_artwork_id (artwork_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

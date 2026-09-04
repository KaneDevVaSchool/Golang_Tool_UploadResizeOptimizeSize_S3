-- Artwork views: chống đếm trùng view trong khoảng thời gian ngắn.
-- Trước khi tăng artworks.view_count, service kiểm tra đã có bản ghi
-- (artwork_id, visitor_token) trong 24h gần nhất chưa qua idx_views_artwork_visitor_date.
CREATE TABLE IF NOT EXISTS artwork_views (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    artwork_id BIGINT UNSIGNED NOT NULL,
    visitor_token VARCHAR(64) NOT NULL,
    viewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_artwork_views_artwork
        FOREIGN KEY (artwork_id) REFERENCES artworks (id) ON DELETE CASCADE,
    INDEX idx_views_artwork_visitor_date (artwork_id, visitor_token, viewed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

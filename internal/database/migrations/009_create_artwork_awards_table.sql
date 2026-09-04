-- Artwork <-> Award: quan hệ N-N. UI hiện tại chỉ cần 1 giải/tác phẩm nhưng
-- thiết kế N-N để mở rộng (1 tác phẩm có thể nhận nhiều giải qua các năm/hạng mục).
CREATE TABLE IF NOT EXISTS artwork_awards (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    artwork_id BIGINT UNSIGNED NOT NULL,
    award_id BIGINT UNSIGNED NOT NULL,
    awarded_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_artwork_awards_artwork
        FOREIGN KEY (artwork_id) REFERENCES artworks (id) ON DELETE CASCADE,
    CONSTRAINT fk_artwork_awards_award
        FOREIGN KEY (award_id) REFERENCES awards (id) ON DELETE CASCADE,
    UNIQUE KEY uq_artwork_awards_pair (artwork_id, award_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

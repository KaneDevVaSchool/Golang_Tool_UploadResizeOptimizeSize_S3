-- Nhóm chủ đề sáng tạo trong thể lệ hội thi (vd "Trí tưởng tượng & thế giới
-- thần tiên", "Mái trường Việt Mỹ - Nơi những điều đẹp đẽ được lắng nghe"...).
-- Tách bảng danh mục thay vì lưu text tự do trên artworks để lọc/thống kê
-- được theo nhóm và tránh admin gõ sai chính tả - cùng mẫu với awards/schools.
CREATE TABLE IF NOT EXISTS topic_categories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    -- Mỗi cuộc thi (Tiểu học / THCS-THPT) có bộ nhóm chủ đề riêng. NULL = áp
    -- dụng chung cho mọi cấp học (hiếm khi cần, nhưng để ngỏ như awards.grade_level_id).
    education_level ENUM('primary', 'secondary') NULL,
    display_order INT NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_topic_categories_slug (slug),
    INDEX idx_topic_categories_education_level (education_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- artworks.topic_category_id: RESTRICT mặc định (không cascade) - cùng chính
-- sách với school_id/grade_level_id, chặn xoá nhầm 1 nhóm chủ đề đang có tác
-- phẩm. NULL được vì tác phẩm cũ (trước tính năng này) không có nhóm chủ đề.
--
-- Không dùng "ADD COLUMN IF NOT EXISTS": bản MySQL 8.1 dùng ở dev (ServBay)
-- báo lỗi cú pháp 1064 với cú pháp này (xem giải thích ở
-- 015_add_award_grade_level.sql). Idempotency dựa vào schema_migrations.
ALTER TABLE artworks
    ADD COLUMN topic_category_id BIGINT UNSIGNED NULL AFTER grade_level_id;

ALTER TABLE artworks
    ADD CONSTRAINT fk_artworks_topic_category
        FOREIGN KEY (topic_category_id) REFERENCES topic_categories (id);

CREATE INDEX idx_artworks_topic_category ON artworks (topic_category_id);

-- Artwork downloads: nhật ký ai tải file gốc, lúc nào, từ đâu.
-- Trước đây tải ảnh gốc (cả khu quản trị lẫn trang public) không để lại dấu
-- vết gì - nếu ảnh bị phát tán sai mục đích thì không truy được nguồn. Bảng
-- này chỉ ghi log, không có ràng buộc nghiệp vụ nào khác nên ghi thẳng qua
-- ArtworkService, không tách repository riêng phức tạp hơn artwork_views.
CREATE TABLE IF NOT EXISTS artwork_downloads (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    artwork_id BIGINT UNSIGNED NOT NULL,
    -- NULL khi tải từ trang public (không đăng nhập, không định danh được
    -- người tải) - chỉ có giá trị khi tải từ khu quản trị.
    admin_user_id BIGINT UNSIGNED NULL,
    -- 'admin' hoặc 'public' - phân biệt hai nguồn tải vì chỉ 'admin' có
    -- admin_user_id, còn 'public' phải dựa vào ip_address để tra khi cần.
    source VARCHAR(16) NOT NULL,
    ip_address VARCHAR(64) NOT NULL,
    user_agent VARCHAR(255) NOT NULL DEFAULT '',
    downloaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_artwork_downloads_artwork
        FOREIGN KEY (artwork_id) REFERENCES artworks (id) ON DELETE CASCADE,
    CONSTRAINT fk_artwork_downloads_admin_user
        FOREIGN KEY (admin_user_id) REFERENCES admin_users (id) ON DELETE SET NULL,
    INDEX idx_artwork_downloads_artwork (artwork_id, downloaded_at),
    INDEX idx_artwork_downloads_admin_user (admin_user_id, downloaded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Admin sessions: session lưu DB (không in-memory) để sống sót qua restart server.
-- id là chính token ngẫu nhiên, giá trị này được đặt trong HttpOnly cookie phía client.
CREATE TABLE IF NOT EXISTS admin_sessions (
    id VARCHAR(64) PRIMARY KEY,
    admin_user_id BIGINT UNSIGNED NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_admin_sessions_admin_user
        FOREIGN KEY (admin_user_id) REFERENCES admin_users (id) ON DELETE CASCADE,
    INDEX idx_admin_sessions_expires_at (expires_at),
    INDEX idx_admin_sessions_admin_user_id (admin_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

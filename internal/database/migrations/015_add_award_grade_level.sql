-- Gắn giải thưởng với 1 khối lớp cụ thể - hội thi 2026 chia giải riêng theo
-- từng khối (vd Tiểu học: mỗi khối 1-5 có 1 Nhất/1 Nhì/2 Ba, tổng 20 giải).
-- NULL giữ nguyên ý nghĩa cũ: giải dùng chung toàn hệ thống, không tách khối
-- (vd "Đặc biệt" chọn 20-24 tác phẩm tiêu biểu toàn hệ thống).
--
-- Không dùng "ADD COLUMN IF NOT EXISTS" / "ADD CONSTRAINT ... FOREIGN KEY
-- IF NOT EXISTS": bản MySQL 8.1 đang dùng ở dev (ServBay) báo lỗi cú pháp
-- 1064 với "ADD COLUMN IF NOT EXISTS" dù tài liệu MySQL nói hỗ trợ từ
-- 8.0.29 - không rõ do build ServBay hay khác biệt phiên bản, nhưng đo
-- được là lỗi thật (xem docs/plan/03-risks.md). Idempotency ở đây dựa vào
-- schema_migrations chặn chạy lại file đã áp dụng (xem cơ chế migration,
-- docs/detail_design/01-database.md), giống cách 014_add_perf_indexes.sql
-- dùng CREATE INDEX trần không kèm IF NOT EXISTS.
ALTER TABLE awards
    ADD COLUMN grade_level_id BIGINT UNSIGNED NULL AFTER slug;

ALTER TABLE awards
    ADD CONSTRAINT fk_awards_grade_level
        FOREIGN KEY (grade_level_id) REFERENCES grade_levels (id);

CREATE INDEX idx_awards_grade_level ON awards (grade_level_id);

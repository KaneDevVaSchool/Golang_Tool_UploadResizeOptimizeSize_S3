-- Thêm màu hiển thị cho nhóm chủ đề - cùng cột color_hex như awards, dùng để
-- tô icon nhóm chủ đề trong dropdown ArtworkMetaForm và trang quản lý, thay
-- vì mã cứng 1 màu nâu (#725139) cho mọi nhóm.
--
-- Không dùng "ADD COLUMN IF NOT EXISTS": bản MySQL 8.1 dùng ở dev (ServBay)
-- báo lỗi cú pháp 1064 với cú pháp này (xem giải thích ở
-- 015_add_award_grade_level.sql). Idempotency dựa vào schema_migrations.
ALTER TABLE topic_categories
    ADD COLUMN color_hex VARCHAR(7) NOT NULL DEFAULT '#725139' AFTER slug;

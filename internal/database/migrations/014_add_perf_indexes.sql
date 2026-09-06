-- Index phục vụ hai truy vấn nóng nhất của trang công khai.
--
-- 1) Danh sách tác phẩm (artwork_repository.go -> List): mọi truy vấn công
--    khai đều lọc is_published = 1 rồi ORDER BY created_at DESC, nhưng
--    created_at chưa hề có index nào. MySQL vì thế chọn idx_artworks_is_published
--    (gần như toàn bảng đều = 1, chọn lọc gần bằng không) rồi filesort lại
--    toàn bộ kết quả cho mỗi trang.
--
--    Composite (is_published, created_at DESC) vừa lọc vừa cho sẵn thứ tự,
--    nên MySQL đọc đúng số dòng của trang rồi dừng.
CREATE INDEX idx_artworks_published_created
    ON artworks (is_published, created_at DESC);

-- 2) Bình luận theo tác phẩm (comment_repository.go -> ListByArtwork):
--    WHERE artwork_id = ? AND is_hidden = 0 ORDER BY created_at DESC.
--    Bảng đang có idx_comments_artwork_id và idx_comments_created_at riêng
--    rẽ, nên MySQL chỉ dùng được một trong hai rồi filesort phần còn lại.
CREATE INDEX idx_comments_artwork_visible_created
    ON artwork_comments (artwork_id, is_hidden, created_at DESC);

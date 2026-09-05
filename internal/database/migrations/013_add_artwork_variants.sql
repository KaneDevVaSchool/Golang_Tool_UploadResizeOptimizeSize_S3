-- Biến thể ảnh (thumb/medium/large × webp/jpg) sinh lúc upload.
--
-- Trước migration này, lưới gallery tải thẳng ảnh gốc cho từng ô: cột
-- thumbnail_url có sẵn trong schema nhưng KHÔNG code path nào từng ghi vào
-- nó, nên với ảnh dự thi cỡ vài MB thì một trang 36 tranh phải kéo hàng chục
-- MB. Nay mỗi tác phẩm có sẵn các cỡ nhỏ để frontend chọn qua <picture>/srcset.
--
-- Dùng JSON thay vì thêm 6 cột riêng: số biến thể của mỗi ảnh không cố định
-- (ảnh gốc nhỏ hơn cỡ đích thì không sinh biến thể đó - xem skipVariant trong
-- internal/service/image_variants.go), và bộ cỡ có thể đổi về sau mà không
-- phải migrate lại schema.
--
-- Dạng dữ liệu: {"thumb_webp":"https://...","thumb_jpg":"https://...", ...}
-- thumbnail_url vẫn được ghi (trỏ thumb_jpg) để code cũ và trang admin không gãy.
ALTER TABLE artworks
    ADD COLUMN variants JSON NULL AFTER thumbnail_url;

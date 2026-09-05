# Kế hoạch — Tổng quan

Trạng thái dự án và lộ trình, đối chiếu trực tiếp với code tại commit `d0ec7c2`
(nhánh `feature/artwork-contest-system`), khảo sát ngày **2026-09-06**.

## Danh mục

| Tài liệu | Nội dung |
|---|---|
| [01-current-state.md](./01-current-state.md) | Đã xong / đang dở / nợ kỹ thuật, có dẫn chiếu code |
| [02-roadmap.md](./02-roadmap.md) | Việc tiếp theo, xếp theo ưu tiên, kèm tiêu chí hoàn thành |
| [03-risks.md](./03-risks.md) | Rủi ro đã xác định và biện pháp giảm thiểu |

## Tóm tắt trạng thái

Hệ thống **đã chạy được đầy đủ ba khu vực**: trang public, admin panel, công cụ upload.
Toàn bộ luồng nghiệp vụ chính từ upload ảnh đến hiển thị công khai đều hoạt động.

Kiểm chứng ngày 2026-09-06:

| Kiểm tra | Kết quả |
|---|---|
| `go build ./...` | ✅ Biên dịch sạch |
| `go test ./...` | ✅ Toàn bộ gói đạt |

Tính năng sinh biến thể ảnh WebP — từng là điểm dở dang duy nhất — **đã hoàn thành**: đổi
bộ mã hoá sang `gen2brain/webp` (lossy VP8, không cần cgo), WebP nay nhỏ hơn JPEG 76–82%.
Chi tiết và số đo trong [01-current-state.md](./01-current-state.md).

## Bối cảnh

Dự án phục vụ hội thi vẽ tranh kỷ niệm **20 năm Hệ thống Trường Việt Mỹ**. Đặc điểm chi
phối nhiều quyết định kỹ thuật:

- **Có thời hạn cứng** — gắn với sự kiện kỷ niệm, không lùi được.
- **Quy mô xác định** — 5 cơ sở, 12 khối lớp, ước tính vài nghìn tác phẩm.
- **Đỉnh truy cập ngắn** — đông người xem vào lúc công bố kết quả, sau đó giảm dần.
- **Người vận hành không chuyên** — ban tổ chức là giáo viên, không phải kỹ sư.

Vì thế lộ trình ưu tiên **độ tin cậy và dễ vận hành** hơn là tính năng mới hay tối ưu sớm.

## Nguyên tắc khi lập kế hoạch tiếp

1. **Sửa cái đã biết hỏng trước khi thêm cái mới.** Các mục ⚠️ trong tài liệu thiết kế
   chi tiết là hàng đợi ưu tiên.
2. **Đừng tối ưu khi chưa đo.** Nhiều lựa chọn hiện tại (`LIKE` thay FULLTEXT, nạp toàn bộ
   bảng danh mục) là đúng ở quy mô này.
3. **Mỗi thay đổi phải vận hành được bởi người không chuyên.** Thêm phụ thuộc là thêm việc
   cho người trực hệ thống.

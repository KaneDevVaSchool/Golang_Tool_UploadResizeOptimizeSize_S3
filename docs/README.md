# Tài liệu hệ thống — VASchools Art Gallery

Bộ tài liệu cho hệ thống hội thi vẽ tranh "20 năm Trường Việt Mỹ": upload ảnh lên S3,
quản trị tác phẩm/giải thưởng, và trang public trưng bày.

Toàn bộ tài liệu trong thư mục này được viết lại ngày **2026-09-06**, đối chiếu trực tiếp
với source code tại commit `d0ec7c2` (nhánh `feature/artwork-contest-system`). Mọi con số,
tên biến, tên bảng, tên endpoint đều lấy từ code — không phải từ tài liệu cũ.

## Bắt đầu từ đâu

| Bạn là | Đọc theo thứ tự |
|---|---|
| Người mới vào dự án | [ARCHITECTURE](./ARCHITECTURE.md) → [detail_design/](./detail_design/README.md) |
| Cần gọi API | [API](./API.md) |
| Deploy lần đầu | [deploys/00-tu-dau-den-cuoi](./deploys/00-tu-dau-den-cuoi.md) |
| Deploy lại / vận hành | [deploys/](./deploys/README.md) |
| Cần biết làm gì tiếp | [plan/](./plan/README.md) |
| Cần sửa 1 package cụ thể | [MODULES](./MODULES.md) |

## Bản đồ tài liệu

### Tài liệu nền

| File | Nội dung |
|---|---|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Kiến trúc tổng thể, layer, luồng request, middleware chain, mô hình dữ liệu |
| [MODULES.md](./MODULES.md) | Bóc tách từng package Go: trách nhiệm, phụ thuộc, lưu ý khi sửa |
| [API.md](./API.md) | Tham chiếu đầy đủ từng endpoint: request/response/mã lỗi |

### `detail_design/` — Thiết kế chi tiết

Đặc tả mức thiết kế cho từng miền chức năng: dữ liệu vào/ra, quy tắc nghiệp vụ,
ràng buộc, và lý do đằng sau mỗi quyết định.

| File | Phạm vi |
|---|---|
| [00-overview.md](./detail_design/README.md) | Mục lục, quy ước ký hiệu, ranh giới miền |
| [01-database.md](./detail_design/01-database.md) | 13 bảng MySQL: cột, index, khoá ngoại, lý do denormalize |
| [02-upload-pipeline.md](./detail_design/02-upload-pipeline.md) | Upload đơn, chunked, transaction; validate; S3 key |
| [03-artwork-domain.md](./detail_design/03-artwork-domain.md) | Vòng đời tác phẩm, bulk upload 2 bước, enrich, gán giải |
| [04-public-engagement.md](./detail_design/04-public-engagement.md) | Reaction/comment/view ẩn danh, visitor_token, chống trùng |
| [05-auth-security.md](./detail_design/05-auth-security.md) | Google OAuth, session, CSRF, rate limit, phân tầng bảo vệ |
| [06-frontend.md](./detail_design/06-frontend.md) | Router, 3 khu vực UI, state, API client, hiệu ứng |

### `deploys/` — Triển khai vận hành

| File | Phạm vi |
|---|---|
| [README.md](./deploys/README.md) | Chọn phương án, checklist tổng |
| [00-tu-dau-den-cuoi.md](./deploys/00-tu-dau-den-cuoi.md) | **Deploy lần đầu**: hướng dẫn cực chi tiết từ lúc chưa mua VPS đến khi nghiệm thu xong |
| [01-vps-systemd.md](./deploys/01-vps-systemd.md) | Quy trình rút gọn: binary + systemd + Nginx + Certbot |
| [02-configuration.md](./deploys/02-configuration.md) | Toàn bộ biến môi trường: ý nghĩa, mặc định, ràng buộc |
| [03-operations.md](./deploys/03-operations.md) | Vận hành: log, backup, sự cố thường gặp, rollback |
| [S3-PUBLIC-READ.md](./S3-PUBLIC-READ.md) | Cấu hình bucket policy cho ảnh public |

### `plan/` — Kế hoạch

| File | Phạm vi |
|---|---|
| [README.md](./plan/README.md) | Trạng thái hiện tại, lộ trình |
| [01-current-state.md](./plan/01-current-state.md) | Đã xong / đang dở / nợ kỹ thuật, đối chiếu code |
| [02-roadmap.md](./plan/02-roadmap.md) | Các giai đoạn tiếp theo, ưu tiên, tiêu chí hoàn thành |
| [03-risks.md](./plan/03-risks.md) | Rủi ro đã xác định + biện pháp giảm thiểu |

## Quy ước chung trong toàn bộ tài liệu

- **Đường dẫn code** viết dạng `internal/service/artwork_service.go:180` để mở thẳng được.
- **Trạng thái** đánh dấu: ✅ đã có trong code · 🚧 làm dở · 📋 chưa làm · ⚠️ có rủi ro.
- Khi tài liệu và code mâu thuẫn, **code là đúng** — hãy sửa tài liệu và ghi lại lý do.

## Điều đã sửa so với bộ tài liệu cũ

Bản khảo sát ngày 2026-09-06 phát hiện các sai lệch sau trong tài liệu cũ, đã sửa hết:

| Sai lệch | Thực tế trong code |
|---|---|
| `ARCHITECTURE.md` ghi CSDL là PostgreSQL | MySQL 8 — driver `go-sql-driver/mysql`, DSN `user:pass@tcp(...)`, schema dùng `ENGINE=InnoDB` |
| `DEPLOYMENT.md` hướng dẫn deploy bằng Dockerfile | `Dockerfile` đã bị xoá khỏi repo; phương án thật là binary + systemd |
| `MODULES.md` ghi "hiện có 1 file migration" | Nhiều file migration tăng dần, đánh số từ `001` (xem `internal/database/migrations/` để biết số lượng hiện tại — đừng chép cứng con số vào tài liệu) |
| Tài liệu cũ không nhắc trang public | Trang public là phần lớn nhất của UI hiện tại (4 trang, 22 component) |


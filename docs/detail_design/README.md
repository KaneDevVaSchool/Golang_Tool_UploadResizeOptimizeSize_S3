# Thiết kế chi tiết — Mục lục

Thư mục này đặc tả **mức thiết kế** cho từng miền chức năng: dữ liệu vào/ra, quy tắc nghiệp
vụ, ràng buộc, trạng thái, và lý do đằng sau mỗi quyết định.

Khác biệt với các tài liệu khác:

- [ARCHITECTURE.md](../ARCHITECTURE.md) trả lời *hệ thống được lắp ráp thế nào*.
- [MODULES.md](../MODULES.md) trả lời *package nào làm gì*.
- **Thư mục này** trả lời *nghiệp vụ chạy ra sao và tại sao lại thiết kế như vậy*.
- [API.md](../API.md) trả lời *gọi endpoint nào, gửi gì, nhận gì*.

## Danh mục

| # | Tài liệu | Nội dung chính |
|---|---|---|
| 01 | [Cơ sở dữ liệu](./01-database.md) | 14 bảng: cột, kiểu, index, khoá ngoại, dữ liệu seed, lý do denormalize |
| 02 | [Pipeline upload](./02-upload-pipeline.md) | Upload đơn, bulk upload, validate nhiều lớp, sinh S3 key, sinh biến thể, tải ảnh có watermark |
| 03 | [Miền tác phẩm](./03-artwork-domain.md) | Vòng đời tác phẩm, bulk upload 2 bước, enrich, gán giải, lọc |
| 04 | [Tương tác public](./04-public-engagement.md) | Reaction/comment/view ẩn danh, visitor_token, chia sẻ OG |
| 05 | [Xác thực & bảo mật](./05-auth-security.md) | Google OAuth, session DB, CSRF, rate limit, phân tầng phòng vệ |
| 06 | [Frontend](./06-frontend.md) | Router, 3 khu vực UI, quản lý state, API client, hiệu ứng |

## Quy ước ký hiệu dùng chung

| Ký hiệu | Nghĩa |
|---|---|
| ✅ | Đã có trong code, đã chạy |
| 🚧 | Có một phần, còn thiếu |
| 📋 | Chưa làm, đã lên kế hoạch |
| ⚠️ | Có rủi ro/hạn chế cần biết |
| `file.go:42` | Vị trí code tương ứng |

## Ranh giới giữa các tầng

Ba quy tắc được giữ nhất quán trong toàn bộ code; khi thêm tính năng mới hãy theo đúng:

**Handler không chứa nghiệp vụ.** Handler chỉ: kiểm tra method → parse input → gọi service →
map lỗi thành mã HTTP → trả JSON. Mọi quyết định nghiệp vụ nằm ở service.

**Service sở hữu transaction.** Chỉ service mở/commit/rollback transaction. Repository nhận
`*sql.Tx` như tham số khi cần tham gia transaction của service (xem chữ ký
`StudentRepository.Create(ctx, tx, ...)`), tự nó không quyết định ranh giới transaction.

**Repository không biết HTTP.** Không có `http.Request`, không mã lỗi HTTP, không JSON tag
quyết định hành vi. Repository chỉ nói chuyện với MySQL và AWS SDK.

Ngoại lệ duy nhất đáng chú ý: `PublicHandler` gọi thẳng một số repository
(`reactionRepo`, `commentRepo`, `viewRepo`) thay vì qua service. Đây là **nợ kỹ thuật đã
biết** — các thao tác này hiện chỉ là CRUD một bảng nên chưa cần tầng service, nhưng nếu
thêm luật (lọc từ khoá, kiểm duyệt, thông báo) thì phải tách service trước.
Xem [plan/01-current-state.md](../plan/01-current-state.md).

# VA Pictures — Quy tắc dự án

Hệ thống hội thi vẽ tranh "20 năm Trường Việt Mỹ": upload ảnh lên S3, quản trị tác
phẩm/giải thưởng, trang public trưng bày.

Stack: **Go 1.24 (backend) + React 19 + TypeScript + Vite (frontend) + MySQL 8 + Amazon S3**.
Triển khai: **binary + systemd + Nginx**, không dùng Docker.

Đọc file này trước khi sửa code. Chi tiết từng phần xem `docs/`.

---

## 0. Quy tắc quan trọng nhất — Audit với docs và plan

**Mọi thay đổi code đều phải đối chiếu với tài liệu, trước khi làm và sau khi làm.**

Đây không phải thủ tục hình thức. Bộ tài liệu cũ của dự án từng ghi CSDL là PostgreSQL
trong khi code dùng MySQL, và hướng dẫn deploy bằng Dockerfile đã bị xoá khỏi repo. Tài
liệu sai còn tệ hơn không có tài liệu, vì người đọc tin vào nó.

### Trước khi bắt đầu một thay đổi

1. Đọc `docs/plan/02-roadmap.md` — việc sắp làm đã nằm trong lộ trình chưa, ở mức ưu tiên
   nào? Nếu chưa có, **hỏi trước** thay vì tự thêm.
2. Đọc tài liệu thiết kế của miền liên quan trong `docs/detail_design/` — đã có ràng buộc
   hay quyết định thiết kế nào chi phối chỗ sắp sửa không?
3. Đọc `docs/plan/01-current-state.md` mục "Nợ kỹ thuật" — chỗ sắp sửa có đang nằm trong
   danh sách đã biết không?

### Sau khi thay đổi xong

Chạy skill `docs-audit` (hoặc tự rà theo bảng dưới). Cập nhật **mọi** tài liệu bị ảnh hưởng
trong **cùng commit** với code:

| Sửa gì | Bắt buộc cập nhật |
|---|---|
| Thêm/sửa/xoá endpoint | `docs/API.md` |
| Thêm/sửa bảng, cột, index | `docs/detail_design/01-database.md` |
| Đổi luồng upload | `docs/detail_design/02-upload-pipeline.md` |
| Đổi nghiệp vụ tác phẩm/giải | `docs/detail_design/03-artwork-domain.md` |
| Đổi reaction/comment/view | `docs/detail_design/04-public-engagement.md` |
| Đổi auth, session, CSRF, rate limit | `docs/detail_design/05-auth-security.md` |
| Thêm/xoá trang, route, component lớn | `docs/detail_design/06-frontend.md` |
| Thêm/đổi/xoá biến môi trường | `.env.example` **và** `docs/deploys/02-configuration.md` |
| Đổi chuỗi middleware, thêm package | `docs/ARCHITECTURE.md`, `docs/MODULES.md` |
| Đổi quy trình deploy, script | `docs/deploys/00-tu-dau-den-cuoi.md`, `deploy/README.md` |
| Hoàn thành một mục roadmap | `docs/plan/02-roadmap.md` (đánh dấu ✅) **và** `docs/plan/01-current-state.md` |
| Phát hiện rủi ro mới | `docs/plan/03-risks.md` |

### Nguyên tắc khi tài liệu và code mâu thuẫn

**Code là đúng.** Sửa tài liệu cho khớp code, và ghi lại lý do nếu sự lệch đó có ý nghĩa.
Đừng sửa code cho khớp tài liệu trừ khi tài liệu mô tả một yêu cầu nghiệp vụ thật.

### Con số trong tài liệu

Đừng viết con số mà bạn không đếm được bằng một lệnh. Tài liệu cũ từng ghi "hiện có 1 file
migration" khi thực tế có 13. Nếu phải nêu số lượng, đếm lại tại thời điểm viết — hoặc
diễn đạt không cần số ("toàn bộ biến môi trường" thay vì "45 biến môi trường").

---

## 1. Kiến trúc — Handler → Service → Repository

Bắt buộc với mọi tính năng mới:

```
Handler (mỏng: parse request, gọi service, trả JSON)
    └── Service (nghiệp vụ, điều phối, transaction)
            └── Repository (nơi DUY NHẤT chạy SQL)
                    └── Model
```

- Handler **không** chạy SQL trực tiếp.
- Service **không** chạy SQL trực tiếp.
- Phụ thuộc khai báo qua interface, nối dây trong `internal/container/container.go`.

⚠️ Ngoại lệ đã tồn tại: `PublicHandler` gọi thẳng repository cho reaction/comment/view.
Chấp nhận được vì chỉ là CRUD một bảng, nhưng **đừng nhân rộng** — thêm luật nghiệp vụ nào
vào đó thì phải tách service trước.

## 2. Tiếng Việt trong code và tài liệu

- **Comment, commit message, tài liệu**: tiếng Việt. Người bảo trì là giáo viên và kỹ sư
  trong trường.
- **Tên biến, hàm, package, bảng, cột**: tiếng Anh.
- Comment giải thích **tại sao**, không phải **cái gì**. Code đã nói cái gì rồi.

Ví dụ đúng (`internal/service/image_variants.go`):

```go
// Encoder WebP. golang.org/x/image/webp chỉ giải mã được, không có encoder;
// còn nativewebp thì chỉ hỗ trợ VP8L (lossless) - với tranh vẽ và ảnh chụp,
// lossless cho ra file lớn gấp nhiều lần JPEG nên vô dụng ở đây.
```

## 3. Xử lý lỗi

- Bọc lỗi kèm ngữ cảnh: `fmt.Errorf("không giải mã được ảnh: %w", err)`.
- **Không** `panic` trong đường xử lý request.
- Lỗi ở khâu **phụ trợ** không được làm hỏng thao tác **chính**. Mẫu chuẩn là
  `buildVariants`: sinh biến thể ảnh lỗi thì chỉ ghi log, vì ảnh gốc đã an toàn trên S3 và
  frontend có đường lui. Bắt người dùng upload lại từ đầu vì một khâu tối ưu hỏng là thiệt
  hơn nhiều.
- Thông báo lỗi trả về cho người dùng bằng tiếng Việt, không lộ chi tiết nội bộ.

## 4. Cấu hình

- Mọi cấu hình đọc từ biến môi trường qua `internal/config/builder.go`. **Không** hard-code
  hằng số môi trường trong code nghiệp vụ.
- Thêm biến mới thì phải làm đủ **ba** việc: đọc trong `builder.go`, thêm vào `.env.example`
  kèm chú thích, ghi vào `docs/deploys/02-configuration.md`.
- Cấu hình nguy hiểm ở production phải **chặn khởi động**, không phải cảnh báo rồi chạy
  tiếp — xem mẫu kiểm tra `CORS_ORIGINS != "*"` trong `builder.go`.

## 5. Cơ sở dữ liệu

- MySQL 8, bảng mã `utf8mb4` (tên tiếng Việt có dấu, bình luận có emoji).
- Migration đánh số tăng dần trong `internal/database/migrations/`, **không bao giờ sửa
  file đã commit** — thêm file mới.
- Mỗi migration phải idempotent (`IF NOT EXISTS`, `IF EXISTS`).
- Mọi truy vấn nhận tham số từ người dùng phải dùng placeholder `?`. Không nối chuỗi SQL.
- Tránh N+1: dùng mẫu batch đã có (`awardRepo.ListByArtworkIDs`) thay vì lặp truy vấn.

## 6. Frontend

- React 19 + TypeScript, `function` component, hooks.
- **Ảnh tác phẩm luôn đi qua `web/src/lib/artworkImage.ts`** — không tự viết logic chọn
  `variants`/`thumbnail_url`/`image_url` ở từng chỗ. Không phải tác phẩm nào cũng có biến
  thể, và đường lui phải nhất quán.
- Route mới thêm bằng `React.lazy` + `RouteFallback`, không import tĩnh.
- Thư viện nặng chỉ dùng ở một khu vực (như recharts ở admin) phải tách chunk trong
  `vite.config.ts`.
- CSS dùng biến trong `web/src/styles/tokens.css`, không hard-code màu.
- Responsive bắt buộc: mobile (≤480px), tablet (≤768px), desktop (≥1280px).

## 7. Kiểm thử

- Chạy `go build ./...` và `go test ./...` trước khi commit. **Cả hai phải sạch.**
- Tính năng mới ở tầng service phải có test. Ưu tiên fake/stub (xem `fakeUploadService`
  trong `internal/service/artwork_variants_test.go`) hơn là mock framework.
- Test phải khoá lại **hành vi có chủ đích**, đặc biệt các quyết định dễ bị vô tình đảo
  ngược — ví dụ "biến thể lỗi thì không được làm hỏng cả lần upload".

## 8. Git

- Commit **theo từng giai đoạn có nghĩa**, không dồn một commit lớn. Backend / frontend /
  tài liệu / hạ tầng là các commit riêng.
- Định dạng: `<type>(<scope>): <mô tả tiếng Việt>` — `feat`, `fix`, `docs`, `test`,
  `refactor`, `chore`.
- Thân commit giải thích **tại sao**, không liệt kê lại file đã sửa (git diff làm việc đó).
- Không commit `.env`, binary, `node_modules/`, dump database.
- File `.sh`, `.service`, `.conf` phải giữ line ending LF (đã cấu hình trong
  `.gitattributes`) — CRLF sẽ làm bash trên VPS lỗi `\r: command not found`.

## 9. Bảo mật

- Bí mật chỉ nằm trong `.env` (quyền `600`, chủ sở hữu `appuser`). Không đưa vào code, log,
  hay thông báo lỗi.
- Mọi input từ người dùng phải validate ở tầng service, không tin tưởng frontend.
- File upload kiểm tra magic byte, không chỉ tin phần mở rộng.
- Thêm endpoint public thì phải cân nhắc rate limit — mẫu có sẵn trong `container.go`.

---

## Tài liệu tham chiếu

| Cần gì | Đọc |
|---|---|
| Bản đồ toàn bộ tài liệu | [docs/README.md](docs/README.md) |
| Kiến trúc tổng thể | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Từng package Go làm gì | [docs/MODULES.md](docs/MODULES.md) |
| Tham chiếu API | [docs/API.md](docs/API.md) |
| Thiết kế chi tiết theo miền | [docs/detail_design/](docs/detail_design/README.md) |
| Deploy lần đầu | [docs/deploys/00-tu-dau-den-cuoi.md](docs/deploys/00-tu-dau-den-cuoi.md) |
| Vận hành, sự cố | [docs/deploys/03-operations.md](docs/deploys/03-operations.md) |
| Việc tiếp theo | [docs/plan/02-roadmap.md](docs/plan/02-roadmap.md) |

# VASchools Art Gallery — S3 Upload Tool + Admin Panel + Trang public 20 năm VASchools

HTTP API service (Go) cho hội thi vẽ tranh "20 năm Trường Việt Mỹ": upload ảnh lên Amazon S3,
quản trị tác phẩm/giải thưởng qua admin panel (đăng nhập Google OAuth), và trang public kỷ niệm
20 năm thành lập Hệ thống Trường Việt Mỹ (không cần đăng nhập). React UI phục vụ cả 3 khu vực:
trang public (`/`), admin panel (`/admin`), công cụ upload nội bộ (`/upload`).

## Documentation

📚 **[Toàn bộ tài liệu → docs/](./docs/README.md)** — bắt đầu từ đây.

| Tài liệu | Nội dung |
|---|---|
| **[Architecture](./docs/ARCHITECTURE.md)** | Kiến trúc, luồng request, chuỗi middleware, mô hình dữ liệu |
| **[Detail Design](./docs/detail_design/README.md)** | Thiết kế chi tiết 6 miền: CSDL, upload, tác phẩm, tương tác public, bảo mật, frontend |
| **[Module Breakdown](./docs/MODULES.md)** | Từng package Go: trách nhiệm, phụ thuộc, lưu ý khi sửa |
| **[API Reference](./docs/API.md)** | Tham chiếu đầy đủ endpoint: request/response/mã lỗi |
| **[Deploys](./docs/deploys/README.md)** | Triển khai VPS, cấu hình, vận hành, xử lý sự cố |
| **[Plan](./docs/plan/README.md)** | Trạng thái hiện tại, lộ trình, rủi ro |

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+ (cho web UI)
- **MySQL 8+** (bắt buộc cho admin panel/trang public — dùng [ServBay](https://www.servbay.com/) khi phát triển local trên macOS/Windows, hoặc MySQL server bất kỳ)
- AWS S3 bucket và credentials
- Google Cloud Console OAuth Client (cho đăng nhập admin) — xem hướng dẫn bên dưới

### Cấu hình cơ bản (.env)

Copy `.env.example` thành `.env` rồi điền:

```env
PORT=8080
AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=your-bucket-name

DATABASE_ENABLED=true
DATABASE_DRIVER=mysql
DATABASE_URL=root:your-password@tcp(localhost:3306)/va_stu_pic_db_prd
```

`DATABASE_URL` dùng format `user:pass@tcp(host:port)/dbname` — `parseTime=true`, `charset=utf8mb4`,
`multiStatements=true` được tự động bổ sung nếu thiếu (xem `internal/database.NormalizeMySQLDSN`),
không cần gõ tay. Database sẽ tự tạo bảng khi server khởi động lần đầu (`DATABASE_AUTO_MIGRATE=true`
mặc định) — không cần chạy `cmd/migrate` thủ công trừ khi muốn kiểm soát riêng.

### Thiết lập Google OAuth (đăng nhập admin)

1. Vào [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials), tạo **OAuth 2.0 Client ID** loại "Web application".
2. Thêm **Authorized redirect URI** khớp chính xác domain sẽ deploy, ví dụ:
   - Dev local: `http://localhost:8080/auth/google/callback`
   - Production: `https://<domain-của-bạn>/auth/google/callback`
3. Copy Client ID/Secret vào `.env`:
   ```env
   GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=GOCSPX-xxxxx
   GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
   ADMIN_ALLOWED_EMAIL_DOMAIN=vaschools.edu.vn,hcm.vaschools.edu.vn
   ```
4. Server vẫn khởi động bình thường nếu bỏ trống các biến này — chỉ `/auth/google/*` trả `503` cho tới khi cấu hình đầy đủ, không chặn phần còn lại của hệ thống.

`ADMIN_ALLOWED_EMAIL_DOMAIN` giới hạn admin chỉ đăng nhập được bằng email thuộc các domain liệt kê
(phẩy phân tách nếu nhiều domain); để trống = không giới hạn (chỉ nên dùng khi dev).

#### Whitelist tài khoản đăng nhập

Hệ thống chỉ cho phép **đúng các email trong whitelist** đăng nhập admin. Email ngoài danh sách bị
từ chối ngay ở bước callback OAuth — không được tự tạo tài khoản admin, kể cả khi cùng domain trường.

Danh sách mặc định (9 tài khoản) khai báo tại `defaultAllowedAdminEmails` trong
[internal/config/builder.go](internal/config/builder.go). Để đổi mà không sửa code, set `ADMIN_ALLOWED_EMAILS`
trong `.env` (phẩy phân tách):

```env
ADMIN_ALLOWED_EMAILS=khoana@hcm.vaschools.edu.vn,toanbq@vaschools.edu.vn
```

Thứ tự ưu tiên: `ADMIN_ALLOWED_EMAILS` (whitelist email chính xác) **cao hơn**
`ADMIN_ALLOWED_EMAIL_DOMAIN`. Khi whitelist không rỗng, kiểm tra domain bị bỏ qua hoàn toàn.
Email cũng phải đã được Google xác thực (`email_verified`) mới đăng nhập được.

### Web UI

```bash
cd web
npm install
npm run build
```

Sau đó chạy Go server — UI được phục vụ tại `http://localhost:8080` từ `web/dist`.

**Dev mode** (Vite hot reload + proxy `/api` và `/auth`):

```bash
# terminal 1
go run cmd/server/main.go

# terminal 2
cd web
npm run dev
```

Mở `http://localhost:5173`.

### Chạy server / migration thủ công

```bash
# Chạy server (migration tự chạy nếu DATABASE_AUTO_MIGRATE=true, mặc định bật)
go run cmd/server/main.go

# Chạy/kiểm tra migration thủ công (tuỳ chọn)
go run cmd/migrate/main.go -up
go run cmd/migrate/main.go -status
```

Không có UI build sẵn, `/` vẫn trả JSON thông tin API.

## Các khu vực chính

| Route | Mô tả | Yêu cầu đăng nhập |
|---|---|---|
| `/` | Công cụ upload S3 nội bộ (Lumen UI) — giữ nguyên, không thay đổi | Không |
| `/admin/login` | Đăng nhập admin qua Google OAuth | — |
| `/admin` | Dashboard: thống kê tổng số tranh, phân bổ khu vực/khối lớp, top 10 tác phẩm | Có |
| `/admin/artworks` | Danh sách tác phẩm — tìm kiếm, lọc, sửa, xoá, đánh dấu tiêu biểu | Có |
| `/admin/artworks/upload` | Upload hàng loạt tác phẩm lên S3 rồi gắn metadata (học sinh/trường/khối/giải) | Có |
| `/admin/awards` | Quản lý giải thưởng (tên, màu, icon, thứ tự) | Có |
| `/trien-lam` | Trang public kỷ niệm 20 năm VASchools — gallery, phòng triển lãm theo khối, billboard vinh danh | Không |

## Features

- Upload S3 với resize/optimize (giữ nguyên toàn bộ pipeline cũ): single-shot, chunked upload tới 200MB, WordPress-style resize
- Admin panel: đăng nhập Google OAuth (session cookie lưu MySQL, không in-memory), quản lý tác phẩm/giải thưởng, dashboard thống kê (recharts)
- Trang public ẩn danh: xem gallery theo khu vực/khối lớp, react (6 loại cảm xúc) và bình luận không cần tài khoản — chống spam bằng rate-limit riêng (20 req/phút/IP) cho các thao tác ghi
- Toast thông báo có âm thanh (Web Audio API tổng hợp, không dùng file audio)
- CSRF, CORS, API key authentication, rate limiting, concurrency limiting cho toàn bộ API
- Health check và metrics endpoints

## License

MIT

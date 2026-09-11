# 02 — Tham chiếu cấu hình

Toàn bộ biến môi trường đọc từ `.env` (hoặc biến môi trường hệ thống). Nguồn sự thật:
`internal/config/builder.go`.

Ứng dụng nạp `.env` một lần lúc khởi động (`sync.Once`). **Đổi `.env` phải khởi động lại**
service mới có hiệu lực.

## Môi trường

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `APP_ENV` | `development` | `production` bật kiểm tra bắt buộc + cookie secure |

Khi `APP_ENV=production`, server **từ chối khởi động** nếu:

| Điều kiện | Thông báo lỗi |
|---|---|
| `API_REQUIRE_KEY≠true` hoặc `API_KEY` rỗng | `production requires API_REQUIRE_KEY=true and a non-empty API_KEY` |
| `CORS_ORIGINS` là `*` hoặc rỗng | `production requires an explicit CORS_ORIGINS allowlist (not *)` |

`APP_ENV=production` cũng tự bật `Secure` cho cookie CSRF và cookie session.

## AWS / S3

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `AWS_REGION` | `us-east-1` | Tự ghi đè nếu bucket thật ở region khác |
| `S3_BUCKET_NAME` | — | **Bắt buộc** |
| `S3_BASE_PATH` | rỗng | Tiền tố khoá, ví dụ `vaschools-uploads` |
| `S3_ENDPOINT` | rỗng | Chỉ cho MinIO/LocalStack |
| `S3_FORCE_PATH_STYLE` | `false` | Tự bật khi có `S3_ENDPOINT` |
| `S3_USE_ACL` | `false` | Bucket AWS hiện đại nên dùng policy thay ACL |
| `S3_USE_PRESIGNED_URL` | `false` | ⚠️ Xem cảnh báo bên dưới |
| `S3_PRESIGNED_URL_EXPIRY` | `60` | Phút |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | — | Bỏ trống nếu dùng IAM role |

Thông tin đăng nhập AWS được tìm theo thứ tự: biến môi trường → IAM role → `~/.aws/credentials`
→ EC2 metadata. **Ưu tiên IAM role** khi chạy trên EC2/ECS — không có khoá tĩnh nào để rò rỉ.

⚠️ **`S3_USE_PRESIGNED_URL` phải là `false` cho kho ảnh công khai.** URL ký sẵn được lưu
vào `artworks.s3_url` lúc upload và **hết hạn** sau `S3_PRESIGNED_URL_EXPIRY` phút — ảnh
trên trang public sẽ trả 403 sau đó. Muốn ảnh công khai lâu dài thì mở quyền đọc bằng bucket
policy, xem [S3-PUBLIC-READ.md](../S3-PUBLIC-READ.md).

Ứng dụng cảnh báo trong log nếu phát hiện khoá AWS còn là giá trị mẫu (`your_access_key…`,
`changeme`).

## Máy chủ HTTP

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `PORT` | `8080` | Nginx proxy tới cổng này |
| `SERVER_READ_TIMEOUT_SECONDS` | `15` | `.env.example` gợi ý `180` cho upload lớn |
| `SERVER_WRITE_TIMEOUT_SECONDS` | `30` | `.env.example` gợi ý `180` |
| `SERVER_IDLE_TIMEOUT_SECONDS` | `60` | |
| `SERVER_SHUTDOWN_TIMEOUT_SECONDS` | `5` | Thời gian chờ request đang chạy khi tắt |

⚠️ Read/write timeout mặc định trong code (15s/30s) **thấp hơn** giá trị trong `.env.example`
(180s). Nếu `.env` không khai báo, upload file lớn có thể bị cắt giữa chừng. Nên khai báo
tường minh ở production.

## Giới hạn upload

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `UPLOAD_MAX_SIZE_MB` | `20` | Trần mặc định cho một request upload |
| `UPLOAD_ABSOLUTE_MAX_MB` | `200` | Trần cứng cho một file ảnh tác phẩm (đường bulk upload của admin) |
| `UPLOAD_TIMEOUT_SECONDS` | `300` | Timeout thao tác S3 |

Một điều chỉnh tự động cần biết: `AbsoluteMaxSize` **không bao giờ nhỏ hơn** `MaxSize` — tự
nâng lên nếu cấu hình đặt ngược.

⚠️ `client_max_body_size` trong Nginx phải **≥** `UPLOAD_ABSOLUTE_MAX_MB`, nếu không Nginx
chặn request trước khi ứng dụng nhìn thấy nó (trả 413).

## Cơ sở dữ liệu

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `DATABASE_ENABLED` | `false` | ⚠️ **Phải `true`** cho admin/public |
| `DATABASE_DRIVER` | `mysql` | Chỉ hỗ trợ MySQL |
| `DATABASE_URL` | — | `user:pass@tcp(host:port)/dbname` |
| `DATABASE_MAX_OPEN` | `25` | |
| `DATABASE_MAX_IDLE` | `5` | |
| `DATABASE_MAX_LIFETIME` | `300` | Giây |
| `DATABASE_AUTO_MIGRATE` | `true` | Chạy migration lúc khởi động |

DSN được tự bổ sung `parseTime=true`, `charset=utf8mb4`, `multiStatements=true` nếu thiếu —
không cần gõ tay, và giá trị bạn tự đặt sẽ không bị ghi đè.

Kết nối DB kiểm tra ngay lúc khởi động với timeout 5 giây. ⚠️ **DB không chạy = server
không khởi động được** (fail-fast, không thử lại). Với systemd `Restart=on-failure`, service
sẽ tự thử lại mỗi 5 giây — hữu ích khi MySQL khởi động chậm hơn ứng dụng sau khi reboot.

## Đăng nhập admin

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `GOOGLE_CLIENT_ID` | rỗng | Trống → `/auth/google/*` trả 503 |
| `GOOGLE_CLIENT_SECRET` | rỗng | |
| `GOOGLE_REDIRECT_URL` | rỗng | Phải khớp **chính xác** với Google Console |
| `SESSION_SECRET` | rỗng | Dùng phía server; session thật lưu DB |
| `SESSION_TTL_HOURS` | `168` | 7 ngày; giá trị < 1 bị đưa về mặc định |
| `ADMIN_ALLOWED_EMAIL_DOMAIN` | rỗng | Nhiều domain cách nhau bởi dấu phẩy |
| `ADMIN_ALLOWED_EMAILS` | 9 email mặc định | **Ưu tiên cao hơn** kiểm tra domain |

Tên cookie session cố định trong code: `vas_admin_session`.

Thứ tự quyết định quyền đăng nhập:

```text
ADMIN_ALLOWED_EMAILS không rỗng → chỉ các email đó (bỏ qua domain)
                        rỗng   → xét ADMIN_ALLOWED_EMAIL_DOMAIN
                                   có  → phải thuộc domain
                                   rỗng → không giới hạn  ⚠️
```

Để trống `ADMIN_ALLOWED_EMAILS` **không** có nghĩa là mở cửa: code dùng danh sách mặc định
9 tài khoản ban quản trị trong `builder.go:20`. Xem
[detail_design/05-auth-security.md](../detail_design/05-auth-security.md).

## Bảo mật API

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `API_ENABLED` | `true` | |
| `API_REQUIRE_KEY` | `false` | Bắt buộc `true` ở production |
| `API_KEY` | rỗng | Sinh bằng `openssl rand -base64 32` |
| `CORS_ORIGINS` | `*` | ⚠️ Không được là `*` ở production |
| `CSRF_ENABLED` | `true` | Tắt nếu triển khai thuần API |
| `CSRF_SECURE_COOKIE` | theo `APP_ENV` | `true` khi đã có HTTPS |

### ⚠️ Xung đột cần biết: API key và trang public

Bật `API_REQUIRE_KEY=true` áp middleware lên **toàn bộ** `/api/*`, bao gồm cả
`/api/v1/public/*`. Nhưng production lại **bắt buộc** bật nó. Hệ quả: trang public sẽ đòi
API key mà khách ẩn danh không có.

Ba cách xử lý, theo thứ tự khuyến nghị:

1. **Frontend cùng gốc (khuyến nghị).** SPA được chính binary Go phục vụ, nên nếu bạn nhúng
   API key vào biến build của frontend thì key lộ ra client — **không nên**. Thay vào đó,
   xem cách 2.
2. **Loại trừ `/api/v1/public/` khỏi kiểm tra API key** ở tầng ứng dụng. Đây là thay đổi
   code nhỏ trong `container.go` và là hướng đúng về lâu dài — xem
   [plan/02-roadmap.md](../plan/02-roadmap.md).
3. **Chặn ở Nginx thay vì API key**: giữ `API_REQUIRE_KEY=false` (khi đó phải để
   `APP_ENV=development`, mất các kiểm tra production khác) và hạn chế `/api/v1/upload*`
   theo IP trong vhost. Kém nhất vì mất các bảo vệ khác.

Hãy kiểm tra thực tế trạng thái trang public sau khi bật API key ở production trước khi
công bố tên miền.

## Rate limit và đồng thời

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `RATE_LIMIT_ENABLED` | `true` | |
| `RATE_LIMIT_REQUESTS` | `100` | Mỗi cửa sổ, mỗi IP |
| `RATE_LIMIT_WINDOW_MINUTES` | `1` | |
| `RATE_LIMIT_CLEANUP_MINUTES` | `10` | Chu kỳ dọn bộ nhớ |
| `CONCURRENCY_LIMIT_ENABLED` | `true` | |
| `MAX_CONCURRENT_UPLOADS` | `500` | Cân nhắc hạ theo RAM VPS |
| `CONCURRENCY_ACQUIRE_TIMEOUT_SECONDS` | `30` | Chờ trước khi báo lỗi |

Hai giới hạn **cố định trong code**, không cấu hình được: metrics 10 req/phút, và ghi dữ
liệu public 20 req/phút.

Không còn route tải ảnh gốc công khai — đã gỡ có chủ đích (xem
[03-artwork-domain.md](../detail_design/03-artwork-domain.md) và
[03-risks.md](../plan/03-risks.md)), nên bộ đếm riêng cho nó (`RATE_LIMIT_DOWNLOAD_*`) cũng
đã bị gỡ theo. Khu quản trị vẫn tải được ảnh gốc qua
`GET /api/v1/admin/artworks/{id}/download`, dùng chung rate limit của toàn bộ API admin.

## Proxy tin cậy

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `TRUSTED_PROXIES` | `127.0.0.0/8,::1/128` | Dải được phép đặt `X-Forwarded-For`/`X-Real-IP` |

⚠️ **Biến này quyết định toàn bộ rate limit có thật sự hoạt động hay không.** Ứng dụng chỉ
đọc header chuyển tiếp khi chặng kết nối trực tiếp nằm trong danh sách này; ngược lại dùng
thẳng địa chỉ kết nối. Không có kiểm tra đó thì bất kỳ ai cũng chỉ cần thêm một header
`X-Forwarded-For` ngẫu nhiên vào mỗi request là có bộ đếm mới, và mọi giới hạn theo IP —
kể cả 20 req/phút cho bình luận — trở thành vô hiệu.

Ba tình huống triển khai:

| Cách chạy | Giá trị đúng |
|---|---|
| Nginx cùng máy (mặc định của dự án) | `127.0.0.0/8,::1/128` |
| App phơi thẳng ra Internet | **Để rỗng** |
| Sau Cloudflare / load balancer riêng | Dải IP của dịch vụ đó |

Khai sai dải ở tình huống thứ ba khiến **mọi khách bị gom vào một bộ đếm** — cả trang tự
khoá đúng lúc đông người nhất, đúng kịch bản R6 trong [plan/03-risks.md](../plan/03-risks.md).

## Chống tải trọn site

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `BOT_GUARD_ENABLED` | `true` | Bật lớp nhận diện công cụ tải hàng loạt |
| `BOT_GUARD_MAX_REQUESTS_PER_MINUTE` | `240` | Số request/phút/IP trước khi bị coi là máy quét |
| `BOT_GUARD_MAX_PATHS_PER_MINUTE` | `150` | Số **đường dẫn khác nhau**/phút — dấu hiệu đặc trưng của trình tải site |
| `BOT_GUARD_BLOCK_MINUTES` | `10` | Thời gian giữ hình phạt sau khi vượt ngưỡng |

Bot tìm kiếm và bot mạng xã hội hợp lệ (Googlebot, bingbot, `facebookexternalhit`,
coccocbot, Zalo…) được **miễn hoàn toàn** — chi tiết cơ chế ở
[detail_design/05-auth-security.md](../detail_design/05-auth-security.md).

## Header bảo mật

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `SECURITY_HSTS_ENABLED` | theo `APP_ENV` | Để trống = bật khi `production` |
| `SECURITY_CSP_IMAGE_SOURCES` | rỗng | Origin ảnh ngoài; domain S3 đã tự suy ra |
| `SECURITY_CSP_CONNECT_SOURCES` | rỗng | Origin được phép gọi XHR/fetch tới |
| `MAX_JSON_BODY_KB` | `1024` | Trần body cho endpoint không phải upload |

⚠️ **Chỉ bật HSTS khi site đã chạy HTTPS hoàn toàn.** Header này bảo trình duyệt từ chối
mọi kết nối HTTP tới tên miền trong suốt `max-age` (một năm) — bật nhầm khi còn phục vụ
HTTP sẽ khoá người dùng khỏi site và không thể gỡ từ phía máy chủ.

Domain S3 được suy tự động từ `S3_BUCKET_NAME`/`AWS_REGION`/`S3_ENDPOINT`, còn
`lh3.googleusercontent.com` (avatar admin từ Google OAuth) đã khai sẵn trong code, nên bình
thường không cần khai `SECURITY_CSP_IMAGE_SOURCES`. Chỉ thêm khi phục vụ ảnh qua CDN riêng —
thiếu thì CSP chặn đúng ảnh tác phẩm, và lỗi chỉ lộ ra trên trình duyệt người dùng cuối chứ
không xuất hiện trong log server.

`MAX_JSON_BODY_KB` tồn tại vì Nginx đặt `client_max_body_size 200m` ở mức server để đường
upload đi lọt — nếu không có trần riêng ở tầng ứng dụng thì endpoint bình luận cũng nhận
được body 200MB.

## Nhật ký

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `LOG_DIR` | `./storage/logs` | Đọc trực tiếp bằng `os.Getenv` trong `logging` |
| `LOG_PREFIX` | `app` | Tên file: `app-YYYY-MM-DD.log` |

Log ghi đồng thời ra stdout (systemd journal thu) và file theo ngày.
⚠️ **Không có cơ chế tự xoá log cũ** — xem [03-operations.md](./03-operations.md).

> **Đã gỡ (2026-09-07)**: nhóm `WORDPRESS_*` và `IMAGE_*` (resize kiểu WordPress). Code đọc
> chúng đã bị xoá, nên nếu `.env` cũ còn các dòng này thì chúng chỉ nằm đó vô tác dụng — xoá
> đi cho gọn.

## Thư mục và đường dẫn tương đối

| Đường dẫn | Dùng cho | Cấu hình được |
|---|---|---|
| `./uploads` | File tạm trong lúc upload | ❌ cố định trong `builder.go` |
| `./storage/logs` | Log theo ngày | ✅ qua `LOG_DIR` |
| `./web/dist` | Giao diện đã build (SPA) | ❌ cố định |
| `./internal/database/migrations` | Migration khi `DATABASE_AUTO_MIGRATE=true` | ❌ cố định |

⚠️ **Cả bốn đường dẫn trên đều tương đối so với thư mục làm việc.** systemd unit đặt
`WorkingDirectory=/opt/s3-upload-tool` nên chúng trỏ đúng. Chạy binary từ thư mục khác sẽ
gây ba kiểu hỏng khác nhau, đều báo lỗi rõ ràng chứ không im lặng:

- File tạm rơi vào thư mục lạ.
- Giao diện không phục vụ được — `/` trả JSON info thay vì trang web.
- **Auto-migrate không tìm thấy file migration** — app không khởi động được.

## Mẫu `.env` cho production

```env
APP_ENV=production

AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=vas-art-gallery
S3_BASE_PATH=vaschools-uploads
S3_USE_ACL=false
S3_USE_PRESIGNED_URL=false

PORT=8080
SERVER_READ_TIMEOUT_SECONDS=180
SERVER_WRITE_TIMEOUT_SECONDS=180
SERVER_SHUTDOWN_TIMEOUT_SECONDS=30

UPLOAD_MAX_SIZE_MB=20
UPLOAD_ABSOLUTE_MAX_MB=200
UPLOAD_TIMEOUT_SECONDS=300

DATABASE_ENABLED=true
DATABASE_DRIVER=mysql
DATABASE_URL=vasapp:<mật-khẩu>@tcp(localhost:3306)/va_stu_pic_db_prd
DATABASE_AUTO_MIGRATE=true

API_REQUIRE_KEY=true
API_KEY=<openssl rand -base64 32>
CORS_ORIGINS=https://pictures.vaschools.edu.vn
CSRF_ENABLED=true
CSRF_SECURE_COOKIE=true

GOOGLE_CLIENT_ID=<client-id>
GOOGLE_CLIENT_SECRET=<client-secret>
GOOGLE_REDIRECT_URL=https://pictures.vaschools.edu.vn/auth/google/callback
SESSION_TTL_HOURS=168
ADMIN_ALLOWED_EMAILS=<danh sách email ban tổ chức>

RATE_LIMIT_ENABLED=true
CONCURRENCY_LIMIT_ENABLED=true
MAX_CONCURRENT_UPLOADS=100

# Nginx chạy cùng máy nên chỉ tin loopback. Sai dòng này là hỏng toàn bộ
# giới hạn theo IP - xem mục "Proxy tin cậy" ở trên.
TRUSTED_PROXIES=127.0.0.1/32,::1/128
SECURITY_HSTS_ENABLED=true
BOT_GUARD_ENABLED=true
MAX_JSON_BODY_KB=1024

LOG_DIR=./storage/logs
```

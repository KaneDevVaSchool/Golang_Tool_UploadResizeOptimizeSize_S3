# 05 — Xác thực và bảo mật

**Code chính**: `internal/auth/`, `internal/handlers/admin_auth_handler.go`,
`internal/middleware/{admin_auth,csrf,apikey,ratelimit,cors,concurrency}.go`

## 1. Ba mô hình xác thực song song

Hệ thống dùng ba cơ chế khác nhau cho ba nhóm người dùng — không phải sự thiếu nhất quán mà
là kết quả của ba yêu cầu khác nhau:

| Khu vực | Cơ chế | Vì sao chọn |
|---|---|---|
| `/api/v1/admin/*` | Session cookie (Google OAuth) | Admin là người thật, cần thu hồi được quyền ngay |
| `/api/v1/public/*` | Không xác thực + rate limit | Yêu cầu nghiệp vụ: tương tác không rào cản |
| `/api/v1/upload*` | API key (tuỳ chọn) | Client máy-với-máy, không có trình duyệt để giữ session |

## 2. Luồng đăng nhập admin

```text
① Người dùng bấm "Đăng nhập với Google"
   GET /auth/google/login
     ├─ chưa cấu hình Client ID/Secret → 503 với thông báo rõ ràng
     ├─ sinh state ngẫu nhiên 24 byte
     ├─ đặt cookie vas_oauth_state (HttpOnly, TTL 5 phút, SameSite=Lax)
     └─ redirect sang Google

② Google xác thực người dùng, gọi về
   GET /auth/google/callback?code=...&state=...
     ├─ đọc + xoá ngay cookie state, so khớp với query    ← chống CSRF
     ├─ đổi code lấy access token
     ├─ gọi Google UserInfo API
     ├─ kiểm tra email_verified == true                   ← bắt buộc
     ├─ kiểm tra email nằm trong danh sách cho phép
     ├─ tìm admin_users theo google_sub
     │     chưa có → tạo mới (chỉ khi đã qua kiểm tra ở trên)
     ├─ tạo session trong DB, đặt cookie HttpOnly
     └─ redirect về /admin

③ Mọi request admin sau đó
   cookie session → AdminAuthMiddleware → ValidateSession → gắn user vào context
```

### Vì sao bắt buộc `email_verified`

Whitelist so khớp theo **chuỗi email**. Nếu không kiểm tra `email_verified`, một người có
thể tạo tài khoản Google khai email trùng danh sách cho phép mà chưa xác minh, và đi thẳng
vào admin. Kiểm tra này (`admin_auth_handler.go:186-191`) đóng lối đó.

### Hai tầng danh sách cho phép

```text
ADMIN_ALLOWED_EMAILS không rỗng?
   ├─ Có  → CHỈ các email trong danh sách được vào (bỏ qua hoàn toàn kiểm tra domain)
   └─ Không → xét ADMIN_ALLOWED_EMAIL_DOMAIN
                ├─ có domain → email phải thuộc một trong các domain
                └─ rỗng     → KHÔNG GIỚI HẠN  ⚠️ chỉ dùng khi dev
```

Whitelist email cụ thể **ưu tiên cao hơn** domain (`admin_auth_handler.go:84-88`). Chủ đích:
domain trường có hàng nghìn tài khoản (giáo viên, học sinh), nhưng chỉ chín người trong ban
tổ chức được vào admin.

Khi `.env` để trống `ADMIN_ALLOWED_EMAILS`, code dùng danh sách mặc định gồm 9 tài khoản
ban quản trị, khai báo tại `config/builder.go:20` (`defaultAllowedAdminEmails`). Nghĩa là
**mặc định đã an toàn** — quên cấu hình không dẫn tới mở toang admin.

⚠️ Chỉ khi cả hai biến đều rỗng *và* danh sách mặc định bị sửa thành rỗng thì mới thành
"không giới hạn". Đừng làm vậy ở production.

### Tự tạo tài khoản khi đăng nhập lần đầu

Không có màn hình "mời admin". Người trong danh sách cho phép đăng nhập lần đầu sẽ được tạo
bản ghi `admin_users` tự động (`admin_auth_handler.go:206-217`). An toàn vì việc kiểm tra
danh sách đã diễn ra **trước** đó — danh sách cho phép chính là cơ chế mời.

## 3. Session

| Thuộc tính | Giá trị | Lý do |
|---|---|---|
| Nơi lưu | Bảng `admin_sessions` | Thu hồi được ngay; JWT thì không |
| Token | 32 byte `crypto/rand`, base64 url-safe | Không đoán được |
| TTL | `SESSION_TTL_HOURS`, mặc định 168 (7 ngày) | Cân bằng tiện dụng và rủi ro |
| Cookie | `HttpOnly` | JavaScript không đọc được → XSS không lấy được session |
| `Secure` | Theo `APP_ENV=production` | Chỉ gửi qua HTTPS ở production |
| `SameSite` | `Lax` | **Bắt buộc** — callback OAuth là điều hướng cross-site |

**Vì sao `SameSite=Lax` chứ không `Strict`.** Sau khi Google chuyển hướng về, trình duyệt
coi đó là điều hướng từ site khác. `Strict` sẽ **không gửi cookie** trong request đó, và
người dùng vừa đăng nhập xong lại thấy mình chưa đăng nhập. `Lax` gửi cookie cho điều hướng
GET cấp cao nhất — vừa đủ cho luồng này, vẫn chặn CSRF qua form POST.

**Dọn session hết hạn**: goroutine chạy mỗi giờ (`container.go:393-414`), dừng sạch qua
channel khi tắt máy. Việc kiểm tra hết hạn vẫn diễn ra ở mỗi lần xác thực, nên goroutine
này chỉ để bảng không phình — không phải cơ chế bảo mật.

**Đăng xuất luôn thành công**: endpoint logout đặt ngoài `AdminAuthMiddleware`. Session đã
chết phía server thì request vẫn đi lọt để xoá cookie phía client.

## 4. CSRF

Áp dụng **toàn cục** (mọi route, không riêng admin) theo mẫu double-submit cookie:

```text
GET/HEAD/OPTIONS   → bỏ qua kiểm tra; nếu chưa có cookie thì sinh token và đặt vào
POST/PUT/PATCH/DELETE → so cookie csrf_token với header X-CSRF-Token (hoặc field form)
                         không khớp / thiếu → 403 INVALID_CSRF
```

| Thuộc tính | Giá trị | Lưu ý |
|---|---|---|
| Cookie | `csrf_token` | `HttpOnly=false` — **bắt buộc**, JS phải đọc được để gửi lại |
| Header | `X-CSRF-Token` | |
| `SameSite` | `Strict` | Khác cookie session (`Lax`) vì token này không cần sống sót qua điều hướng cross-site |
| TTL | 3600 giây | |

Việc `HttpOnly=false` **không phải lỗ hổng**: bản chất double-submit là client phải đọc
được token để gửi lại. Bảo vệ đến từ chỗ site khác không đọc được cookie của domain này
(same-origin policy), chứ không phải từ việc giấu token khỏi JavaScript.

⚠️ **Client không phải trình duyệt** (curl, script tích hợp) sẽ bị 403 khi gọi `POST`. Hoặc
lấy token qua một `GET` trước rồi gửi kèm, hoặc đặt `CSRF_ENABLED=false` nếu triển khai
thuần API. Xem [API.md](../API.md).

## 5. API key

```text
API_REQUIRE_KEY=true  →  mọi /api/* phải có header X-API-Key
```

Hai điểm về thiết kế:

- **Không bao giờ nhận key qua query string** — query string bị ghi vào log Nginx, log
  ứng dụng và lịch sử trình duyệt.
- `/api/v1/health` **luôn miễn xác thực**, để load balancer và giám sát thăm dò được.

⚠️ Khi bật API key, **toàn bộ** `/api/*` bị áp — kể cả `/api/v1/public/*`. Nghĩa là trang
public sẽ ngừng hoạt động với người xem ẩn danh. Nếu vừa muốn public mở vừa muốn bảo vệ
upload, hãy giữ `API_REQUIRE_KEY=false` và dựa vào session cho admin + rate limit cho public.

## 6. Rate limit

Ba bộ đếm độc lập, mỗi bộ có `map[IP]counter` riêng:

| Phạm vi | Giới hạn | Áp cho |
|---|---|---|
| Toàn cục | `RATE_LIMIT_REQUESTS`/`WINDOW` (mặc định 100/phút) | Mọi request |
| `/api/v1/metrics` | 10/phút, **cố định trong code** | Chống dò thông tin vận hành |
| Ghi dữ liệu public | 20/phút | **Chỉ** `POST`/`DELETE` dưới `/api/v1/public/*` |

Bộ thứ ba lồng trong bộ thứ nhất: một request POST bình luận tính vào **cả hai** bộ đếm.

Mỗi `RateLimiter` chạy goroutine dọn bộ nhớ theo `RATE_LIMIT_CLEANUP_MINUTES`. Container giữ
danh sách mọi limiter đã tạo và `Stop()` tất cả khi tắt máy (`container.go:609-613`) —
không rò rỉ goroutine.

⚠️ Bộ đếm nằm **trong bộ nhớ tiến trình**. Restart là mất, và nhiều instance sẽ không dùng
chung. Chấp nhận được với một instance; nếu scale ngang phải chuyển sang Redis.

## 7. Giới hạn đồng thời

Semaphore toàn server (`golang.org/x/sync/semaphore`) giới hạn số request xử lý cùng lúc.
Vượt quá thì **chờ** tối đa `CONCURRENCY_ACQUIRE_TIMEOUT_SECONDS` rồi mới trả lỗi — xếp
hàng thay vì từ chối ngay, vì upload ảnh là thao tác người dùng chủ động và chờ vài giây
tốt hơn báo lỗi.

Mặc định `MAX_CONCURRENT_UPLOADS=500` là khá rộng; điều chỉnh theo RAM thực tế của VPS, vì
mỗi upload đang xử lý giữ một file tạm và bộ đệm.

## 8. Kiểm tra bắt buộc ở production

`config/builder.go:426-432` **chặn server khởi động** nếu `APP_ENV=production` mà:

| Điều kiện | Thông báo |
|---|---|
| `API_REQUIRE_KEY=false` hoặc `API_KEY` rỗng | `production requires API_REQUIRE_KEY=true and a non-empty API_KEY` |
| `CORS_ORIGINS=*` hoặc rỗng | `production requires an explicit CORS_ORIGINS allowlist (not *)` |

Fail-fast có chủ đích: cấu hình sai bị phát hiện lúc khởi động, không phải sau khi đã chạy
và để lộ dữ liệu. ⚠️ Lưu ý ràng buộc thứ nhất kéo theo cảnh báo ở mục 5 — bật API key ở
production sẽ ảnh hưởng trang public; xem [deploys/02-configuration.md](../deploys/02-configuration.md)
để biết cách xử lý.

## 9. Bảo vệ khác rải rác trong code

| Bảo vệ | Nơi thực hiện |
|---|---|
| Chống path traversal ở tên file | `utils.SanitizeFilename()` |
| Chống path traversal khi phục vụ SPA | `spaFileServer` kiểm tra đường dẫn tuyệt đối nằm trong `dist` |
| Chặn liệt kê thư mục | `secureFileServer` trả 404 cho đường dẫn kết thúc bằng `/` |
| Chống SQL injection | Toàn bộ truy vấn dùng tham số `?`, kể cả WHERE động |
| Không lộ dữ liệu nhạy cảm | `visitor_token`, `ip_address`, `s3_key` đánh `json:"-"` |
| Không lộ quyền sở hữu | Xoá bình luận trả cùng lỗi cho "không tồn tại" và "không phải của bạn" |
| Chống XSS | React escape khi render; `html/template` escape ở trang chia sẻ |
| Chuẩn hoá thông tin lỗi | `handlers/error_mapper.go` — `sanitizeError` không trả chi tiết nội bộ ra ngoài |

## 10. Những gì hệ thống **không** bảo vệ

Nêu rõ để không ai hiểu nhầm về mức bảo đảm:

- **Danh tính người tương tác public.** `visitor_token` giả mạo được. Xem
  [04-public-engagement.md](./04-public-engagement.md).
- **Phân quyền theo vai trò.** Cột `role` tồn tại nhưng mọi admin có quyền như nhau —
  không có phân biệt người duyệt/người xem.
- **Nhật ký thao tác admin.** Không ghi lại ai xoá tác phẩm nào lúc nào. Chỉ có
  `artworks.created_by` cho việc tạo.
- **Mã hoá dữ liệu khi lưu.** Dựa vào mã hoá ở tầng đĩa/dịch vụ, không mã hoá ở tầng ứng dụng.
- **Chặn tấn công từ chối dịch vụ phân tán.** Rate limit theo IP trong bộ nhớ không đủ; cần
  Cloudflare hoặc tương đương.

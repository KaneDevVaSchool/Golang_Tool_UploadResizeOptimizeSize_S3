# Kiến trúc hệ thống

> Đối chiếu code tại commit `d0ec7c2`. Xem [detail_design/](./detail_design/README.md) cho
> đặc tả chi tiết từng miền, [MODULES.md](./MODULES.md) cho từng package.

## 1. Hệ thống này là gì

Một service Go duy nhất phục vụ **hai nhóm người dùng qua trình duyệt**, cộng một nhóm
client máy-với-máy, trên cùng một cổng HTTP:

| Khu vực | Đường dẫn | Người dùng | Xác thực |
|---|---|---|---|
| Trang public "20 năm VASchools" | `/`, `/tac-pham-tieu-bieu`, `/phong-trien-lam`, `/bang-vang` | Phụ huynh, học sinh, khách | Không — ẩn danh hoàn toàn |
| Admin panel | `/admin/*` | Ban tổ chức hội thi | Google OAuth + session cookie |
| API upload S3 | `/api/v1/upload*` | Client máy-với-máy (không qua trình duyệt) | API key (tuỳ cấu hình) |

Hai khu vực trình duyệt dùng chung **một binary, một database, một bucket S3**, phân biệt
bằng route và tầng middleware chứ không tách service. Lựa chọn này phù hợp quy mô hiện tại
(một sự kiện, vài nghìn tác phẩm) và giữ chi phí vận hành ở mức một VPS.

⚠️ Trang frontend `/upload` (`UploadTool.tsx` — giao diện thao tác trực tiếp API trên bằng
tay) đã bị xoá; endpoint `/api/v1/upload*` **không đổi**, chỉ không còn UI nội bộ để gọi thủ
công qua trình duyệt. Xem [detail_design/06-frontend.md](./detail_design/06-frontend.md).

## 2. Sơ đồ tầng

```text
                        ┌──────────────────────────────────┐
   Trình duyệt  ───────▶│  Nginx (443)  reverse proxy      │
                        │  client_max_body_size 200m       │
                        └───────────────┬──────────────────┘
                                        │ 127.0.0.1:8080
                        ┌───────────────▼──────────────────┐
                        │  cmd/server/main.go              │
                        │  .env → logging → Container      │
                        │  → http.Server → graceful stop   │
                        └───────────────┬──────────────────┘
                                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │  internal/container — composition root duy nhất                │
   │  dựng mọi dependency, wire route + middleware, Shutdown()      │
   └───────┬──────────────────────────────────────────────┬─────────┘
           │                                              │
   ┌───────▼─────────┐                          ┌─────────▼──────────┐
   │  Middleware     │                          │  internal/handlers │
   │  (chain toàn    │─────────────────────────▶│  HTTP ⇄ JSON       │
   │   cục + cục bộ) │                          │  không có logic    │
   └─────────────────┘                          └─────────┬──────────┘
                                                          ▼
                                                ┌────────────────────┐
                                                │  internal/service  │
                                                │  nghiệp vụ, tx,    │
                                                │  điều phối         │
                                                └─────────┬──────────┘
                                                          ▼
                                                ┌────────────────────┐
                                                │ internal/repository│
                                                │  SQL + AWS SDK     │
                                                └────┬──────────┬────┘
                                                     ▼          ▼
                                              ┌──────────┐  ┌────────┐
                                              │ MySQL 8  │  │  S3    │
                                              └──────────┘  └────────┘
```

Quy tắc phụ thuộc **một chiều**: `handlers → service → repository → database/models`.
Không tầng nào gọi ngược lên. `utils`, `metrics`, `logging`, `middleware` là leaf package
dùng chung, không phụ thuộc ngược.

## 3. Composition root

`internal/container/container.go` là nơi **duy nhất** biết cách dựng hệ thống. Không có
biến toàn cục, không `init()` ẩn, không service locator. Mọi thứ được truyền qua constructor.

### Khởi tạo có điều kiện

Container dựng hệ thống theo **năng lực sẵn có**, không fail toàn bộ khi thiếu một phần:

| Điều kiện | Nếu tắt/thiếu thì sao |
|---|---|
| `DATABASE_ENABLED=false` | Toàn bộ admin + domain + public API **không được mount**. Upload S3 thuần vẫn chạy. |
| Thiếu `GOOGLE_CLIENT_ID/SECRET` | `AdminAuthHandler` **vẫn được tạo**, `/auth/google/*` trả 503 có thông báo rõ thay vì 404. |
| Chưa build `web/dist` | `/` trả JSON info endpoint thay vì lỗi — tiện khi dev backend riêng. ⚠️ Watermark ảnh tải về cũng đọc mốc từ đây, nên thiếu `web/dist` thì ảnh public tải về **không có watermark** (chỉ ghi log, không báo lỗi). |
| `BOT_GUARD_ENABLED=false` | Không chặn công cụ tải trọn site; rate limit thường vẫn còn. |
| `TRUSTED_PROXIES` rỗng | Bỏ qua `X-Forwarded-For`, mọi giới hạn tính theo địa chỉ kết nối trực tiếp — sau Nginx nghĩa là **toàn bộ khách gộp thành một IP**. |

Chủ ý: người vận hành dựng dần từng phần (chạy upload trước, thêm DB sau, thêm OAuth cuối)
mà không lần nào gặp màn hình trắng không rõ nguyên nhân.

Hai khối `if db != nil` trong `NewContainer` (admin auth và domain) là
nơi thể hiện quy tắc này.

### Tự dò region của bucket

`resolveBucketRegion()` hỏi S3 xem bucket thật nằm ở region nào và
**ghi đè** `AWS_REGION` nếu lệch. Lý do: cấu hình sai region là lỗi khó chẩn đoán nhất khi
deploy (upload báo lỗi ký request mơ hồ). Bỏ qua bước này khi dùng endpoint tuỳ chỉnh
(MinIO/LocalStack) vì `GetBucketLocation` không áp dụng.

## 4. Chuỗi middleware

Middleware được áp theo thứ tự **áp sau = nằm ngoài**. Request đi từ ngoài vào:

```text
Request
  │
  ├─▶ ConcurrencyLimit   semaphore toàn server, chờ tối đa ACQUIRE_TIMEOUT
  ├─▶ BotGuard           chặn công cụ tải trọn site; bot tìm kiếm được miễn
  ├─▶ RateLimit          100 req/phút/IP (mặc định)
  ├─▶ BodyLimit          1MB cho JSON, UPLOAD_ABSOLUTE_MAX_MB cho đường upload
  ├─▶ CSRF               double-submit cookie; bỏ qua GET/HEAD/OPTIONS
  ├─▶ Metrics            đếm request/latency/status
  ├─▶ Logging            ghi log kèm request ID
  ├─▶ RequestID          sinh/nhận X-Request-ID
  ├─▶ SecurityHeaders    CSP, Permissions-Policy, COOP/CORP, HSTS có điều kiện
  ├─▶ Gzip               nén text/JSON > 1KB  ◀── trong cùng, sát mux nhất
  │
  ▼
mux gốc
  ├── /api/*        ─▶ [CORS] ─▶ [APIKeyAuth nếu bật] ─▶ apiMux
  │                                                        ├── /api/v1/upload*      công khai theo API key
  │                                                        ├── /api/v1/admin/*   ─▶ [AdminAuth] session
  │                                                        ├── /api/v1/public/*  ─▶ [RateLimit 20/phút cho POST|DELETE]
  │                                                        │                        [RateLimit 30/phút cho GET .../download]
  │                                                        └── /api/v1/metrics   ─▶ [RateLimit 10/phút]
  ├── /auth/google/*   luồng redirect OAuth — CỐ Ý nằm ngoài /api để không dính CORS/API-key
  ├── /chia-se/tac-pham/{id}   HTML có Open Graph, render server-side
  ├── /robots.txt              text/plain, Disallow /admin /api /auth, trỏ Sitemap
  ├── /sitemap.xml             XML, 4 URL tĩnh + tác phẩm đã publish (qua /chia-se/tac-pham/{id})
  ├── /wp-content/uploads/*    static, đã chặn liệt kê thư mục
  └── /                        SPA React, fallback index.html
```

Năm điểm đáng chú ý về thiết kế chain này:

**Rate limit phân tầng.** Bốn bộ đếm độc lập: toàn cục (100/phút), metrics (10/phút), ghi
dữ liệu public (20/phút), và tải ảnh gốc (30/phút). Hai bộ cuối chỉ áp theo method/đường
dẫn — người xem lướt trang (toàn `GET` thường) không bao giờ chạm tới, nhưng người spam
bình luận hoặc gom tranh hàng loạt thì có.

**BotGuard nằm ngoài RateLimit.** Có chủ đích: máy quét bị loại **trước** khi kịp tiêu tốn
hạn mức chung của những người thật cùng đi ra từ một IP NAT — trường học dùng chung một IP
ra ngoài. Đảo thứ tự hai lớp này sẽ khiến một công cụ tải site làm cạn hạn mức của cả trường.

**BodyLimit nằm ngoài CSRF.** Cũng có chủ đích: thân request quá khổ bị cắt trước khi bất
kỳ lớp nào đọc nó, kể cả lớp đọc form của CSRF.

**Logout không qua AdminAuth.** `/api/v1/admin/auth/logout` đăng ký *ngoài*
`AdminAuthMiddleware` (trong `GetServerHandler`). Nếu session đã hết hạn phía server, request
logout vẫn phải đi lọt để xoá cookie phía client — trả 401 cho người chỉ muốn đăng xuất là
vô nghĩa.

**OAuth nằm ngoài `/api`.** Luồng Google là redirect trình duyệt, không phải JSON API.
Đặt trong `/api` sẽ khiến CORS middleware và API-key middleware can thiệp vào một luồng
mà chúng không hiểu.

## 5. Ba luồng request chính

### 5.1 Upload đơn — `POST /api/v1/upload`

```text
multipart/form-data
  → ParseMultipartForm, lấy file, validate type + size theo header
  → UploadService.UploadImage
      → tạo temp file trong UPLOAD_DIR
      → io.Copy có giới hạn LimitReader(maxSize+1) để phát hiện file vượt hạn
      → timeout copy co giãn theo dung lượng: 60s + 2s/MB, trần 5 phút
      → S3Repository.Upload (multipart, part 8MB, concurrency 4)
      → sinh URL: object URL trực tiếp, hoặc presigned nếu bật
  → defer xoá temp file
  ← {success, data:{url, key, size, name}}
```

Chi tiết đầy đủ: [detail_design/02-upload-pipeline.md](./detail_design/02-upload-pipeline.md).

### 5.2 Tải ảnh tác phẩm về

```text
GET /api/v1/public/artworks/{id}/download   → chỉ tác phẩm đã xuất bản
GET /api/v1/admin/artworks/{id}/download    → mọi tác phẩm, kể cả đang ẩn
  → S3Repository.GetObject (stream qua backend, không redirect)
  → nhánh public: đóng watermark VAS vào góc dưới phải
  → ghi 1 dòng vào artwork_downloads (lỗi ghi log chỉ cảnh báo, không chặn tải)
  ← Content-Disposition: attachment
```

Cả hai đi qua backend thay vì trả link S3: cùng origin nên không phụ thuộc cấu hình CORS của
bucket, và đó là chỗ duy nhất chèn được watermark cùng nhật ký. Đổi lại, băng thông ảnh đi
qua VPS chứ không thẳng từ S3 về máy khách.

### 5.3 Xem tác phẩm trên trang public

```text
GET /api/v1/public/artworks/{id}?visitor_token=<uuid>
  → ArtworkService.GetArtwork → enrich (học sinh/trường/khối/giải/reaction/comment)
  → nếu artwork chưa publish → 404 (không lộ tồn tại)
  → ArtworkViewRepository.RecordView: ghi artwork_views + tăng view_count
      mỗi lần gọi là một lượt, không còn chống trùng theo thời gian
  → đọc lại artwork từ DB để trả về view_count vừa cập nhật
  ← ArtworkWithMeta đầy đủ
```

## 6. Mô hình dữ liệu — nhìn tổng thể

```text
  admin_users ──1:N──▶ admin_sessions
       │
       │ created_by
       ▼
   artworks ◀──N:1── students ──N:1──▶ schools     (16 cơ sở → 3 region)
       │                    └────N:1──▶ grade_levels (12 khối → 2 cấp)
       │
       ├──N:N──▶ awards        (qua artwork_awards)
       ├──1:N──▶ artwork_reactions   (ẩn danh, visitor_token)
       ├──1:N──▶ artwork_comments    (ẩn danh, visitor_token)
       ├──1:N──▶ artwork_views       (mỗi lần mở là 1 dòng)
       ├──1:N──▶ artwork_downloads   (nhật ký tải ảnh gốc)
       └──N:1──▶ uploads             (bảng chết, không còn ghi)
```

`artworks` giữ **bản sao** `school_id` và `grade_level_id` dù đã có qua `students`. Đây là
denormalize có chủ đích: dashboard đếm theo khu vực/khối chạy trực tiếp trên `artworks`,
không phải JOIN qua `students` mỗi lần thống kê. Đánh đổi: khi đổi trường của học sinh phải
cập nhật cả hai nơi.

Chi tiết từng bảng: [detail_design/01-database.md](./detail_design/01-database.md).

## 7. Quyết định kiến trúc và lý do

| Quyết định | Lý do | Đánh đổi |
|---|---|---|
| Một binary phục vụ cả 3 khu vực | Quy mô một sự kiện; vận hành 1 VPS; không cần điều phối service | Không scale riêng từng khu vực được |
| Session admin lưu DB | Sống sót qua restart/deploy — admin không bị đá ra mỗi lần cập nhật | Mỗi request admin tốn 1 query |
| Tương tác public ẩn danh | Yêu cầu nghiệp vụ: phụ huynh xem là thả tim được ngay, không đăng ký | Chống lạm dụng chỉ dựa vào rate limit + visitor_token |
| Xoá tác phẩm **xoá luôn** file S3 | Không để bucket tích rác không ai dọn; xoá S3 trước, hỏng thì giữ nguyên bản ghi DB nên không có bản ghi trỏ vào ảnh đã mất | Bấm nhầm là mất ảnh vĩnh viễn — đổi từ quyết định ngược lại ngày 2026-09-07 |
| Ảnh tải về đi qua backend | Chỗ duy nhất đóng được watermark và ghi nhật ký; không phụ thuộc CORS của bucket | Băng thông ảnh dồn qua VPS |
| Migration tự chạy lúc khởi động | Deploy một bước, không quên chạy migrate | Cần cẩn trọng khi có nhiều instance |
| `enrichArtworks` batch query | Tránh N+1 khi hiển thị danh sách kèm giải/reaction | Vẫn còn 1 query/artwork cho student (xem nợ kỹ thuật) |

## 8. Tắt máy an toàn

Thứ tự trong `main.go:57-70` là quan trọng và **không được đảo**:

1. Nhận `SIGINT`/`SIGTERM`.
2. `server.Shutdown(ctx)` — ngừng nhận request mới, chờ request đang chạy xong (tối đa
   `SERVER_SHUTDOWN_TIMEOUT_SECONDS`).
3. `container.Shutdown()` — dừng rate limiter, dừng BotGuard, dừng cleanup session, đóng DB.

Đảo ngược hai bước cuối sẽ khiến request đang xử lý mất kết nối DB/S3 giữa chừng.

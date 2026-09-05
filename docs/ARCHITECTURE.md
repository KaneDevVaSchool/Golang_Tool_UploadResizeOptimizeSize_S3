# Kiến trúc hệ thống

> Đối chiếu code tại commit `d0ec7c2`. Xem [detail_design/](./detail_design/README.md) cho
> đặc tả chi tiết từng miền, [MODULES.md](./MODULES.md) cho từng package.

## 1. Hệ thống này là gì

Một service Go duy nhất phục vụ **ba nhóm người dùng khác nhau** trên cùng một cổng HTTP:

| Khu vực | Đường dẫn | Người dùng | Xác thực |
|---|---|---|---|
| Trang public "20 năm VAS" | `/`, `/tac-pham-tieu-bieu`, `/phong-trien-lam`, `/bang-vang` | Phụ huynh, học sinh, khách | Không — ẩn danh hoàn toàn |
| Admin panel | `/admin/*` | Ban tổ chức hội thi | Google OAuth + session cookie |
| Công cụ upload nội bộ | `/upload` | Nhân viên kỹ thuật | API key (tuỳ cấu hình) |

Ba khu vực dùng chung **một binary, một database, một bucket S3**, phân biệt bằng route và
tầng middleware chứ không tách service. Lựa chọn này phù hợp quy mô hiện tại (một sự kiện,
vài nghìn tác phẩm) và giữ chi phí vận hành ở mức một VPS.

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
| `WORDPRESS_ENABLED=false` | `/api/v1/wp-upload` và `/wp-content/uploads/` không mount. |
| Chưa build `web/dist` | `/` trả JSON info endpoint thay vì lỗi — tiện khi dev backend riêng. |

Chủ ý: người vận hành dựng dần từng phần (chạy upload trước, thêm DB sau, thêm OAuth cuối)
mà không lần nào gặp màn hình trắng không rõ nguyên nhân.

`container.go:187-202` (admin auth) và `container.go:288-312` (domain) là hai khối
`if db != nil` thể hiện quy tắc này.

### Tự dò region của bucket

`resolveBucketRegion()` (`container.go:661`) hỏi S3 xem bucket thật nằm ở region nào và
**ghi đè** `AWS_REGION` nếu lệch. Lý do: cấu hình sai region là lỗi khó chẩn đoán nhất khi
deploy (upload báo lỗi ký request mơ hồ). Bỏ qua bước này khi dùng endpoint tuỳ chỉnh
(MinIO/LocalStack) vì `GetBucketLocation` không áp dụng.

## 4. Chuỗi middleware

Middleware được áp theo thứ tự **áp sau = nằm ngoài**. Request đi từ ngoài vào:

```text
Request
  │
  ├─▶ ConcurrencyLimit   semaphore toàn server, chờ tối đa ACQUIRE_TIMEOUT
  ├─▶ RateLimit          100 req/phút/IP (mặc định)
  ├─▶ CSRF               double-submit cookie; bỏ qua GET/HEAD/OPTIONS
  ├─▶ Metrics            đếm request/latency/status
  ├─▶ Logging            ghi log kèm request ID
  ├─▶ RequestID          sinh/nhận X-Request-ID  ◀── trong cùng, chạy đầu tiên
  │
  ▼
mux gốc
  ├── /api/*        ─▶ [CORS] ─▶ [APIKeyAuth nếu bật] ─▶ apiMux
  │                                                        ├── /api/v1/upload*      công khai theo API key
  │                                                        ├── /api/v1/admin/*   ─▶ [AdminAuth] session
  │                                                        ├── /api/v1/public/*  ─▶ [RateLimit 20/phút cho POST|DELETE]
  │                                                        └── /api/v1/metrics   ─▶ [RateLimit 10/phút]
  ├── /auth/google/*   luồng redirect OAuth — CỐ Ý nằm ngoài /api để không dính CORS/API-key
  ├── /chia-se/tac-pham/{id}   HTML có Open Graph, render server-side
  ├── /wp-content/uploads/*    static, đã chặn liệt kê thư mục
  └── /                        SPA React, fallback index.html
```

Ba điểm đáng chú ý về thiết kế chain này:

**Rate limit phân tầng.** Có ba bộ đếm độc lập: toàn cục (100/phút), metrics (10/phút),
và ghi dữ liệu public (20/phút). Bộ thứ ba chỉ áp cho `POST`/`DELETE` — người xem lướt
trang (toàn `GET`) không bao giờ chạm giới hạn này, nhưng người spam bình luận thì có.
Xem `container.go:515-527`.

**Logout không qua AdminAuth.** `/api/v1/admin/auth/logout` đăng ký *ngoài*
`AdminAuthMiddleware` (`container.go:484`). Nếu session đã hết hạn phía server, request
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

### 5.2 Upload chunked — file đến 200MB

```text
POST /upload/init      → ChunkSession (UUID), tính totalChunks, tạo uploads/chunks/<id>/
POST /upload/chunk     → ghi part_%06d, kiểm tra kích thước từng phần khớp kỳ vọng
POST /upload/complete  → đủ chunk? → ghép theo thứ tự → ValidateFileContent (magic byte)
                         → upload S3 → dọn session + thư mục
POST /upload/abort     → huỷ, xoá chunk đã nhận
```

Session lưu **in-memory** (`map[string]*ChunkSession`) — cố ý, vì chunk session chỉ sống
trong một lần upload ngắn. Restart server làm mất session dở dang, chấp nhận được. Ngược
lại, session admin lưu DB để sống sót qua restart. Hai lựa chọn trái ngược nhau nhưng đều
đúng với vòng đời tương ứng.

Tự dọn: TTL 45 phút, quét mỗi 5 phút, trần 64 session đồng thời.

### 5.3 Xem tác phẩm trên trang public

```text
GET /api/v1/public/artworks/{id}?visitor_token=<uuid>
  → ArtworkService.GetArtwork → enrich (học sinh/trường/khối/giải/reaction/comment)
  → nếu artwork chưa publish → 404 (không lộ tồn tại)
  → ArtworkViewRepository.RecordView: đã xem trong 24h chưa?
      chưa → ghi artwork_views + tăng view_count
      rồi  → bỏ qua, không tăng
  ← ArtworkWithMeta đầy đủ
```

## 6. Mô hình dữ liệu — nhìn tổng thể

```text
  admin_users ──1:N──▶ admin_sessions
       │
       │ created_by
       ▼
   artworks ◀──N:1── students ──N:1──▶ schools     (5 cơ sở → 3 region)
       │                    └────N:1──▶ grade_levels (12 khối → 2 cấp)
       │
       ├──N:N──▶ awards        (qua artwork_awards)
       ├──1:N──▶ artwork_reactions   (ẩn danh, visitor_token)
       ├──1:N──▶ artwork_comments    (ẩn danh, visitor_token)
       ├──1:N──▶ artwork_views       (chống đếm trùng 24h)
       └──N:1──▶ uploads             (lịch sử upload, nullable)
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
| Chunk session in-memory | Vòng đời ngắn, không đáng ghi DB | Mất session dở khi restart |
| Tương tác public ẩn danh | Yêu cầu nghiệp vụ: phụ huynh xem là thả tim được ngay, không đăng ký | Chống lạm dụng chỉ dựa vào rate limit + visitor_token |
| Xoá tác phẩm **không** xoá file S3 | Bấm nhầm không mất dữ liệu vĩnh viễn | S3 tích rác, cần dọn định kỳ (xem [plan/03-risks.md](./plan/03-risks.md)) |
| Migration tự chạy lúc khởi động | Deploy một bước, không quên chạy migrate | Cần cẩn trọng khi có nhiều instance |
| `enrichArtworks` batch query | Tránh N+1 khi hiển thị danh sách kèm giải/reaction | Vẫn còn 1 query/artwork cho student (xem nợ kỹ thuật) |

## 8. Tắt máy an toàn

Thứ tự trong `main.go:57-70` là quan trọng và **không được đảo**:

1. Nhận `SIGINT`/`SIGTERM`.
2. `server.Shutdown(ctx)` — ngừng nhận request mới, chờ request đang chạy xong (tối đa
   `SERVER_SHUTDOWN_TIMEOUT_SECONDS`).
3. `container.Shutdown()` — dừng rate limiter, dọn chunk session, dừng cleanup session, đóng DB.

Đảo ngược hai bước cuối sẽ khiến request đang xử lý mất kết nối DB/S3 giữa chừng.

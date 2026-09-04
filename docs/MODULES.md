# Module Breakdown

Tài liệu này bóc tách chi tiết từng package/module trong codebase: trách nhiệm, API công khai, phụ thuộc, và điểm cần lưu ý khi sửa đổi. Đọc [ARCHITECTURE.md](./ARCHITECTURE.md) trước nếu cần bức tranh tổng thể trước khi đi vào chi tiết từng module.

## Sơ đồ phụ thuộc giữa các package

```
cmd/server, cmd/migrate
        │
        ▼
internal/container   ──depends on──▶  config, database, handlers, metrics, middleware, repository, service
        │
        ▼
internal/handlers    ──depends on──▶  database, metrics, middleware, models, repository, service, utils
        │
        ▼
internal/service      ──depends on──▶  database, models, repository, utils, metrics
        │
        ▼
internal/repository   ──depends on──▶  database, models   (+ AWS SDK v2 trực tiếp)
        │
        ▼
internal/{config, database, models, utils, metrics, logging, middleware}   ← không phụ thuộc lẫn nhau (leaf packages)
```

Quy tắc phụ thuộc một chiều: `handlers → service → repository → database/models`. `utils` và `metrics` là thư viện dùng chung, không phụ thuộc ngược lên các layer trên. `middleware` độc lập hoàn toàn (chỉ dùng `net/http` + `google/uuid` + `golang.org/x/sync/semaphore`), được `container` wire vào chain chứ tự nó không biết gì về service/handler.

---

## `cmd/server` — Entry point

**File**: [main.go](../cmd/server/main.go)

Trách nhiệm duy nhất: load `.env` → setup file logging → gọi `container.NewContainer()` → khởi động `http.Server` → lắng nghe `SIGINT`/`SIGTERM` → graceful shutdown theo thứ tự: **drain HTTP request đang chạy trước (`server.Shutdown`), rồi mới đóng container** (DB, rate limiter goroutines, chunk sessions). Thứ tự này quan trọng — đảo ngược sẽ làm request đang xử lý mất kết nối DB/S3 giữa chừng.

Không chứa business logic — nếu cần thêm behavior khi khởi động/tắt server, sửa ở `container.go`, không sửa `main.go`.

## `cmd/migrate` — Migration runner

**File**: [cmd/migrate/main.go](../cmd/migrate/main.go)

Chạy SQL migration trong `internal/database/migrations/*.sql` (hiện có 1 file: `001_create_uploads_table.sql`). Dùng cờ `-up`. Không có cơ chế `-down`/rollback tự động — xem trực tiếp file migration nếu cần biết cấu trúc bảng.

---

## `internal/config` — Cấu hình

**Files**: [config.go](../internal/config/config.go), [builder.go](../internal/config/builder.go), [errors.go](../internal/config/errors.go)

- **`config.go`**: định nghĩa struct `Config` và các sub-struct (`ServerConfig`, `AWSConfig`, `UploadConfig`, `DatabaseConfig`, `DirectoriesConfig`, `RateLimitConfig`, `CSRFConfig`, `ConcurrencyConfig`, `APIConfig`, `WordPressConfig`, `ImageOptimizationConfig`, `ImageSizeConfig`). Đây là **nguồn sự thật duy nhất** cho toàn bộ giá trị cấu hình dùng trong ứng dụng.
- **`builder.go`**: implement builder pattern (`ConfigBuilder.With*()` từng phần → `Build()`). `BuildFromEnv()` là entry point thật sự dùng trong `container.go` — đọc toàn bộ biến môi trường, áp default, rồi gọi các hàm `With*`.
- **`errors.go`**: `ErrMissingBucketName`.

**Validate quan trọng cần biết khi sửa**:
- `Build()` bắt buộc `S3_BUCKET_NAME` không rỗng, và nếu `API_REQUIRE_KEY=true` thì `API_KEY` không được rỗng.
- `BuildFromEnv()` có thêm validate **cứng cho `APP_ENV=production`**: bắt buộc `API_REQUIRE_KEY=true` + `API_KEY` không rỗng, và `CORS_ORIGINS` phải là allowlist cụ thể (không được là `*` hoặc rỗng) — nếu vi phạm, `Load()` trả lỗi và server không khởi động được (xem `container.go` gọi `config.Load()` → `log.Fatalf` nếu lỗi).
- Ở production, nếu người dùng không set `CSRF_SECURE_COOKIE` thì builder **tự động bật `true`** thay vì giữ default `false`.
- `LoadEnvFile()` dùng `sync.Once` — chỉ load `.env`/`.env.local` một lần dù gọi nhiều lần (an toàn khi `main.go` và `builder.go` đều gọi).
- `WithUpload()` tự kẹp `AbsoluteMaxSize` sao cho `absoluteMax / maxSize <= 64` — khớp với `maxChunksPerUpload` hard-code trong `service/chunk_upload.go`. **Nếu đổi hằng số 64 ở một nơi, phải đổi luôn ở nơi kia** (hiện không có single source of truth giữa 2 package cho con số này).

## `internal/container` — Composition root (DI)

**File**: [container.go](../internal/container/container.go)

Đây là nơi **wiring toàn bộ ứng dụng** — không chứa business logic, chỉ khởi tạo và kết nối dependency theo đúng thứ tự:

1. `config.Load()`
2. Tạo thư mục `uploads/`, `wp-uploads/` (nếu WordPress bật)
3. Khởi tạo AWS SDK v2 config (credential chain) + tự dò region thật của bucket qua `resolveBucketRegion` (gọi `GetBucketLocation` qua `us-east-1` endpoint) nếu khác `AWS_REGION` cấu hình — tránh lỗi `PermanentRedirect` phổ biến khi bucket ở region khác region cấu hình
4. Tạo `s3.Client` + `manager.Uploader` (part size 8MB, concurrency 4)
5. Khởi tạo DB (nếu `DATABASE_ENABLED=true`)
6. Tạo `RepositoryFactory` → `S3Repository`; `ServiceFactory` → `UploadService`; tạo `ChunkUploadService` trực tiến (không qua factory)
7. Tạo `APIHandler`, và `WPHandler` (nếu WordPress bật)

`GetServerHandler()` build toàn bộ `http.ServeMux` + middleware chain (chi tiết xem [ARCHITECTURE.md § Middleware chain](./ARCHITECTURE.md#middleware-chain-thứ-tự-áp-dụng-ngoài-cùng-trước)). Cũng chứa 2 helper quan trọng:
- `secureFileServer`: wrap `http.FileServer` để chặn directory listing (trả 404 cho path kết thúc bằng `/`) — dùng cho `/wp-content/uploads/`.
- `spaFileServer`: serve React SPA từ `web/dist`, có path-traversal guard riêng (`filepath.Clean` + check prefix) trước khi fallback về `index.html` cho client-side routing; nếu `web/dist/index.html` không tồn tại, trả JSON info endpoint thay vì lỗi 404 (dev-friendly).

`Shutdown()` dọn theo thứ tự: dừng rate limiters → dừng chunk upload cleanup goroutine (xoá luôn mọi session dở dang trên disk) → đóng DB.

⚠️ `warnPlaceholderAWSCredentials()` chỉ log cảnh báo (không chặn khởi động) nếu phát hiện key/secret chứa `your_access`/`your_secret`/`changeme` — không phải validate an toàn thật sự, chỉ để tránh nhầm lẫn khi copy `.env.example`.

## `internal/database` — DB wrapper

**File**: [database.go](../internal/database/database.go)

Wrapper mỏng quanh `*sql.DB`/`*sql.Tx` (driver `go-sql-driver/mysql`, tức chỉ hỗ trợ MySQL dù field tên là `Driver` generic — đã chuyển từ `lib/pq`/PostgreSQL). `NewDB()` ping ngay lúc khởi tạo với timeout 5s — nếu DB down lúc container khởi động, server **không start được** (fail-fast, không retry). `NormalizeMySQLDSN()` tự bổ sung `parseTime=true`/`charset=utf8mb4`/`multiStatements=true` vào DSN nếu thiếu (không ghi đè giá trị đã set). `migrate.go` (`RunMigrations`) tự chạy toàn bộ `internal/database/migrations/*.sql` theo thứ tự khi container khởi động (bật/tắt qua `DATABASE_AUTO_MIGRATE`), idempotent qua bảng `schema_migrations` — song song tồn tại với `cmd/migrate` (CLI chạy tay/kiểm tra status), cả hai dùng chung format bảng nhưng độc lập vòng đời.

## `internal/models` — Data structures

**Files**: [upload.go](../internal/models/upload.go), [upload_record.go](../internal/models/upload_record.go)

Thuần struct, không có logic. `UploadResponse{URL, Key}` dùng chung cho mọi luồng upload (single-shot, chunked, transaction). `UploadRecord` map trực tiếp với bảng `uploads` trong MySQL — `UploadStatus` chỉ có 3 giá trị: `pending` / `completed` / `failed`.

Ngoài `upload.go`/`upload_record.go` còn 11 model domain cho hội thi vẽ tranh (`admin_user`, `session`, `school`, `grade_level`, `student`, `artwork`, `award`, `artwork_award`, `reaction`, `comment`, `artwork_view`) — xem section [Admin panel + trang public "20 năm VAS"](#admin-panel--trang-public-20-năm-vas) bên dưới.

## `internal/repository` — Data access layer

**Files**: [s3_repository.go](../internal/repository/s3_repository.go), [upload_repository.go](../internal/repository/upload_repository.go), [factory.go](../internal/repository/factory.go), [errors.go](../internal/repository/errors.go)

- **`S3Repository`** (interface): `Upload`, `GeneratePresignedURL`, `CheckConnectivity`. Implementation mỏng quanh AWS SDK v2 (`s3manager.Uploader` cho multipart upload, `s3.PresignClient` cho presigned URL, `HeadBucket` cho health check). Không có logic nghiệp vụ — chỉ dịch tham số sang AWS SDK call.
- **`UploadRepository`** (interface): CRUD cho bảng `uploads`, nhận `*database.Tx` cho ghi (bắt buộc chạy trong transaction) và `*database.DB` cho đọc. Dùng **prepared statement** cho 2 query đọc (`GetUploadByID`, `GetUploadsByStatus`) — có `Close()` riêng để đóng statement khi shutdown (⚠️ container hiện **không gọi** `uploadRepo.Close()` trong `Shutdown()` — statement leak nhẹ khi tắt server, không nghiêm trọng vì process kết thúc ngay sau đó nhưng đáng sửa nếu container còn sống lâu hơn, ví dụ dùng trong test).
- **`RepositoryFactory`**: chỉ hỗ trợ `RepositoryTypeS3` — factory pattern ở đây chủ yếu để dễ mở rộng sau này (ví dụ thêm GCS/Azure Blob), hiện tại chỉ có 1 nhánh switch.

## `internal/service` — Business logic

**Files**: [upload_service.go](../internal/service/upload_service.go), [chunk_upload.go](../internal/service/chunk_upload.go), [image_resize.go](../internal/service/image_resize.go), [image_optimizer.go](../internal/service/image_optimizer.go), [factory.go](../internal/service/factory.go), [errors.go](../internal/service/errors.go), [constants.go](../internal/service/constants.go)

### `UploadService` (upload_service.go)
Hai phương thức: `UploadImage` (single-shot) và `UploadImageWithTransaction` (bọc DB transaction, rollback thủ công qua named return + `defer`/`recover` để xử lý cả panic). Cả hai đều theo pattern: **ghi ra temp file trước → validate → upload S3 → build URL**, không bao giờ stream thẳng multipart body lên S3 — lý do là cần biết chính xác size để validate và cần `Seek(0,0)` trước khi PutObject.

Timeout copy file **scale theo kích thước** (`60s + 2s/MB`, tối đa 5 phút) — nếu tăng `UPLOAD_MAX_SIZE_MB` quá cao, các upload gần giới hạn có thể chạm trần 5 phút dù mạng đủ nhanh; cân nhắc điều chỉnh hệ số này nếu đổi giới hạn upload mặc định nhiều.

### `ChunkUploadService` (chunk_upload.go)
Quản lý session upload nhiều phần **hoàn toàn in-memory** (`map[string]*ChunkSession` + `sync.Mutex`), không có backing store. Các hằng số quan trọng: `chunkSessionTTL=45min`, `chunkCleanupEvery=5min`, `maxChunkSessions=64`, `maxChunksPerUpload=64`. `cleanupLoop()` chạy nền dọn session hết hạn; `Stop()` dọn sạch toàn bộ khi shutdown server.

Từng chunk ghi ra file riêng `part_%06d` trên disk (dùng `.tmp` + `os.Rename` để atomic), `Complete()` ghép tuần tự rồi validate content (magic byte) trước khi upload S3 — **không dùng S3 Multipart Upload API thật** (khác với `manager.Uploader` trong `UploadService` — chunk service tự ghép file rồi upload 1 lần bằng `S3Repository.Upload`).

⚠️ **Giới hạn cần nhớ khi scale ngang**: session không share giữa các instance — xem thêm ARCHITECTURE.md.

### `ImageResizeService` + `ImageOptimizer` (image_resize.go, image_optimizer.go)
Chỉ dùng cho luồng WordPress-style (`/api/v1/wp-upload`), hoàn toàn độc lập với S3/service upload chính:
- `ImageResizeService.ResizeImage`: decode ảnh gốc 1 lần, resize ra nhiều size cấu hình sẵn bằng **bilinear interpolation tự viết tay** (không dùng thư viện resize ngoài, chỉ `golang.org/x/image/webp` để decode webp) — xử lý song song theo hàng ảnh bằng worker pool (`runtime.NumCPU()` goroutine).
- `ImageOptimizer.OptimizeImage`: với JPEG/PNG, thử nhiều mức quality/compression rồi giữ bản nhỏ nhất (brute-force, không phải thuật toán tối ưu thật). Quality JPEG tự động điều chỉnh theo kích thước file gốc (`calculateOptimalJPEGQuality` — file càng lớn, nén càng mạnh, xem `constants.go` cho ngưỡng cụ thể).
- Lưu ý: `IMAGE_ENABLE_WEBP=true` **không thực sự encode ra WebP** — `saveImage()` trong `image_resize.go` fallback về JPEG khi format là webp kèm log cảnh báo "WebP encoding not fully supported". Config này hiện chưa có tác dụng thật, cần làm rõ với người dùng nếu họ dựa vào nó.

### `ServiceFactory` (factory.go)
Chỉ hỗ trợ `ServiceTypeUpload` — tương tự `RepositoryFactory`, là factory pattern chuẩn bị cho mở rộng, hiện chỉ có 1 nhánh.

### `errors.go` / `constants.go`
`FileSizeError` là error type duy nhất mang theo dữ liệu (actual/max size) để handler dịch sang response cụ thể. `ErrInvalidFileFormat` liệt kê rõ các loại file được hỗ trợ trong message — đồng bộ với danh sách extension thật trong `utils/file_utils.go` (`IsImage`/`IsDocument`/`IsVideo`/`IsAudio`/`IsArchive`).

---

## `internal/handlers` — HTTP layer

**Files**: [api_handler.go](../internal/handlers/api_handler.go), [chunk_handler.go](../internal/handlers/chunk_handler.go), [wp_handler.go](../internal/handlers/wp_handler.go), [base_handler.go](../internal/handlers/base_handler.go), [error_mapper.go](../internal/handlers/error_mapper.go), [metrics_handler.go](../internal/handlers/metrics_handler.go)

- **`BaseHandler`** (base_handler.go): `SendSuccess`/`SendError`/`SendJSON` — chuẩn hoá format response `{success, data, error}` dùng chung cho mọi handler.
- **`APIHandler`** (api_handler.go): `HandleUpload`, `HandleUploadWithTransaction`, `HandleHealth`. Chứa `processUploadRequest` dùng chung để parse + validate multipart (giới hạn theo `maxUploadSize`).
- **chunk_handler.go**: 4 handler cho luồng chunk (`Init`/`Upload`/`Complete`/`Abort`), tách file riêng khỏi `api_handler.go` dù cùng struct `APIHandler` (Go cho phép định nghĩa method trên cùng struct ở nhiều file — dùng để tổ chức code theo domain thay vì kích thước file).
- **`WPHandler`** (wp_handler.go): 1 handler duy nhất `HandleWPUpload` — lưu ý resize lỗi **không làm fail cả request** (trả 200 kèm ảnh gốc, `sizes: []`).
- **`error_mapper.go`**: `mapS3UploadError` dịch lỗi AWS SDK thô (chuỗi lỗi chứa `PermanentRedirect`, `InvalidAccessKeyId`, `NoSuchBucket`, `AccessDenied`...) thành mã lỗi + message thân thiện, tránh leak nguyên văn lỗi AWS ra client. `sanitizeError`/`errorMapper` xử lý các lỗi non-S3 khác theo pattern-matching chuỗi.
- **`MetricsHandler`** (metrics_handler.go): chỉ có `HandleMetrics`, trả `metrics.GetMetrics().GetStats()`.

⚠️ Lưu ý bảo mật đã đúng cách nhưng dễ vỡ nếu sửa: `mapS3UploadError` match theo **substring lowercase của error message** — nếu AWS SDK đổi format message giữa các version, các case này có thể im lặng rơi vào nhánh `default` (mất đi thông tin hữu ích, nhưng không leak gì thêm — fail an toàn).

## `internal/middleware` — Cross-cutting concerns

**Files**: [cors.go](../internal/middleware/cors.go), [csrf.go](../internal/middleware/csrf.go), [apikey.go](../internal/middleware/apikey.go), [ratelimit.go](../internal/middleware/ratelimit.go), [concurrency.go](../internal/middleware/concurrency.go), [logging.go](../internal/middleware/logging.go), [request_id.go](../internal/middleware/request_id.go), [request_id_test.go](../internal/middleware/request_id_test.go)

| Middleware | Cơ chế | Điểm cần lưu ý |
|---|---|---|
| `RequestIDMiddleware` | Lấy `X-Request-ID` từ upstream proxy nếu có, không thì tạo UUID mới | Duy nhất có test (`request_id_test.go`) |
| `LoggingMiddleware` | Log method/URI/IP/duration sau khi request xử lý xong, kèm request ID nếu có | Không log status code response (chỉ `metrics.MetricsMiddleware` có wrap để bắt status) |
| `CORS` | Whitelist origin, xử lý preflight `OPTIONS` riêng | Nếu `allowedOrigins == ["*"]`, **không set** `Access-Control-Allow-Credentials` (browser cấm kết hợp `*` với credentials) |
| `CSRFProtection` | Double-submit cookie pattern (`csrf_token` cookie so khớp header `X-CSRF-Token`) | Bỏ qua GET/HEAD/OPTIONS; cookie **`HttpOnly: false`** có chủ đích (để JS đọc được) — đây không phải lỗ hổng vì double-submit pattern yêu cầu vậy |
| `APIKeyAuth` | So sánh `X-API-Key` bằng `subtle.ConstantTimeCompare` (chống timing attack) | Luôn bypass cho `/api/v1/health` dù có bật key — chủ đích cho load balancer probe |
| `RateLimiter` | Token bucket theo IP, refill tỷ lệ theo thời gian trôi qua | Lấy IP qua `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` theo thứ tự — **tin tưởng header từ client nếu không có reverse proxy set lại**; triển khai sau Nginx phải đảm bảo Nginx ghi đè (không forward) các header này từ client gốc |
| `ConcurrencyLimiter` | `golang.org/x/sync/semaphore.Weighted`, có timeout khi acquire | Áp dụng cho **toàn bộ mọi route**, không riêng route upload — endpoint tĩnh/health cũng tính vào giới hạn đồng thời |

## `internal/utils` — Thư viện dùng chung

**Files**: [content_validation.go](../internal/utils/content_validation.go), [filename.go](../internal/utils/filename.go), [file_utils.go](../internal/utils/file_utils.go), [file_size.go](../internal/utils/file_size.go), [multipart.go](../internal/utils/multipart.go), [s3_url.go](../internal/utils/s3_url.go), [date_utils.go](../internal/utils/date_utils.go), + file `_test.go` tương ứng cho filename/file_size/s3_url

- **`SanitizeFilename`/`ValidateFilename`** (filename.go): chống path traversal (`..`, `/`, `\`) và ký tự nguy hiểm (`< > : " | ? *`), cắt bớt nếu vượt 255 ký tự nhưng giữ nguyên extension.
- **`ValidateFileContent`** (content_validation.go): đọc 512 byte đầu, dùng `http.DetectContentType` (magic byte) so khớp với extension khai báo — **ảnh được validate chặt hơn** (map extension → danh sách MIME hợp lệ cụ thể) so với file loại khác (chỉ so base type, ví dụ `video/*`).
- **`file_utils.go`**: danh sách extension cho phép theo từng category (`IsImage`/`IsDocument`/`IsVideo`/`IsAudio`/`IsArchive`), `GetContentType` (map extension → MIME để set `Content-Type` khi PutObject S3), `GenerateS3Key` (tạo key dạng `category/name-timestamp-random.ext`, có prefix `basePath` tuỳ chọn).
- **`multipart.go`**: `GetFileFromRequest` chấp nhận field `file` hoặc `image` (backward-compat), `ValidateUploadFile` gộp sanitize + content-check + size-check thành 1 hàm dùng chung cho `APIHandler`/`WPHandler`.
- **`s3_url.go`**: `BuildS3ObjectURL` build URL public/path-style tuỳ `endpoint`/`forcePathStyle` — dùng khi **không** bật presigned URL.
- **`file_size.go`**: `FormatFileSize` (human-readable), `ValidateFileSizeFromHeader`.

✅ **Đã sửa (2026-09-04)** — `date_utils.go` trước đây dùng sai Go reference layout (`Format("2025")`, `Format("12")`, `Format("2025-12-12")`). Go's `time.Format` dùng **reference layout** (`2006-01-02 15:04:05`), không phải placeholder tự do như nhiều ngôn ngữ khác — chuỗi cũ không khớp bất kỳ token layout hợp lệ nào nên các hàm **luôn trả về đúng chuỗi literal đó bất kể ngày giờ thực tế** (`GetCurrentYear()` luôn `"2025"`, `GetCurrentMonth()` luôn `"12"`). Đây là nơi duy nhất dùng để tạo thư mục `year/month` khi lưu ảnh WordPress-resize (`image_resize.go`), nên mọi ảnh WP-upload trước bản sửa này đều nằm trong `wp-uploads/2025/12/` bất kể ngày tải lên thật. Đã đổi sang layout đúng — `Format("2006")`, `Format("01")`, `Format("2006-01-02")` — verify bằng test tạm thời cho kết quả đúng ngày hệ thống (`Year=2026 Month=09 Date=2026-09-04`). Luồng S3 chính không bị ảnh hưởng (`GenerateS3Key` vốn đã dùng `time.Now().Format("20060102-150405")` đúng chuẩn từ đầu).

⚠️ **Lưu ý dữ liệu cũ**: ảnh WP-upload lưu trước bản sửa này vẫn nằm ở `wp-uploads/2025/12/` — thư mục đó không tự di chuyển, cần migrate thủ công nếu muốn dồn về đúng cấu trúc theo tháng thật.

## `internal/metrics` — In-memory metrics

**File**: [metrics.go](../internal/metrics/metrics.go)

Singleton (`sync.Once`) đếm số request/lỗi theo endpoint (tối đa 1000 endpoint để tránh unbounded growth — dùng `r.URL.Path` làm key, nên **path có ID động, vd `/api/v1/upload/complete`, là cố định** nhưng nếu sau này thêm route kiểu `/files/:id` sẽ nổ số lượng key), latency trung bình (giữ tối đa 100 sample gần nhất mỗi endpoint, dùng copy-on-write slice để tránh race). Toàn bộ **mất khi restart** — không phải giải pháp metrics lâu dài, chỉ phù hợp quan sát nhanh qua `/api/v1/metrics`. Nếu cần observability thật cho production, cân nhắc thay bằng Prometheus client.

## `internal/logging` — File logging

**File**: [file.go](../internal/logging/file.go)

`SetupFromEnv()` ghi đè `log.SetOutput` thành `io.MultiWriter(stdout, dailyFile)` — áp dụng **toàn cục** cho package `log` chuẩn, nghĩa là mọi `log.Printf` trong toàn bộ codebase (kể cả trong service/handler) tự động được ghi file mà không cần truyền logger qua tham số. File xoay theo ngày (`app-YYYY-MM-DD.log`), tự tạo file mới khi qua ngày mới (`ensureFileLocked` so sánh `curDate`). Không tự xoá log cũ — cần cron/logrotate riêng nếu muốn giới hạn dung lượng `storage/logs/` lâu dài.

---

## `web/` — Frontend (React + Vite + TypeScript)

**Cấu trúc** ([web/src/](../web/src/)):
```
main.tsx              # entry point, mount App vào DOM
App.tsx                # component gốc: quản lý state upload flow, chọn mode (S3/WordPress)
lib/
  api.ts                # toàn bộ giao tiếp với backend (fetch/XHR), KHÔNG có component nào gọi fetch trực tiếp
  imageTransform.ts     # xử lý crop/zoom ảnh trước khi upload (client-side, canvas-based)
  previewImage.ts        # tạo/revoke object URL cho preview, tránh memory leak
components/
  Dropzone.tsx            # kéo-thả / chọn file
  PreviewPanel.tsx        # danh sách file đang chờ/ đang upload, hiển thị progress
  ZoomViewport.tsx + PreviewImage.tsx   # xem trước ảnh có zoom/pan
  ImageLightbox.tsx        # phóng to ảnh full-screen
  ResultPanel.tsx          # hiển thị kết quả upload (URL, size...)
  StepTimeline.tsx          # timeline các bước upload (init → chunk → complete)
  ProgressRing.tsx           # vòng tròn progress cho từng file
  ConfirmDialog.tsx           # dialog xác nhận (huỷ upload, xoá...)
  Onboarding.tsx                # hướng dẫn lần đầu dùng (lưu trạng thái đã xem qua localStorage)
```

### `lib/api.ts` — lớp giao tiếp API (đáng chú ý nhất để hiểu integration)

- Đọc `VITE_API_BASE_URL` và `VITE_API_KEY` từ biến môi trường build-time (Vite `import.meta.env`) — **API key nhúng vào bundle frontend nếu set**, nghĩa là **không an toàn nếu bundle public** (ai cũng đọc được key qua DevTools). Chỉ nên dùng `VITE_API_KEY` khi frontend + backend cùng một bên tin cậy triển khai, không phải giải pháp bảo mật cho public SPA — CSRF + CORS mới là lớp bảo vệ chính cho public UI.
- Tự đọc `csrf_token` cookie và đính vào header `X-CSRF-Token` cho mọi request POST — khớp với `CSRFProtection` middleware backend (double-submit pattern).
- Logic chọn **single-shot vs chunked** nằm ở client (`uploadFile()`): nếu `file.size > limits.maxSize` (lấy từ `/api/v1/health`) thì tự chuyển sang `uploadChunked()`. Nghĩa là **frontend phải luôn gọi `fetchHealth()` trước để biết ngưỡng thật của server** — hard-code sai ngưỡng ở client sẽ khiến upload bị từ chối ở backend dù UI tưởng đang dùng đúng luồng.
- `uploadChunked()` tự động `abortChunkUpload()` khi có lỗi giữa chừng (dọn session phía server), và retry từng chunk tối đa 3 lần với backoff tuyến tính (400ms × attempt) trước khi coi là lỗi.
- Message lỗi tiếng Việt được viết cứng trong file này (`messageFromBody`, `validateClientFile`) — không tách riêng i18n layer.

### Build & serve

`npm run build` xuất ra `web/dist` (Vite) — Go server (`spaFileServer` trong `container.go`) serve trực tiếp thư mục này, không cần Node.js runtime lúc production (chỉ cần lúc build). Dev mode dùng Vite dev server riêng (`npm run dev`, cổng 5173) với proxy `/api` trỏ về Go server cổng 8080 — xem cấu hình cụ thể trong [web/vite.config.ts](../web/vite.config.ts) nếu cần đổi cổng.

---

## Gợi ý khi thêm tính năng mới

- **Thêm loại storage khác ngoài S3** (GCS, Azure Blob...): thêm implementation mới cho interface `repository.S3Repository` (đổi tên interface nếu cần tổng quát hơn), đăng ký thêm case trong `RepositoryFactory.CreateRepository`. Service layer không cần đổi vì chỉ phụ thuộc interface.
- **Thêm endpoint mới**: thêm handler trong `internal/handlers`, đăng ký route trong `container.go` → `GetServerHandler()`. Nhớ cân nhắc middleware nào cần áp dụng (đặc biệt CSRF nếu là state-changing method, và rate limit nếu endpoint tốn tài nguyên).
- **Đổi giới hạn upload**: sửa `.env` (`UPLOAD_MAX_SIZE_MB`/`UPLOAD_ABSOLUTE_MAX_MB`) — không cần sửa code trừ khi muốn đổi số 64 (`maxChunksPerUpload`) đang hard-code riêng ở cả `config/builder.go` và `service/chunk_upload.go`.
- **Cần chunk session sống sót qua restart / nhiều instance**: phải viết lại `ChunkUploadService` để dùng backend chia sẻ (Redis/DB) thay vì `map` in-memory — đây là thay đổi kiến trúc, không phải config.

## Admin panel + trang public "20 năm VAS"

Mở rộng lớn thêm vào codebase gốc (vốn chỉ là công cụ upload S3 generic) để phục vụ hội thi vẽ
tranh kỷ niệm 20 năm Hệ thống Trường Việt Mỹ — theo đúng convention layer sẵn có
(`handlers → service → repository → database/models`), không phá vỡ luồng upload cũ.

**Domain mới** (`internal/models/`): `school` (5 cơ sở, gộp 3 khu vực trưng bày `saigon`/`cantho`/`vungtau`), `grade_level` (12 khối, 2 cấp `primary` 1-5 / `secondary` 6-12), `student`, `artwork` (bảng trung tâm, denormalize `school_id`/`grade_level_id` từ student để query dashboard nhanh), `award` + `artwork_award` (N-N, dù UI hiện tại chỉ cần 1 giải/tác phẩm), `reaction` (6 loại ẩn danh), `comment` (ẩn danh, tự nhập tên hiển thị), `artwork_view` (chống đếm trùng view 24h), `admin_user` + `session` (Google OAuth).

**Auth admin** (`internal/auth/`, `internal/middleware/admin_auth.go`): Google OAuth (`golang.org/x/oauth2`) + session lưu MySQL (không in-memory, sống sót qua restart) — khác hẳn `API_KEY` middleware (shared-secret) dùng cho `/api/v1/upload*`. `AdminAuthMiddleware` bọc toàn bộ `/api/v1/admin/*`, trả 401 JSON (không redirect — FE tự điều hướng qua React Router). Route `/auth/google/login|callback` là browser-redirect flow thật, mount ngoài `/api/` để không dính CORS/API-key middleware. Server khởi động bình thường dù thiếu `GOOGLE_CLIENT_ID/SECRET` — endpoint OAuth tự trả 503 rõ ràng.

**ArtworkService** (`internal/service/artwork_service.go`) tái dùng nguyên bản `UploadService.UploadImage` cho bulk-upload — thiết kế 2 bước tách biệt: bước 1 (`BulkUploadToS3`) chỉ đẩy ảnh lên S3 song song (giới hạn concurrency 5), CHƯA ghi bảng `artworks`; bước 2 (`CreateArtworkFromUpload`) tạo `students`+`artworks` trong 1 transaction sau khi admin điền metadata. Tách 2 bước để tránh phải cho phép NULL trên schema `artworks`.

**Public API ẩn danh** (`internal/handlers/public_handler.go`, mount `/api/v1/public/*`): reaction/comment/view không cần session — định danh qua `visitor_token` (UUID sinh ở FE, lưu `localStorage`, KHÔNG phải xác thực danh tính thật). Comment content/display_name escape qua `html.EscapeString` chống XSS. Write endpoint (POST/DELETE reaction, POST comment) bọc thêm `middleware.RateLimiter` riêng 20 req/phút/IP (nghiêm hơn rate limit chung toàn API) — implement bằng 1 `http.HandlerFunc` kiểm tra `r.Method` rồi route qua limiter hay không, KHÔNG dùng 2 `ServeMux` chồng lên cùng pattern (Go panic nếu đăng ký trùng pattern trên 2 mux áp cùng 1 path).

**Routing dùng Go 1.22+ ServeMux pattern** (`GET /path/{id}`, `r.PathValue("id")`) thay vì tự parse chuỗi path hay router thư viện ngoài — xem `handlers.parsePathID` helper dùng chung.

**Frontend** (`web/src/`): React Router thêm vào SPA vốn đơn trang — `/` giữ nguyên UploadTool cũ (di chuyển logic sang `pages/UploadTool.tsx`, không đổi), `/admin/*` (Google OAuth login, sidebar/header kiểu va-workspace, dashboard recharts, quản lý tác phẩm/giải thưởng), `/trien-lam` (trang public 5 section: Hero, cổng chọn khối, gallery tiêu biểu, phòng triển lãm theo khối, billboard vinh danh — `clip-path` CSS theo hạng giải). Toast có âm thanh tổng hợp bằng Web Audio API (`lib/sound.ts`, không dùng file audio). `index.css` gốc khoá `overflow:hidden` cho UploadTool (1 màn hình cố định) — `hooks/useScrollableBody.ts` mở khoá riêng cho các trang mới cần cuộn, không đụng UploadTool.

## Liên quan

- [ARCHITECTURE.md](./ARCHITECTURE.md) — tổng quan luồng request và middleware chain
- [API.md](./API.md) — tham chiếu endpoint
- [DEPLOYMENT.md](./DEPLOYMENT.md) — deploy VPS

# Architecture

## Tổng quan

S3 Upload Tool là một HTTP API service viết bằng Go, phục vụ 3 nhóm chức năng chính:

1. **Upload file trực tiếp lên S3** — single-shot (≤ `UPLOAD_MAX_SIZE_MB`, mặc định 20MB) hoặc chunked (đến `UPLOAD_ABSOLUTE_MAX_MB`, mặc định 200MB).
2. **Upload có transaction với database** — ghi nhận trạng thái upload (pending/completed/failed) vào PostgreSQL.
3. **WordPress-style local resize** — nhận ảnh, resize ra nhiều kích thước, tối ưu (JPEG/PNG/WebP), lưu local và serve qua HTTP.

Server phục vụ đồng thời:
- API JSON tại `/api/v1/*`
- SPA React (Lumen UI) tại `/` từ `web/dist` (build Vite) — nếu chưa build thì `/` trả JSON info endpoint thay vì lỗi.
- Static file WordPress-style uploads tại `/wp-content/uploads/*` (không cho list thư mục).

## Sơ đồ thành phần

```
cmd/server/main.go
   │  load .env → setup file logging → build Container → start http.Server → graceful shutdown
   ▼
internal/container/container.go   (composition root / DI)
   │
   ├── internal/config            (env → typed Config, validate production constraints)
   ├── AWS SDK v2 session          (credential chain: env → IAM role → ~/.aws → EC2 metadata)
   │      └── auto-resolve bucket region nếu khác AWS_REGION cấu hình
   ├── internal/database           (*sql.DB wrapper, optional — chỉ khởi tạo nếu DATABASE_ENABLED)
   │
   ├── internal/repository
   │      ├── S3Repository         Upload / GeneratePresignedURL / CheckConnectivity (qua s3manager.Uploader)
   │      └── UploadRepository     CRUD upload_records (Postgres), dùng transaction
   │
   ├── internal/service
   │      ├── UploadService        single-shot upload (temp file → validate → S3 → URL/presigned URL)
   │      ├── ChunkUploadService   init/save-chunk/complete/abort, in-memory session map + TTL cleanup
   │      └── ImageResizeService + ImageOptimizer   WordPress resize pipeline (local disk)
   │
   ├── internal/handlers
   │      ├── APIHandler           /api/v1/upload*, /api/v1/health
   │      ├── WPHandler            /api/v1/wp-upload
   │      └── MetricsHandler       /api/v1/metrics
   │
   └── internal/middleware
          RequestID → Logging → Metrics → CSRF → RateLimit → ConcurrencyLimit → CORS → APIKeyAuth
```

`Container` (internal/container/container.go) là composition root duy nhất: khởi tạo mọi dependency, wire handler/middleware, expose `GetServerHandler()` cho `main.go` và `Shutdown()` để dọn dẹp (dừng rate limiter goroutines, đóng DB, xoá chunk sessions còn treo).

## Luồng request

### 1. Upload trực tiếp — `POST /api/v1/upload`

```
Client (multipart/form-data)
  → RequestID/Logging/Metrics/CSRF/RateLimit/Concurrency/CORS/APIKey middleware
  → APIHandler.HandleUpload
      → processUploadRequest: ParseMultipartForm, lấy file, ValidateUploadFile (type/size)
      → UploadService.UploadImage
          → ghi ra temp file trong UPLOAD dir (io.LimitReader maxSize+1 để chặn vượt hạn mức)
          → copy có timeout scale theo size (~2s/MB, tối thiểu 60s, tối đa 5m)
          → S3Repository.Upload (multipart PutObject qua s3manager, part 8MB, concurrency 4)
          → nếu USE_PRESIGNED_URL: GeneratePresignedURL, ngược lại trả object URL trực tiếp
      ← xoá temp file (defer), trả UploadResponse{URL, Key}
  ← JSON {success, data: {url, key, size, name}}
```

### 2. Upload chunked — file lớn (đến 200MB)

```
POST /api/v1/upload/init     → tạo ChunkSession (uuid), tính totalChunks, mkdir uploads/chunks/<id>
POST /api/v1/upload/chunk    → SaveChunk: ghi part_%06d theo index, validate kích thước từng phần
POST /api/v1/upload/complete → Complete:
                                  - kiểm tra đủ mọi chunk
                                  - ghép các phần vào 1 file tạm (theo đúng thứ tự index)
                                  - ValidateFileContent (magic byte, không chỉ tin đuôi file)
                                  - Upload lên S3, build URL/presigned URL
                                  - dọn session + xoá thư mục chunks/<id>
POST /api/v1/upload/abort    → huỷ session, xoá chunk đã nhận
```

Session lưu **in-memory** (không phải DB) — TTL 45 phút, dọn định kỳ mỗi 5 phút (`cleanupLoop`), giới hạn tối đa 64 session đồng thời và 64 chunk/upload. ⚠️ Vì lưu in-memory, session **mất khi restart server** hoặc khi chạy nhiều instance sau load balancer không sticky session.

### 3. Upload với transaction — `POST /api/v1/upload-transaction`

Giống upload trực tiếp nhưng bọc trong DB transaction:
```
BEGIN → INSERT upload_records (status=pending)
      → copy file, validate
      → S3 upload
      → UPDATE status=completed (hoặc failed nếu lỗi ở bất kỳ bước nào) + rollback nếu panic/error
COMMIT
```
Yêu cầu `DATABASE_ENABLED=true` + `DATABASE_URL` hợp lệ, nếu không trả `503 DATABASE_DISABLED`.

### 4. WordPress resize — `POST /api/v1/wp-upload`

```
WPHandler.HandleWPUpload
  → nhận ảnh, lưu gốc vào WORDPRESS_UPLOADS_DIR
  → ImageResizeService: resize theo từng size trong WORDPRESS_IMAGE_SIZES (vd thumbnail:150x150)
  → ImageOptimizer (nếu bật): nén JPEG/PNG theo quality, tuỳ chọn xuất thêm WebP
  → trả URL truy cập qua WORDPRESS_BASE_URL + /wp-content/uploads/...
```
Đây là pipeline **local disk**, không đụng tới S3 — dùng khi cần tương thích kiểu WordPress media library.

### 5. Health check — `GET /api/v1/health`

Kiểm tra: DB ping (nếu bật), S3 `HeadBucket` connectivity, disk (stub, chưa implement đầy đủ trên Windows). Trả `503 NOT_READY` nếu bất kỳ dependency nào lỗi — dùng cho Docker `HEALTHCHECK` / load balancer readiness probe.

## Middleware chain (thứ tự áp dụng, ngoài cùng trước)

```
RequestID → Logging → Metrics → CSRF (nếu bật) → RateLimit (nếu bật) → ConcurrencyLimit (nếu bật)
   └─ mux "/api/*" → CORS → APIKeyAuth (nếu bật + có key) → route handler
```
- **RequestID**: gắn UUID vào context, dùng để trace log.
- **CSRF**: bật mặc định cho SPA same-origin; cần `CSRF_SECURE_COOKIE=true` khi chạy sau HTTPS.
- **RateLimit**: theo IP, cấu hình `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW_MINUTES`; endpoint `/api/v1/metrics` có rate limiter riêng nghiêm ngặt hơn (10 req/window, hard-coded).
- **ConcurrencyLimit**: semaphore giới hạn số upload đồng thời toàn server (`MAX_CONCURRENT_UPLOADS`).
- **CORS**: whitelist origin từ `CORS_ORIGINS` (comma-separated); **không được để `*` ở production** — `config` package chặn khởi động nếu vi phạm.
- **APIKeyAuth**: header-based, chỉ bật khi `API_REQUIRE_KEY=true` và có `API_KEY`.

## Configuration

Toàn bộ cấu hình đọc từ biến môi trường qua `internal/config` (builder pattern: `ConfigBuilder.BuildFromEnv()`), validate ràng buộc production trong `internal/config/errors.go`. Xem `.env.example` ở root làm nguồn sự thật cho danh sách biến.

Điểm quan trọng khi deploy:
- `APP_ENV=production` bắt buộc phải có `API_KEY` + `CORS_ORIGINS` cụ thể (không phải `*`), nếu không server từ chối khởi động.
- AWS credentials nên dùng IAM role (EC2/ECS/EKS) thay vì static keys trong `.env` — `container.go` có cảnh báo (`warnPlaceholderAWSCredentials`) nếu phát hiện key/secret là giá trị placeholder.
- `S3_ENDPOINT` + `S3_FORCE_PATH_STYLE=true` dùng khi chạy với MinIO/LocalStack thay vì AWS thật.

## Data model (khi bật database)

Bảng `upload_records` (migration `internal/database/migrations/001_create_uploads_table.sql`), quản lý bởi `internal/repository/upload_repository.go`:

| field | ý nghĩa |
|---|---|
| id | PK |
| filename / original_name | tên file lưu / tên gốc client gửi |
| file_size, content_type | metadata |
| s3_key, s3_url | điền sau khi upload S3 thành công |
| status | pending → completed \| failed |
| error | lý do lỗi (nếu có) |
| created_at / updated_at | timestamps |

## Observability

- **Logging**: middleware ghi mọi request; file log daily rotation tại `LOG_DIR` (`internal/logging/file.go`), song song stdout.
- **Metrics**: `internal/metrics/metrics.go` đếm số upload thành công/thất bại + latency, expose tại `/api/v1/metrics` (rate-limited riêng).
- **Health**: `/api/v1/health` — dùng cho container orchestrator / uptime monitor.

## Giới hạn đã biết (cần lưu ý khi scale)

1. **Chunk session in-memory** → không chia sẻ giữa nhiều instance; cần sticky session hoặc chuyển sang backend chia sẻ (Redis) nếu scale ngang.
2. **Disk space check** (`checkDiskSpace` trong `api_handler.go`) chưa implement thật trên Windows và chỉ là stub trên Unix — không dùng làm cảnh báo hết dung lượng đáng tin cậy.
3. **Temp file trên local disk** trong lúc upload (kể cả chunk lẫn single-shot) — VPS cần đủ dung lượng trống tối thiểu bằng `UPLOAD_ABSOLUTE_MAX_MB` nhân với `MAX_CONCURRENT_UPLOADS` trong trường hợp xấu nhất.
4. Static/WP uploads phục vụ trực tiếp từ Go process (`http.FileServer`) — nếu traffic lớn nên để Nginx/CDN cache đằng trước thay vì để Go serve file tĩnh.

## Liên quan

- Xem [MODULES.md](./MODULES.md) để bóc tách chi tiết từng package/module, trách nhiệm, phụ thuộc, và các điểm cần lưu ý khi sửa đổi.
- Xem [DEPLOYMENT.md](./DEPLOYMENT.md) để deploy lên VPS (Docker hoặc systemd/binary trực tiếp).
- Xem [API.md](./API.md) cho tham chiếu đầy đủ từng endpoint.

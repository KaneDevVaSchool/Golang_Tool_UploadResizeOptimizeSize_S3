# API Reference

Base URL mặc định: `http://localhost:8080` (đổi theo `PORT` / domain thật khi deploy).

Mọi response đều theo format chung:

```json
// success
{ "success": true, "data": { ... } }

// error
{ "success": false, "error": { "code": "SOME_CODE", "message": "..." } }
```

## Xác thực & bảo mật

- **API Key** (nếu `API_REQUIRE_KEY=true`): gửi header `X-API-Key: <key>`. Không bao giờ chấp nhận key qua query string. `/api/v1/health` luôn miễn xác thực (cho load balancer/k8s probe).
- **CORS**: chỉ origin trong `CORS_ORIGINS` (comma-separated) được phép gọi từ browser.
- **CSRF**: bật mặc định cho same-origin SPA — nếu gọi API từ domain khác/non-browser client, xem cách lấy CSRF token trong `internal/middleware/csrf.go` hoặc tắt `CSRF_ENABLED` cho pure API use-case.
- **Rate limit**: mặc định 100 req/phút/IP (`RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW_MINUTES`); riêng `/api/v1/metrics` giới hạn cứng 10 req/phút.
- **Concurrency limit**: tối đa `MAX_CONCURRENT_UPLOADS` upload xử lý đồng thời toàn server; vượt quá sẽ chờ tối đa `CONCURRENCY_ACQUIRE_TIMEOUT_SECONDS` rồi trả lỗi.

---

## `POST /api/v1/upload`

Upload trực tiếp một file (≤ `UPLOAD_MAX_SIZE_MB`, mặc định 20MB).

**Request**: `multipart/form-data`, field file (tên field: xem `utils.GetFileFromRequest` — hỗ trợ `file`).

**Response 200**:
```json
{
  "success": true,
  "data": {
    "url": "https://bucket.s3.region.amazonaws.com/key...",
    "key": "images/2026/09/uuid-photo.jpg",
    "size": 123456,
    "name": "photo.jpg"
  }
}
```

**Lỗi thường gặp**:
| status | code | ý nghĩa |
|---|---|---|
| 400 | `INVALID_FORM` | multipart form không hợp lệ |
| 400 | `FILE_NOT_FOUND` | thiếu file trong request |
| 400 | `INVALID_FILE_TYPE` | định dạng không hỗ trợ |
| 400 | `FILE_TOO_LARGE` | vượt `UPLOAD_MAX_SIZE_MB` |
| 405 | `METHOD_NOT_ALLOWED` | không phải POST |
| 5xx | `UPLOAD_FAILED` | lỗi S3 / lỗi hệ thống |

---

## `POST /api/v1/upload-transaction`

Giống `/upload` nhưng ghi nhận vào database (yêu cầu `DATABASE_ENABLED=true`). Trả thêm `record` (trạng thái upload đã lưu).

**Response 200**:
```json
{
  "success": true,
  "data": {
    "url": "...",
    "key": "...",
    "size": 123456,
    "name": "photo.jpg",
    "record": {
      "id": 42,
      "filename": "photo.jpg",
      "original_name": "photo.jpg",
      "file_size": 123456,
      "content_type": "image/jpeg",
      "s3_key": "...",
      "s3_url": "...",
      "status": "completed",
      "created_at": "2026-09-04T10:00:00Z",
      "updated_at": "2026-09-04T10:00:01Z"
    }
  }
}
```

Nếu database chưa bật → `503 DATABASE_DISABLED`.

---

## Chunked upload (file lớn, đến `UPLOAD_ABSOLUTE_MAX_MB`, mặc định 200MB)

Quy trình: `init` → `chunk` (lặp lại cho từng phần) → `complete` (hoặc `abort` để huỷ giữa chừng).

### `POST /api/v1/upload/init`

**Request** (JSON):
```json
{ "filename": "video.mp4", "total_size": 157286400 }
```

**Response 200**:
```json
{
  "success": true,
  "data": {
    "upload_id": "b3f1...-uuid",
    "chunk_size": 20971520,
    "total_chunks": 8,
    "total_size": 157286400,
    "filename": "video.mp4"
  }
}
```

### `POST /api/v1/upload/chunk`

**Request**: `multipart/form-data` với fields:
- `upload_id` (string, từ bước init)
- `index` (int, 0-based)
- `chunk` hoặc `file` (binary phần dữ liệu, kích thước phải khớp `chunk_size` — trừ phần cuối cùng)

**Response 200**:
```json
{ "success": true, "data": { "upload_id": "...", "index": 0, "size": 20971520, "status": "received" } }
```

### `POST /api/v1/upload/complete`

**Request** (JSON): `{ "upload_id": "..." }`

Ghép toàn bộ chunk theo thứ tự, validate nội dung (magic byte), upload lên S3, trả kết quả giống `/api/v1/upload`:
```json
{ "success": true, "data": { "url": "...", "key": "...", "size": 157286400, "name": "video.mp4" } }
```

### `POST /api/v1/upload/abort`

**Request** (JSON): `{ "upload_id": "..." }` → xoá session + chunk đã nhận trên disk.
```json
{ "success": true, "data": { "status": "aborted" } }
```

**Lỗi đặc thù chunk upload**:
| status | code | ý nghĩa |
|---|---|---|
| 404 | `SESSION_NOT_FOUND` | upload_id sai / hết hạn (TTL 45 phút) |
| 409 | `SESSION_BUSY` | session đang trong lúc `complete` |
| 429 | `TOO_MANY_SESSIONS` | vượt quá 64 session đồng thời toàn server |
| 400 | `VALIDATION_ERROR` | thiếu field / chunk sai kích thước / thiếu chunk khi complete |

⚠️ Session lưu **in-memory** trong tiến trình — sẽ mất nếu restart server giữa chừng, và không chia sẻ được giữa nhiều instance nếu chạy nhiều bản sao sau load balancer (cần sticky session).

---

## `POST /api/v1/wp-upload`

Upload ảnh kiểu WordPress: lưu local, tự resize theo `WORDPRESS_IMAGE_SIZES`, tối ưu (JPEG/PNG/WebP nếu `IMAGE_OPTIMIZATION_ENABLED=true`). Chỉ nhận file ảnh, **không upload S3**.

**Request**: `multipart/form-data`, field file ảnh.

**Response 200**:
```json
{
  "success": true,
  "data": {
    "file": { "name": "photo.jpg", "type": "image/jpeg", "url": "http://host/wp-content/uploads/2026/09/photo.jpg", "size": 204800 },
    "sizes": [
      { "name": "thumbnail", "file": "photo-150x150.jpg", "width": 150, "height": 150, "url": "http://host/wp-content/uploads/2026/09/photo-150x150.jpg" },
      { "name": "medium", "file": "photo-300x300.jpg", "width": 300, "height": 300, "url": "..." }
    ]
  }
}
```

Ghi chú: nếu resize thất bại, endpoint vẫn trả `200` với ảnh gốc và `sizes: []` (không fail toàn bộ request).

File phục vụ tĩnh qua `GET /wp-content/uploads/<path>` (không cho list thư mục — truy cập thư mục trả 404).

---

## `GET /api/v1/health`

Không cần API key. Dùng cho readiness probe / Docker `HEALTHCHECK`.

**Response 200** (khoẻ mạnh) hoặc **503** (`status: "unavailable"`, nếu DB/S3 lỗi):
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "service": "s3-upload-api",
    "max_size": 20971520,
    "max_size_formatted": "20.00 MB",
    "absolute_max_size": 209715200,
    "absolute_max_size_formatted": "200.00 MB",
    "chunk_upload": true,
    "request_id": "...",
    "checks": {
      "disk": { "status": "ok", "note": "..." },
      "database": { "status": "ok", "open_connections": 2, "in_use": 0, "idle": 2, "wait_count": 0 },
      "s3_service": { "status": "ok" }
    }
  }
}
```

## `GET /api/v1/metrics`

Số liệu upload (tổng số, thành công/thất bại, latency). Rate-limited nghiêm ngặt (10 req/phút) để tránh lạm dụng.

## `GET /`

- Nếu `web/dist` đã build (React SPA) → serve `index.html` + static assets, fallback về `index.html` cho client-side routing.
- Nếu chưa build → trả JSON:
```json
{ "service": "s3-upload-api", "version": "1.0", "endpoints": ["/api/v1/upload", "..."] }
```

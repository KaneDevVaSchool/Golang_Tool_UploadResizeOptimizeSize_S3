# Bóc tách module

Trách nhiệm, phụ thuộc và lưu ý khi sửa từng package. Đọc
[ARCHITECTURE.md](./ARCHITECTURE.md) trước nếu cần bức tranh tổng thể.

Đối chiếu code trên nhánh `feature/artwork-contest-system`, rà lại ngày **2026-09-07**.
Quy mô: **103 file Go** (~13.600 dòng trong `internal/`, đã tính cả test) và **82 file
TypeScript/TSX**.

## Sơ đồ phụ thuộc

```text
cmd/server, cmd/migrate
        │
        ▼
internal/container ──▶ config, database, handlers, metrics, middleware, repository, service, auth
        │
        ▼
internal/handlers  ──▶ auth, database, middleware, models, repository, service, utils
        │
        ▼
internal/service   ──▶ database, models, repository, utils
        │
        ▼
internal/repository ──▶ database, models   (+ AWS SDK v2)
        │
        ▼
internal/{config, database, models, utils, metrics, logging, middleware}   ← leaf, không phụ thuộc lẫn nhau
```

Quy tắc **một chiều**: `handlers → service → repository → database/models`. `middleware` độc
lập hoàn toàn (chỉ dùng `net/http` + `uuid` + `semaphore`), được `container` lắp vào chuỗi
chứ tự nó không biết gì về service/handler.

---

## `cmd/server` — Điểm khởi động

**File**: [main.go](../cmd/server/main.go) · 71 dòng

Nạp `.env` → thiết lập log → `container.NewContainer()` → chạy `http.Server` → chờ
`SIGINT`/`SIGTERM` → tắt an toàn.

⚠️ **Thứ tự tắt máy quan trọng, không được đảo**: drain request HTTP trước
(`server.Shutdown`), rồi mới đóng container (DB, rate limiter, bộ đếm chống quét). Đảo ngược sẽ
làm request đang xử lý mất kết nối DB/S3 giữa chừng.

Không chứa nghiệp vụ. Cần thêm hành vi lúc khởi động/tắt thì sửa `container.go`, không sửa
file này.

## `cmd/migrate` — Chạy migration bằng tay

**File**: [main.go](../cmd/migrate/main.go) · 355 dòng

Chạy `internal/database/migrations/*.sql` qua cờ `-up`. Song song với việc container tự chạy
migration lúc khởi động — cả hai dùng chung bảng `schema_migrations` nhưng độc lập vòng đời.

Không có `-down`/rollback tự động.

## `cmd/seed` — Xoá sạch + nạp dữ liệu demo

**File**: [main.go](../cmd/seed/main.go), [data.go](../cmd/seed/data.go)

Dựng lại thủ công một phần dây chuyền repository/service của `container.go` (không kéo
theo handler/middleware) để: xoá sạch dữ liệu nghiệp vụ (`artworks`, `students`, `awards`,
`topic_categories`, `artwork_reactions`, `artwork_comments`, `artwork_views` — **không** đụng
`admin_users`/`admin_sessions`/`grade_levels`), xoá object S3 mà các `artworks` bị xoá đang
tham chiếu (thu thập `s3_key` + key trong `variants` **trước** khi xoá DB row, không quét
bucket), rồi nạp lại danh mục (schools/awards/topic_categories) và tạo tác phẩm demo bằng
cách upload từng ảnh trong một thư mục nguồn (`--images`, mặc định ảnh theme cục bộ) lên S3
qua `ArtworkService.CreateArtworkFromUpload` — không lặp lại ảnh, mỗi ảnh đúng 1 tác phẩm.

⚠️ Ghi đè luôn bảng `schools` bằng 16 cơ sở thật của Hệ thống Việt Mỹ, thay cho 5 cơ sở cũ
trong migration 004 (đã lỗi thời) — xem lý do và danh sách đầy đủ ở
[detail_design/01-database.md](./detail_design/01-database.md). Việc này chỉ an toàn vì
luôn chạy **sau** bước xoá `artworks`/`students`, lúc đó không còn FK nào RESTRICT chặn
`DELETE FROM schools`.

Yêu cầu gõ đúng `XOA` để xác nhận (bỏ qua bằng `--yes` khi chạy không tương tác). Không thể
hoàn tác — chỉ dùng cho môi trường demo/dev, không chạy trên DB đang có dữ liệu hội thi
thật.

---

## `internal/config` — Cấu hình

**Files**: [config.go](../internal/config/config.go) (138 dòng),
[builder.go](../internal/config/builder.go) (589), [errors.go](../internal/config/errors.go)

`config.go` định nghĩa struct `Config` và 11 struct con — **nguồn sự thật duy nhất** cho mọi
giá trị cấu hình.

`builder.go` đọc biến môi trường, áp giá trị mặc định, và **kiểm tra ràng buộc production**.

Điểm cần biết khi sửa:

| Điểm | Chi tiết |
|---|---|
| Nạp `.env` một lần | Qua `sync.Once` — không có nạp lại nóng |
| Chặn khởi động ở production | Thiếu API key hoặc `CORS_ORIGINS=*` → lỗi ngay |
| Danh sách admin mặc định | `defaultAllowedAdminEmails` (dòng 20) — 9 tài khoản; để trống `.env` **không** nghĩa là mở cửa |
| Tự điều chỉnh giới hạn | `AbsoluteMaxSize` được nâng lên bằng `MaxSize` nếu cấu hình đặt nhỏ hơn |
| Cookie secure suy từ `APP_ENV` | Dùng chung cho cả CSRF lẫn session |

⚠️ Timeout đọc/ghi mặc định trong code (15s/30s) **thấp hơn** giá trị gợi ý trong
`.env.example` (180s). Nên khai báo tường minh ở production.

## `internal/database` — Kết nối MySQL

**Files**: [database.go](../internal/database/database.go) (150),
[migrate.go](../internal/database/migrate.go) (160), `migrations/*.sql` (đánh số tăng dần từ
`001`, xem thư mục để biết số lượng hiện tại)

Lớp bọc mỏng quanh `*sql.DB`/`*sql.Tx`, driver `go-sql-driver/mysql`. ⚠️ Trường `Driver`
đặt tên chung chung nhưng **chỉ hỗ trợ MySQL** (đã chuyển từ PostgreSQL trước đây).

`NewDB()` ping ngay lúc khởi tạo, timeout 5 giây → **DB chết là server không khởi động
được** (fail-fast, không thử lại). Với systemd `Restart=on-failure` thì service tự thử lại
mỗi 5 giây, hữu ích khi MySQL khởi động chậm hơn ứng dụng sau reboot.

`NormalizeMySQLDSN()` tự thêm `parseTime=true`, `charset=utf8mb4`, `multiStatements=true`
nếu thiếu, **không ghi đè** giá trị đã đặt. Thiếu `parseTime` thì mọi cột `DATETIME` scan
vào `time.Time` sẽ lỗi.

`RunMigrations()` chạy toàn bộ file theo thứ tự, idempotent qua `schema_migrations`.

## `internal/models` — Cấu trúc dữ liệu

Thuần dữ liệu — không nghiệp vụ, không I/O. Một file một nhóm domain (`artwork.go`,
`award.go`, `topic_category.go`...), có file khai nhiều struct liên quan (model + request/filter
đi kèm).

Quy ước thẻ: `db:"cột"` cho SQL, `json:"tên"` cho API. Vài ánh xạ **không hiển nhiên**:

| Struct | Điểm cần biết |
|---|---|
| `Artwork` | `S3URL` ra JSON là **`image_url`**; `S3Key`, `UploadID`, `CreatedBy` ẩn (`json:"-"`) |
| `ArtworkReaction` | `VisitorToken`, `IPAddress` ẩn — không bao giờ lộ ra ngoài |
| `ArtworkWithMeta` | Nhúng `Artwork` + dữ liệu enrich từ 6 bảng |
| `ArtworkFilter` | Dùng chung admin/public; public luôn ép `IsPublished=true` |

`models/reaction.go` khai báo 6 loại cảm xúc + `ValidReactionTypes` để handler kiểm tra
trước khi chạm DB.

## `internal/utils` — Tiện ích dùng chung

10 file. Leaf package, không phụ thuộc gì trong dự án.

| File | Nội dung | Có test |
|---|---|---|
| `filename.go` | `SanitizeFilename` — chống path traversal | ✅ |
| `file_utils.go` | Phân loại đuôi file, `GenerateS3Key` | |
| `s3_url.go` | `BuildS3ObjectURL` — 3 dạng hạ tầng, escape từng đoạn | ✅ |
| `file_size.go` | Kiểm tra và định dạng dung lượng | ✅ |
| `content_validation.go` | `ValidateFileContent` — magic byte | |
| `image_dimension.go` | `DecodeImageDimensions` | |
| `multipart.go` | Trích file từ request | |
| `date_utils.go` | Định dạng ngày | |

`SanitizeFilename` chạy nhiều bước: lấy `filepath.Base` → bỏ `..`, `/`, `\` → bỏ ký tự điều
khiển và `< > : " | ? *` → cắt còn 255 ký tự (giữ đuôi file).

## `internal/logging` — Ghi log ra file

**File**: [file.go](../internal/logging/file.go)

Ghi đồng thời stdout và file theo ngày `app-YYYY-MM-DD.log`. Đọc `LOG_DIR`/`LOG_PREFIX`
trực tiếp qua `os.Getenv` (không qua `config`).

⚠️ **Không tự xoá log cũ** — cần `logrotate`, xem
[deploys/03-operations.md](./deploys/03-operations.md).

## `internal/metrics` — Chỉ số

Middleware đếm request, độ trễ, phân bố mã trạng thái. Phục vụ `/api/v1/metrics`. Lưu trong
bộ nhớ, mất khi khởi động lại.

---

## `internal/middleware` — Chuỗi xử lý HTTP

Mỗi file một mối quan tâm, ghép lại trong `container.go`.

| File | Dòng | Việc | Đã nối vào chuỗi |
|---|---|---|---|
| `request_id.go` | 36 | Sinh/nhận `X-Request-ID` | ✅ |
| `logging.go` | 36 | Log request kèm ID | ✅ |
| `apikey.go` | 44 | Xác thực `X-API-Key`, so sánh hằng thời gian | ✅ |
| `admin_auth.go` | 55 | Xác thực session, gắn user vào context | ✅ |
| `bodylimit.go` | 61 | Trần kích thước body theo đường dẫn | ✅ |
| `concurrency.go` | 68 | Semaphore toàn server | ✅ |
| `cors.go` | 97 | Danh sách origin cho phép | ✅ |
| `security_headers.go` | 143 | CSP, Permissions-Policy, COOP/CORP, HSTS | ✅ |
| `csrf.go` | 161 | Double-submit cookie | ✅ |
| `clientip.go` | 171 | Xác định IP client qua proxy tin cậy | ✅ (dùng chung) |
| `ratelimit.go` | 193 | Giới hạn tần suất theo IP | ✅ |
| `compress.go` | 209 | Nén gzip phản hồi | ✅ |
| `botguard.go` | 356 | Chặn công cụ tải trọn site | ✅ |

`compress.go` (`GzipMiddleware`) nén phản hồi `text/*` và JSON trên 1KB, bỏ qua các định
dạng đã nén sẵn (ảnh, video). Đặt **trong cùng** chuỗi — sát mux nhất — vì nó cần thấy
`Content-Type` do handler đặt.

Ba điểm về `apikey.go` khi sửa:

- Chỉ chấp nhận key qua **header**, không bao giờ qua query string (query bị ghi vào log).
- Dùng `subtle.ConstantTimeCompare` chống tấn công đo thời gian.
- ⚠️ Chỉ miễn trừ `/api/v1/health`. Nghĩa là bật API key sẽ chặn **cả** `/api/v1/public/*` —
  rủi ro R1 trong [plan/03-risks.md](./plan/03-risks.md).

`clientip.go` là **nền móng của mọi giới hạn theo IP** trong hệ thống. `GetClientIP()` chỉ
đọc `X-Forwarded-For`/`X-Real-IP` khi chặng kết nối trực tiếp nằm trong `TRUSTED_PROXIES`;
ngược lại dùng thẳng `RemoteAddr`. `ClientIPKey()` gom IPv6 về khối `/64` trước khi làm khoá
đếm. ⚠️ Sửa file này là chạm vào rate limit của toàn hệ thống — đọc
[detail_design/05-auth-security.md §6](./detail_design/05-auth-security.md) trước.

`botguard.go` chặn công cụ tải trọn site nhưng **miễn trừ bot tìm kiếm và bot mạng xã hội**.
⚠️ Khi thêm chuỗi vào `scraperAgentMarkers`, tuyệt đối không thêm `"bot"`, `"crawler"` hay
`"spider"` chung chung — Googlebot, bingbot, `facebookexternalhit` đều chứa các chuỗi đó, và
thêm vào là xoá sổ toàn bộ SEO. Có test khoá lại điều này (`botguard_test.go`).

## `internal/auth` — Google OAuth và session

**Files**: [oauth.go](../internal/auth/oauth.go) (82), [session.go](../internal/auth/session.go) (115)

`oauth.go`: dựng `*oauth2.Config`, sinh/đặt/đọc cookie state (TTL 5 phút, dùng một lần).
`IsGoogleOAuthConfigured()` cho phép handler trả 503 rõ ràng thay vì gọi Google với thông
tin rỗng.

`session.go`: `SessionManager` tạo/kiểm tra/xoá session **lưu DB**. Token 32 byte
`crypto/rand`.

⚠️ Cả hai loại cookie dùng `SameSite=Lax`, **không** phải `Strict` — callback OAuth là điều
hướng xuyên site, `Strict` sẽ không gửi cookie và người dùng vừa đăng nhập lại thấy chưa
đăng nhập.

---

## `internal/repository` — Truy cập dữ liệu

16 file. Không biết gì về HTTP.

| File | Dòng | Ghi chú |
|---|---|---|
| `s3_repository.go` | 121 | Upload, tải object về, presigned URL, kiểm tra kết nối |
| `artwork_repository.go` | 342 | CRUD + WHERE động (gồm lọc `region`) + phân trang + `SetFeaturedBatch` + `DeleteBatch` |
| `artwork_download_repository.go` | 43 | Ghi nhật ký lượt tải ảnh (bảng `artwork_downloads`) |
| `award_repository.go` | 252 | Giải thưởng + gán/gỡ N:N + lọc theo `grade_level_id` |
| `topic_category_repository.go` | 137 | Nhóm chủ đề sáng tạo — CRUD, cùng mẫu `award_repository.go` |
| `dashboard_repository.go` | 491 | Truy vấn tổng hợp + xu hướng theo khoảng ngày, độ phủ trường, chỉ số vận hành |
| `comment_repository.go` | 163 | Gồm `DeleteOwned` kiểm tra quyền sở hữu |
| `reaction_repository.go` | 114 | `INSERT IGNORE` → idempotent |
| `admin_user_repository.go` | 120 | Tìm/tạo theo `google_sub` |
| `session_repository.go` | 92 | Gồm `DeleteExpired` |
| `artwork_view_repository.go` | 48 | `RecordView` — mỗi lần gọi ghi 1 dòng, không chống trùng |
| `school_repository.go` · `student_repository.go` · `grade_level_repository.go` | | Danh mục và học sinh |
| `errors.go` | | Lỗi dùng chung tầng repository |
| `factory.go` | 31 | **Chỉ** cho S3/upload — không dùng cho miền nghiệp vụ |

Điểm cần biết khi sửa:

**Hai kiểu chữ ký.** Hàm cần tham gia transaction của service nhận `tx *database.Tx` làm
tham số (`ArtworkRepository.Create`, `StudentRepository.Create`, `UploadRepository.CreateUpload`);
các hàm khác dùng `r.db` trực tiếp. Repository **không** tự quyết định ranh giới transaction —
đó là việc của service.

**Ngoại lệ**: `ArtworkViewRepository.RecordView` tự mở transaction, vì nó là thao tác nguyên
tử độc lập (insert + tăng đếm) không nằm trong luồng nghiệp vụ lớn hơn.

**Factory chỉ dành cho S3.** `RepositoryFactory` chỉ tạo `S3Repository`. Mọi repository miền
nghiệp vụ dùng constructor trực tiếp (`NewArtworkRepository(db)`). Đừng mở rộng factory cho
chúng — quy ước này được giữ nhất quán trong `container.go`.

**Truy vấn luôn tham số hoá.** Kể cả WHERE động cũng ghép chuỗi điều kiện rồi truyền giá trị
qua `?` — không bao giờ nối giá trị vào SQL.

## `internal/service` — Nghiệp vụ

12 file. Tầng duy nhất được mở/commit/rollback transaction.

| File | Dòng | Ghi chú |
|---|---|---|
| `upload_service.go` | 251 | Upload đơn, trích S3 key từ URL, xoá object |
| `artwork_service.go` | 754 | Bulk upload, tạo/sửa/xoá (xoá kèm object S3), xoá hàng loạt, enrich, `SetFeaturedBatch`, `ListPublishedForSitemap`, `LogDownload` |
| `artwork_download_watermark.go` | 180 | Đóng mốc VAS vào ảnh trước khi trả cho khách tải |
| `award_service.go` | 73 | CRUD giải |
| `topic_category_service.go` | 80 | CRUD nhóm chủ đề sáng tạo, cùng mẫu `award_service.go` |
| `dashboard_service.go` | 172 | Gộp số liệu thành 1 DTO, giải nghĩa khoảng ngày (có test ở `dashboard_service_test.go`) |
| `image_variants.go` | 335 | Sinh biến thể thumb/medium/large × WebP/JPEG |
| `constants.go` · `errors.go` · `factory.go` | | Hằng số, lỗi, factory (chỉ upload) |

**Về `image_variants.go`**: encode WebP lossy qua `gen2brain/webp` (WASM + purego,
**không cần cgo** nên khâu build tĩnh giữ nguyên). Sinh song song theo số CPU, và **bỏ qua**
cỡ nào lớn hơn ảnh gốc (`skipVariant`) — không phóng to ảnh nhỏ. Kết quả lưu ở cột JSON
`artworks.variants`; `thumbnail_url` vẫn được ghi để code cũ không gãy.

Điểm cần biết khi sửa:

**Xử lý panic trong transaction.** `defer` bắt `recover()` → rollback → **panic lại**. Không
nuốt panic, cũng không để transaction treo. Mẫu này lặp ở `upload_service.go:229` và
`artwork_service.go:296`.

**`io.Copy` không nhận context.** Nên chạy trong goroutine, quá hạn thì **đóng file** để cắt
I/O, chờ goroutine thoát tối đa 3 giây. Xem `upload_service.go:118-166`.

**Giới hạn đồng thời trong bulk upload**: semaphore 5 luồng, lỗi một file không hỏng cả lô.

`enrichArtworks` (`artwork_service.go:551`) gom awards/reactions/comments/student theo lô
qua các hàm `*ByIDs`/`*Batch` — N+1 cho `student` từng tồn tại, đã hết từ khi
`StudentRepository.ListByIDs` được thêm.

## `internal/handlers` — Tầng HTTP

13 file. Chỉ: kiểm tra method → parse input → gọi service → map lỗi → trả JSON.

| File | Dòng | Phục vụ |
|---|---|---|
| `public_handler.go` | 820 | Toàn bộ `/api/v1/public/*`, trang chia sẻ OG, `/sitemap.xml`, `/robots.txt`, tải ảnh có watermark |
| `artwork_handler.go` | 513 | Quản trị tác phẩm (gồm `HandleSetFeaturedBatch`, `HandleDeleteBatch`, `HandleDownload`) |
| `api_handler.go` | 257 | Upload + health |
| `admin_auth_handler.go` | 274 | Luồng OAuth + phiên |
| `award_handler.go` | 155 | Giải thưởng |
| `topic_category_handler.go` | 140 | Nhóm chủ đề sáng tạo, cùng mẫu `award_handler.go` |
| `error_mapper.go` | 95 | Chuẩn hoá lỗi, `sanitizeError` |
| `meta_handler.go` | 62 | Trường + khối lớp |
| `base_handler.go` | 54 | `SendSuccess`/`SendError` |
| `dashboard_handler.go` · `metrics_handler.go` | | |

⚠️ **`PublicHandler` gọi thẳng repository** (`reactionRepo`, `commentRepo`, `viewRepo`), bỏ
qua tầng service. Chấp nhận được khi chỉ là CRUD một bảng, nhưng thêm bất kỳ luật nghiệp vụ
nào (lọc từ khoá, kiểm duyệt, thông báo) thì phải tách service trước.

`PublicHandler` cũng chứa `html/template` cho trang chia sẻ Open Graph — dùng
`html/template` chứ **không** `text/template`, để tự escape mọi giá trị chèn vào.

## `internal/container` — Nơi lắp ráp

**File**: [container.go](../internal/container/container.go) · 724 dòng

Composition root **duy nhất**. Không biến toàn cục, không `init()` ẩn, không service locator.

Ba phần:

1. **`NewContainer()`** (dòng 75–388) — dựng mọi dependency theo thứ tự: config → thư mục →
   HTTP client AWS → S3 → DB + migration → auth → repository → service → handler.
2. **`GetServerHandler()`** (416–605) — đăng ký route và lắp chuỗi middleware. **Đọc file
   này để biết route thật**, không tin tài liệu.
3. **`Shutdown()`** — dừng rate limiter, dừng bộ đếm chống quét (BotGuard), dừng cleanup
   phiên đăng nhập, đóng DB.

Điểm cần biết khi sửa:

- **Khởi tạo có điều kiện**: hai khối `if db != nil` (187–202 và 288–312) quyết định phần
  nào của hệ thống tồn tại. Thêm tính năng cần DB thì đặt vào đúng khối này.
- **Thứ tự middleware**: áp sau = nằm ngoài. Đọc ngược từ dòng 574 lên để hình dung thứ tự
  request đi qua.
- **Ba rate limiter riêng biệt**, đều được đăng ký vào `c.RateLimiters` để `Shutdown()` dừng
  hết — thêm limiter mới nhớ đăng ký, nếu không sẽ rò rỉ goroutine.
- **`spaFileServer`** kiểm tra đường dẫn tuyệt đối nằm trong `dist` trước khi phục vụ —
  chống path traversal. Sửa cẩn thận.

---

## `web/` — Giao diện

Xem [detail_design/06-frontend.md](./detail_design/06-frontend.md) cho thiết kế đầy đủ.

| Thư mục | Nội dung |
|---|---|
| `src/pages/public/` | 6 trang công khai (chủ, tiêu biểu, phòng triển lãm, bảng vàng, thư ngỏ, 404) + `PublicLayout` |
| `src/pages/admin/` | 6 trang quản trị + `AdminLayout` + trang đăng nhập |
| `src/components/public/` | 25 component trang public |
| `src/components/admin/` | 13 file trong thư mục quản trị (component + kiểu dùng chung) |
| `src/lib/` | Client API tách theo khu vực xác thực |
| `src/hooks/` | `useAdminAuth`, `useDeviceTier`, `usePageMeta`, `useParallaxScroll`, `useRailScroll`, `useScrollableBody` |
| `src/styles/fonts.css` | `@font-face` cho font tự phục vụ — **sinh bằng `scripts/fetch-fonts.sh`**, không sửa tay |
| `public/fonts/` | File `woff2` của Be Vietnam Pro + Fraunces, tách theo subset `unicode-range` |
| `public/splash.js` | Script gỡ splash — để ngoài `index.html` vì CSP cấm script nội tuyến |

Điểm cần biết khi sửa:

- Kiểu TypeScript phải khớp struct Go — chú ý `s3_url` → **`image_url`**.
- Thêm route mới nhớ dùng `lazy()`, nếu không sẽ kéo ngược vào bundle chính.
- Chỉ truy cập `visitor_token` qua `lib/visitorToken.ts`.
- `npm run build` chạy `tsc --noEmit` trước — lỗi kiểu sẽ chặn build.
- **Không** thêm `<script>` nội tuyến hay `<link>` stylesheet trỏ ra origin ngoài vào
  `index.html` — CSP (`script-src 'self'`, `style-src 'self' 'unsafe-inline'`) chặn cả hai, và
  lỗi chỉ hiện trong console trình duyệt người dùng chứ không vào log server.

## `scripts/` — Tiện ích bảo trì

`fetch-fonts.sh` — tải font về `web/public/fonts/` và sinh lại `web/src/styles/fonts.css`.

Đổi bộ weight thì sửa biến `URL` trong script rồi chạy lại, đừng sửa `fonts.css` bằng tay:
lần chạy sau sẽ ghi đè.

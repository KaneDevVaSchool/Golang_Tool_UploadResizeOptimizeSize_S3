# 01 — Trạng thái hiện tại

Đối chiếu code trên nhánh `feature/artwork-contest-system`, khảo sát ngày **2026-09-06**,
rà lại ngày **2026-09-07**.

## 1. Kiểm chứng bằng công cụ

| Kiểm tra | Lệnh | Kết quả |
|---|---|---|
| Biên dịch | `go build ./...` | ✅ Sạch, không lỗi |
| Kiểm thử | `go test ./...` | ✅ Toàn bộ gói đạt |

```text
ok      s3-upload-tool/internal/handlers
ok      s3-upload-tool/internal/middleware
ok      s3-upload-tool/internal/service
ok      s3-upload-tool/internal/utils
```

### Bộ mã hoá WebP — đã giải quyết

Bản khảo sát trước ghi nhận `TestGenerateVariantsProducesAllSizesAndFormats` thất bại vì
WebP sinh ra lớn hơn JPEG khoảng 10 lần. Nguyên nhân: `golang.org/x/image/webp` **chỉ có
bộ giải mã, không có bộ mã hoá**, còn `nativewebp` thì chỉ làm được lossless (VP8L) — với
tranh vẽ và ảnh chụp, lossless cho ra file lớn hơn JPEG nhiều lần.

Đã đổi sang `github.com/gen2brain/webp`: encode lossy thật (VP8) qua WASM nhúng + purego,
tức vẫn **không cần cgo** nên khâu build tĩnh giữ nguyên. Kết quả đo lại:

```text
thumb   webp=  1414B  jpg=  5760B   (webp nhỏ hơn 76%)
medium  webp=  6322B  jpg= 27205B   (webp nhỏ hơn 77%)
large   webp= 11728B  jpg= 62811B   (webp nhỏ hơn 82%)
```

Mục P1.5 trong [02-roadmap.md](./02-roadmap.md) vì thế đã đóng.

## 2. Đã hoàn thành ✅

### Hạ tầng upload

| Hạng mục | Ghi chú |
|---|---|
| Upload đơn lên S3 | Có timeout co giãn theo dung lượng, dọn file tạm bằng `defer` |
| Bulk upload | 5 luồng song song, lỗi một file không hỏng cả lô |
| Nhiều lớp kiểm tra file | Tên file, đuôi, dung lượng (2 lần) — ⚠️ **không còn** kiểm tra magic byte |
| Tải ảnh có watermark | Endpoint public đóng mốc VAS, ghi nhật ký vào `artwork_downloads` |
| Sinh S3 key chống trùng | Timestamp + 12 ký tự ngẫu nhiên từ `crypto/rand` |
| Tự dò region của bucket | Chống lỗi cấu hình region khó chẩn đoán |

### Miền tác phẩm

| Hạng mục | Ghi chú |
|---|---|
| CRUD tác phẩm | Tạo/sửa/xoá/xem, phân trang, lọc đa điều kiện |
| Quy trình 2 bước | Đẩy S3 trước, nhập metadata sau |
| Quản lý giải thưởng | CRUD giải, gán/gỡ cho tác phẩm, một tác phẩm nhận **nhiều giải cùng lúc** (`award_ids`) |
| Giải theo khối lớp | `awards.grade_level_id` (migration `015`) — hội thi chia giải riêng theo từng khối, `grade_level_id` rỗng vẫn là giải dùng chung toàn hệ thống |
| Nhóm chủ đề sáng tạo | Bảng `topic_categories` (migration `016`) + CRUD `/api/v1/admin/topic-categories`, lọc theo cấp học qua `education_level`, quản lý qua trang `/admin/topic-categories` (`TopicCategoriesPage`) — xem [03-artwork-domain.md](../detail_design/03-artwork-domain.md) |
| Bảng điều khiển | Thống kê theo khu vực/khối, xếp hạng trường và tác phẩm |
| Enrich theo lô | Tránh N+1 cho giải/cảm xúc/bình luận |
| Bật/tắt tiêu biểu | Đơn lẻ và hàng loạt (`bulk-featured`) |

### Trang public

| Hạng mục | Ghi chú |
|---|---|
| 6 trang | Trang chủ, tiêu biểu, phòng triển lãm, bảng vàng, thư ngỏ, 404 |
| Cảm xúc ẩn danh | 6 loại, idempotent nhờ `INSERT IGNORE` + ràng buộc UNIQUE |
| Bình luận ẩn danh | Tự nhập tên, tự xoá bình luận của mình |
| Đếm lượt xem | Mỗi lần mở là 1 lượt — bỏ chống trùng 24 giờ ngày 2026-09-07 |
| Tải ảnh có watermark | Qua proxy backend, ghi nhật ký vào `artwork_downloads` |
| Trợ lý mascot | Tìm kiếm trong dữ liệu sẵn có, không gọi dịch vụ AI nào |
| Trang chia sẻ Open Graph | Render phía server để Facebook/Zalo lấy được ảnh preview |
| Tìm kiếm và lọc | Theo tên, trường, khối, cấp học |
| Section theo nhóm chủ đề | `/phong-trien-lam` có thêm 1 section cho mỗi nhóm chủ đề đang active, đặt sau 2 section cấp học, tự ẩn nếu nhóm chưa có tác phẩm — `GalleryTopicSection` |

### Xác thực và bảo mật

| Hạng mục | Ghi chú |
|---|---|
| Đăng nhập Google OAuth | Có kiểm tra state chống CSRF, bắt buộc `email_verified` |
| Session lưu DB | Thu hồi được ngay, tự dọn phiên hết hạn mỗi giờ |
| Danh sách email cho phép | Hai tầng: email cụ thể ưu tiên hơn domain, mặc định đã an toàn |
| CSRF toàn cục | Double-submit cookie, so token bằng `crypto/subtle`, không parse multipart trước khi kiểm tra |
| Rate limit 4 tầng | Toàn cục, metrics, ghi dữ liệu public, tải ảnh gốc |
| Giới hạn đồng thời | Semaphore có thời gian chờ |
| Kiểm tra bắt buộc ở production | Chặn khởi động khi thiếu API key / CORS quá rộng |
| IP client đáng tin | `TRUSTED_PROXIES` — chỉ đọc `X-Forwarded-For` từ proxy khai báo; khoá đếm gom IPv6 về `/64` |
| Chống tải trọn site | `middleware/botguard.go` — nhận diện công cụ tải hàng loạt và nhịp quét, miễn trừ bot tìm kiếm hợp lệ |
| Header phòng thủ | `middleware/security_headers.go` — CSP, COOP/CORP, Referrer-Policy, HSTS tuỳ chọn |
| Trần body theo đường dẫn | `MAX_JSON_BODY_KB` cho endpoint thường, trần upload giữ riêng |

### Vận hành

| Hạng mục | Ghi chú |
|---|---|
| Quy trình triển khai | Làm tay từng bước, ghi trong [deploys/00-tu-dau-den-cuoi.md](../deploys/00-tu-dau-den-cuoi.md) — bốn script `.sh` đã gỡ ngày 2026-09-07 |
| systemd + Nginx | Có sẵn file cấu hình mẫu trong `deploy/` |
| Migration tự động | toàn bộ file trong `internal/database/migrations/`, idempotent qua `schema_migrations` |
| Log theo ngày | Ghi đồng thời stdout + file |
| Tắt máy an toàn | Drain request trước, đóng tài nguyên sau |
| Chỉ số vận hành | `/api/v1/metrics` |

### Tối ưu truyền tải (hoàn thành 2026-09-06)

| Hạng mục | Ghi chú |
|---|---|
| Sinh biến thể ảnh | `internal/service/image_variants.go` — thumb/medium/large × WebP/JPEG, sinh song song theo số CPU, bỏ qua cỡ lớn hơn ảnh gốc |
| Lưu biến thể | Cột `artworks.variants` kiểu JSON (migration `013`); `thumbnail_url` nay **đã được ghi** (`artworkService.buildVariants`) |
| Frontend chọn cỡ | `web/src/lib/artworkImage.ts` dựng `<picture>`/`srcset`, có đường lui khi tác phẩm chưa có biến thể |
| Nén phản hồi HTTP | `internal/middleware/compress.go` **đã nối** vào chuỗi trong `GetServerHandler` |
| Tách chunk vendor | `web/vite.config.ts` — framer-motion/react-router/react tách riêng theo thư viện, ổn định qua nhiều lần deploy |
| Gom truy vấn học sinh | `StudentRepository.ListByIDs` gộp một truy vấn, khử id trùng; `enrichArtworks` hết N+1 |
| Bảng vinh danh một truy vấn | `HasAward` trong `ArtworkFilter` cho phép lấy mọi tác phẩm có giải một lần rồi tự nhóm, thay vì gọi `ListArtworks` cho từng giải |
| Index cho truy vấn nóng | Migration `014`: `(is_published, created_at DESC)` và `(artwork_id, is_hidden, created_at DESC)` — bỏ được filesort |

### Trạng thái chờ khi chuyển trang (hoàn thành 2026-09-06)

Chưa nằm trong roadmap có sẵn — làm theo yêu cầu trực tiếp cải thiện trải nghiệm khi mạng
chậm/lag lúc chuyển trang. Chi tiết kỹ thuật ở
[06-frontend.md § Trạng thái chờ khi chuyển trang](../detail_design/06-frontend.md).

| Hạng mục | Ghi chú |
|---|---|
| Thanh tiến trình chuyển trang | `web/src/components/RouteProgress.tsx` — phát hiện transition treo qua `useSyncExternalStore` trên URL thật, vì `useLocation()` bị giữ lại cùng cây cũ |
| `RouteFallback` vẽ lại | Khung bố cục (skeleton) thay spinner, tránh nhảy layout khi nội dung thật thay vào |
| Splash lúc boot | `web/index.html` — SVG inline (không dùng ảnh mascot 620KB, sẽ tải sau cả nội dung thật trên mạng chậm), tự gỡ bằng `MutationObserver` trên `#root` |
| Hoạt cảnh vào trang | `.route-enter` (tokens.css) — fade + trượt nhẹ, áp cho cả `PublicLayout` và `AdminLayout` |

### Trang tải tác phẩm lên: 2 chế độ (hoàn thành 2026-09-06)

Chưa nằm trong roadmap có sẵn — làm theo yêu cầu trực tiếp nâng cấp trang
`/admin/artworks/upload`. Chi tiết kỹ thuật ở
[06-frontend.md §11](../detail_design/06-frontend.md).

| Hạng mục | Ghi chú |
|---|---|
| Chế độ "Tải 1 ảnh" | Preview lớn + `ArtworkMetaForm` đầy đủ, dùng khi cần xem kỹ trước khi lưu |
| Chế độ "Tải nhiều ảnh" | `ArtworkBulkTable` — bảng nhập liệu, mỗi ảnh 1 hàng, sửa trực tiếp trong ô, hover/focus thumbnail phóng to |
| Thứ tự upload đổi | Ảnh chỉ preview ở client (`URL.createObjectURL`); chạm S3 lúc bấm Lưu, không phải lúc chọn file — đánh đổi khác thiết kế bulk-upload gốc, xem [03-artwork-domain.md](../detail_design/03-artwork-domain.md) |

### Nâng cấp UI/UX trang tải tác phẩm lên + chuẩn validate dùng chung (hoàn thành 2026-09-06)

Chưa nằm trong roadmap có sẵn — làm theo yêu cầu trực tiếp nâng cấp UI/UX
`/admin/artworks/upload` và áp chuẩn validate cho toàn bộ form admin. Chi tiết kỹ thuật ở
[06-frontend.md §12](../detail_design/06-frontend.md), mục roadmap ở
[02-roadmap.md P2.9](./02-roadmap.md).

| Hạng mục | Ghi chú |
|---|---|
| `validateMetaForm()` | Nguồn sự thật duy nhất cho field bắt buộc của `ArtworkMetaFormValues`, dùng chung cho cả logic (`isMetaFormValid`) lẫn UI (dấu `*`, lỗi theo field) |
| Lỗi hiện tại field | Khi field "touched" (blur) hoặc khi bấm Lưu mà form còn thiếu (`showAllErrors`/`forceShowErrors`) — không đỏ lòm ngay lúc form vừa mở trống |
| Phạm vi áp dụng | Cả 3 nơi dùng `ArtworkMetaForm` (tab 1 ảnh, `ArtworkBulkTable`, `ArtworkEditModal`) + form giải thưởng `AwardsPage` |

### Modal admin bị vỡ layout do `.route-enter` — đã sửa (2026-09-06)

Hệ quả không lường trước của mục trên: `will-change: transform` trong `.route-enter` biến
`.admin-content-inner` thành containing block mới cho `position: fixed`, khiến
`ArtworkEditModal` và `ConfirmDialog` (cả hai `position: fixed; inset: 0`) bị nhốt trong
khung cuộn `.admin-content` thay vì phủ toàn viewport — tràn xuống đáy, bị cắt. Sửa bằng
`createPortal(..., document.body)` cho cả hai. Chi tiết cơ chế ở
[06-frontend.md § 10](../detail_design/06-frontend.md). Bài học cho modal/dialog toàn màn
hình thêm sau này: luôn portal ra `document.body`, đừng dựa vào việc ancestor "trông có vẻ"
không đặt `transform`.

### Trang tải tác phẩm lên: redesign bỏ 2 tab, gộp lưới + panel (hoàn thành 2026-09-06)

Giao diện 2 tab mô tả ở mục trên bị đánh giá "xấu, không tối ưu" — thay hẳn, không remix.
Chi tiết kỹ thuật ở [06-frontend.md §11](../detail_design/06-frontend.md), mục roadmap ở
[02-roadmap.md P2.10](./02-roadmap.md).

| Hạng mục | Ghi chú |
|---|---|
| 1 luồng duy nhất | Bỏ tab "1 ảnh"/"nhiều ảnh" — chọn 1 hay nhiều ảnh đều vào chung 1 giao diện: `ArtworkPickerGrid` (lưới thẻ) trái + panel sửa phải |
| `ArtworkPickerGrid.tsx` thay `ArtworkBulkTable.tsx` (đã xoá) | Lưới thẻ ảnh vuông thay bảng HTML mỗi ảnh 1 hàng — không còn cuộn ngang trên mobile, không còn popover phóng to hover |
| Sửa hàng loạt | Chọn ≥2 thẻ → panel "áp dụng cho N ảnh", field điền thì ghi đè mọi thẻ đang chọn (`applyPatch()`), field trống giữ nguyên — tái dùng nguyên `ArtworkMetaForm`, không thêm prop mới |
| `AdminDropzone.tsx` mới | Khung kéo-thả riêng cho khu quản trị (lúc này tách khỏi `Dropzone.tsx` của trang `/upload` độc lập — trang đó đã bị xoá sau, xem mục dưới). Bỏ hiệu ứng framer-motion (xoay 3D, viền chạy) khỏi bản admin |

### Dashboard viết lại: bỏ Recharts (hoàn thành 2026-09-06)

Chưa nằm trong roadmap có sẵn — làm theo yêu cầu trực tiếp cải thiện trang `/admin`. Chi
tiết kỹ thuật ở [06-frontend.md §7](../detail_design/06-frontend.md), mục roadmap ở
[02-roadmap.md P2.8](./02-roadmap.md).

| Hạng mục | Ghi chú |
|---|---|
| Số liệu ra quyết định | `activity` (nhịp hoạt động, mặc định 14 ngày gần nhất — từ 2026-09-07 lọc được theo tháng/khoảng ngày tuỳ chọn qua `from`/`to`, xem [02-roadmap.md P2.12](./02-roadmap.md)), `school_coverage` (trường thiếu khối nào), `operations` (hàng chờ xử lý) — gộp vào `GET /api/v1/admin/dashboard/stats` sẵn có |
| Bỏ Recharts | `StatCard` tự vẽ sparkline SVG; `package.json` và `vite.config.ts` không còn `recharts`/`vendor-charts` |
| Bố cục mới | 4 ô chỉ số + 3 card khu vực (Sài Gòn/Cần Thơ/Vũng Tàu) thay ba biểu đồ cũ |

### Lọc khu vực bằng SQL, sửa lệch cột dashboard, xoá công cụ upload nội bộ (2026-09-06)

Mục roadmap [02-roadmap.md P1.3](./02-roadmap.md) cộng vài việc phát sinh cùng đợt.

| Hạng mục | Ghi chú |
|---|---|
| `Region` vào `ArtworkFilter` | Lọc `school_id IN (SELECT id FROM schools WHERE region = ?)`, thay lọc trong bộ nhớ sau khi cắt 100 bản ghi. Áp dụng cho `/api/v1/public/artworks`, `/featured`, `/api/v1/admin/artworks`. `models.IsKnownRegion` chặn giá trị lạ trước khi vào SQL |
| Sửa `dashboardRepository.TopArtworksByEngagement` | `SELECT` từng liệt kê đủ `artworkSelectColumns` nhưng `Scan` chỉ đọc một phần cột dashboard cần — thêm cột artwork nào (vd `topic_category_id`) là lệch vị trí, `Scan` gán nhầm kiểu và trắng cả trang `/admin`. Sửa bằng cách viết `SELECT` liệt kê đúng khớp `Scan`, không dùng chung `artworkSelectColumns` nữa |
| Xoá công cụ upload nội bộ `/upload` | `UploadTool.tsx`, `Dropzone.tsx`, `PreviewPanel.tsx`, `ResultPanel.tsx`, `Onboarding.tsx`, `StepTimeline.tsx`, `PreviewImage.tsx`, `lib/imageTransform.ts`, `lib/previewImage.ts`, route trong `App.tsx`. Endpoint backend `/api/v1/upload*` **không đổi** — chỉ mất giao diện thao tác tay qua trình duyệt. Xem [ARCHITECTURE.md](../ARCHITECTURE.md) |
| Gỡ dải "theo khu vực" khỏi 3 trang admin | `RegionSummaryStrip.tsx` + `useRegionSummary.ts` xoá khỏi `ArtworksListPage`/`AwardsPage`/`TopicCategoriesPage` — vừa thêm ở mục roadmap trước đó, gỡ lại vì bộ lọc `region=` trực tiếp trên danh sách đã đủ dùng. Endpoint `GET /api/v1/admin/dashboard/region-summary` ở backend vẫn còn, hiện không có nơi gọi |
| Mascot "Rồng nhỏ" ở trang public | `MascotAssistant.tsx` + `lib/mascotSearch.ts` — trợ lý tìm kiếm nổi góc dưới-phải (chỉ desktop ≥1024px), so khớp từ khoá cục bộ (bỏ dấu + Levenshtein khoảng cách ngắn) trên API public sẵn có, **không gọi AI ngoài**. Chi tiết ở [06-frontend.md](../detail_design/06-frontend.md) |

### Tải ảnh gốc kèm watermark, xoá tác phẩm dọn luôn S3, bỏ chống trùng lượt xem (2026-09-06)

Không nằm trong lộ trình gốc — ba thay đổi nghiệp vụ có chủ đích, đảo ngược quyết định cũ.

| Hạng mục | Ghi chú |
|---|---|
| `GET /api/v1/public/artworks/{id}/download` | Tải ảnh gốc kèm watermark logo VAS, stream qua backend (same-origin). Lỗi watermark chỉ ghi log, vẫn trả ảnh gốc — không chặn tải. Mỗi lượt tải ghi vào `artwork_downloads` (`source='public'`, ẩn danh). Chi tiết ở [API.md](../API.md) |
| `GET /api/v1/admin/artworks/{id}/download` | Cùng cơ chế, dành cho admin: **không** watermark, **không** ép `is_published`. Ghi vào `artwork_downloads` kèm `admin_user_id` của người tải — xem [02-roadmap.md P2.14](./02-roadmap.md). Dùng trong nút "Tải ảnh gốc" ở `ArtworkEditModal` và trong tải hàng loạt ở `ArtworksListPage` — xem [02-roadmap.md P2.13](./02-roadmap.md) |
| `DeleteArtwork` xoá luôn S3 | Đảo ngược quyết định cũ (từng cố ý giữ file S3 khi xoá DB để tránh mất dữ liệu do bấm nhầm). `collectArtworkS3Keys` gom key ảnh gốc + mọi biến thể trước khi gọi `s3Repo.Delete`, xoá S3 **trước** DB row. Xem [03-artwork-domain.md §4](../detail_design/03-artwork-domain.md), rủi ro cập nhật ở [03-risks.md R4](./03-risks.md) |
| Bỏ chống trùng lượt xem 24h | `RecordView` giờ luôn +1 `view_count` mỗi lần gọi, không còn dedupe theo `visitor_token`/24h. Tăng tốc độ phình bảng `artwork_views` — xem [03-risks.md R5](./03-risks.md) |

### Trang danh sách tác phẩm: xoá hàng loạt, tải ảnh hàng loạt, dung lượng ảnh, toggle công khai (2026-09-07)

Không nằm trong lộ trình gốc — xem [02-roadmap.md P2.13](./02-roadmap.md) để biết chi tiết
đầy đủ. Đóng luôn mục "Đang làm dở" trước đó (nút tải ảnh gốc cho admin thiếu điểm bấm ở UI).

| Hạng mục | Ghi chú |
|---|---|
| `DELETE /api/v1/admin/artworks/bulk-delete` | Xoá hàng loạt, tuần tự từng tác phẩm (kèm S3), 1 lỗi không chặn cả lô |
| Tải ảnh hàng loạt (`.zip`) | `lib/artworkDownload.ts` + `fflate`, đóng gói ở client vì ảnh trên S3 là cross-origin |
| Dòng dung lượng ảnh | `formatBytes(item.file_size)` ở cả List và Grid của `ArtworksListPage` |
| Toggle "Hiển thị công khai" | Dùng lại `artworks.is_published` có sẵn, đặt trong `ArtworkEditModal` (không phải `ArtworkMetaForm`) |

### SEO kỹ thuật cơ bản: sitemap, robots.txt, meta động, JSON-LD (hoàn thành 2026-09-07)

Không nằm trong lộ trình gốc — xem [02-roadmap.md P2.15](./02-roadmap.md) để biết chi tiết đầy
đủ và lý do không làm SSR/prerender toàn phần.

| Hạng mục | Ghi chú |
|---|---|
| `GET /sitemap.xml`, `GET /robots.txt` | Route Go động, dựng URL tuyệt đối từ header request, cùng cơ chế `HandleArtworkSharePage` |
| `ArtworkService.ListPublishedForSitemap` | Tự lặp trang vượt trần `page_size=100`, không enrich |
| OG/Twitter/JSON-LD nâng cao | Trang chia sẻ tác phẩm thêm `og:image:width/height`, `twitter:image:alt`, breadcrumb `BreadcrumbList` |
| `hooks/usePageMeta.ts`, `components/JsonLd.tsx` | Title/description/canonical động + structured data (`WebSite`/`CollectionPage`/`BreadcrumbList`) cho 4 trang public, không thêm `react-helmet-async` |

## 3. Đang làm dở 🚧

Không còn hạng mục nào dở dang ở nhánh này. Việc tiếp theo xem
[02-roadmap.md](./02-roadmap.md).

## 4. Nợ kỹ thuật ⚠️

Xếp theo mức độ ảnh hưởng thực tế:

### `PublicHandler` gọi thẳng repository

Bỏ qua tầng service cho cảm xúc/bình luận/lượt xem. Chấp nhận được khi chỉ là CRUD một
bảng, nhưng sẽ thành vấn đề ngay khi thêm luật nghiệp vụ (lọc từ khoá, kiểm duyệt, thông báo).

### Hai bảng có thể lệch nhau

`UpdateArtwork` sửa `school_id`/`grade_level_id` trên `artworks` nhưng không đồng bộ ngược
về `students` (`artwork_service.go:264-270`).

### Chưa có API kiểm duyệt bình luận

Cột `is_hidden` có, repository lọc theo nó, nhưng **không có endpoint** để bật/tắt. Hiện
phải `UPDATE` bằng SQL tay.

### Không còn kiểm tra magic byte ở bất kỳ đường upload nào ⚠️

`ValidateFileContent` trước đây **chỉ** chạy ở đường chunked, và đường đó đã bị gỡ ngày
2026-09-07. Nghĩa là hiện không đường nào đối chiếu nội dung thật với đuôi file.

Mức độ đã đổi: trước là "bất đối xứng giữa hai đường", giờ là một lớp phòng thủ **mất hẳn**.
Xem P1.4 trong [02-roadmap.md](./02-roadmap.md).

### Bảng `uploads` không còn ai dùng

Migration 001 vẫn tạo bảng này (không sửa migration đã commit), nhưng
`upload_repository.go` và `upload_record.go` đã bị xoá cùng đường upload có transaction. Bảng
nằm đó rỗng, không code nào đọc/ghi. Dọn được bằng một migration `DROP TABLE` khi tiện.

### Độ phủ kiểm thử thấp

Test tập trung ở `utils`, `middleware`, và phần sinh biến thể ảnh của `service`. **Chưa có
test** cho `repository` và `container`, cũng như cho frontend.

## 5. Chưa có 📋

| Hạng mục | Ghi chú |
|---|---|
| Dọn `artwork_views` cũ | Bảng chỉ tăng, không bao giờ giảm |
| Dọn ảnh mồ côi trên S3 | Từ tác phẩm đã xoá và bulk upload bỏ dở |
| Xoay vòng log | Không tự xoá file log cũ |
| Phân quyền theo vai trò | Cột `role` có nhưng mọi admin quyền như nhau |
| Nhật ký thao tác admin (xoá/sửa) | Đã có phần **tải ảnh** (`artwork_downloads`, xem [02-roadmap.md P2.14](./02-roadmap.md)); ai xoá/sửa tác phẩm lúc nào thì vẫn chưa ghi lại |
| Xuất dữ liệu | Không xuất được CSV/Excel danh sách tác phẩm |

## 6. Tình trạng commit

Khối thay đổi lớn từng tồn đọng trong `git status` (khảo sát trước ghi nhận ~35 file sửa,
~17 file mới) **đã được commit** ngày 2026-09-06, tách theo từng giai đoạn:

| Commit | Phạm vi |
|---|---|
| `551c359` | Backend: biến thể ảnh, gzip, migration `013`, gỡ `Dockerfile` |
| `d0ec7c2` | Frontend: `artworkImage.ts`, tách chunk vendor, trang public |
| tiếp theo | Tài liệu `docs/`, script `deploy/`, bộ quy tắc `.claude/` |

Lưu ý `Dockerfile` đã bị xoá — thay đổi có chủ đích vì phương án triển khai chính thức là
binary + systemd, xem [../deploys/README.md](../deploys/README.md).

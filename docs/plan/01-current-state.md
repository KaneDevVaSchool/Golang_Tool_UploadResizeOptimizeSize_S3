# 01 — Trạng thái hiện tại

Đối chiếu code tại commit `d0ec7c2`, khảo sát ngày **2026-09-06**.

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
| Upload chia phần đến 200MB | Init/chunk/complete/abort, TTL 45 phút, tự dọn phiên |
| Upload kèm transaction | Ghi bản ghi audit `uploads`, rollback đúng cả khi panic |
| Bulk upload | 5 luồng song song, lỗi một file không hỏng cả lô |
| Nhiều lớp kiểm tra file | Tên file, đuôi, dung lượng (2 lần), magic byte (đường chunked) |
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
| 4 trang | Trang chủ, tiêu biểu, phòng triển lãm, bảng vàng |
| Cảm xúc ẩn danh | 6 loại, idempotent nhờ `INSERT IGNORE` + ràng buộc UNIQUE |
| Bình luận ẩn danh | Tự nhập tên, tự xoá bình luận của mình |
| Đếm lượt xem | Chống trùng trong 24 giờ, cập nhật trong transaction |
| Trang chia sẻ Open Graph | Render phía server để Facebook/Zalo lấy được ảnh preview |
| Tìm kiếm và lọc | Theo tên, trường, khối, cấp học |
| Section theo nhóm chủ đề | `/phong-trien-lam` có thêm 1 section cho mỗi nhóm chủ đề đang active, đặt sau 2 section cấp học, tự ẩn nếu nhóm chưa có tác phẩm — `GalleryTopicSection` |

### Xác thực và bảo mật

| Hạng mục | Ghi chú |
|---|---|
| Đăng nhập Google OAuth | Có kiểm tra state chống CSRF, bắt buộc `email_verified` |
| Session lưu DB | Thu hồi được ngay, tự dọn phiên hết hạn mỗi giờ |
| Danh sách email cho phép | Hai tầng: email cụ thể ưu tiên hơn domain, mặc định đã an toàn |
| CSRF toàn cục | Double-submit cookie |
| Rate limit 3 tầng | Toàn cục, metrics, ghi dữ liệu public |
| Giới hạn đồng thời | Semaphore có thời gian chờ |
| Kiểm tra bắt buộc ở production | Chặn khởi động khi thiếu API key / CORS quá rộng |

### Vận hành

| Hạng mục | Ghi chú |
|---|---|
| Script triển khai | 6 bước, chạy lại nhiều lần được, có kiểm tra sức khoẻ |
| systemd + Nginx | Có sẵn file cấu hình mẫu trong repo |
| Migration tự động | toàn bộ file trong `internal/database/migrations/`, idempotent qua `schema_migrations` |
| Log theo ngày | Ghi đồng thời stdout + file |
| Tắt máy an toàn | Drain request trước, đóng tài nguyên sau |
| Chỉ số vận hành | `/api/v1/metrics` |

### Tối ưu truyền tải (hoàn thành 2026-09-06)

| Hạng mục | Ghi chú |
|---|---|
| Sinh biến thể ảnh | `internal/service/image_variants.go` — thumb/medium/large × WebP/JPEG, sinh song song theo số CPU, bỏ qua cỡ lớn hơn ảnh gốc |
| Lưu biến thể | Cột `artworks.variants` kiểu JSON (migration `013`); `thumbnail_url` nay **đã được ghi** (`internal/service/artwork_service.go:305`) |
| Frontend chọn cỡ | `web/src/lib/artworkImage.ts` dựng `<picture>`/`srcset`, có đường lui khi tác phẩm chưa có biến thể |
| Nén phản hồi HTTP | `internal/middleware/compress.go` **đã nối** vào chuỗi tại `internal/container/container.go:575` |
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
| `AdminDropzone.tsx` mới | Khung kéo-thả riêng cho khu quản trị, KHÔNG dùng chung `Dropzone.tsx` (phục vụ `/upload` độc lập, theme tối qua `index.css` nạp global). Bỏ hiệu ứng framer-motion (xoay 3D, viền chạy) khỏi bản admin |

### Dashboard viết lại: bỏ Recharts (hoàn thành 2026-09-06)

Chưa nằm trong roadmap có sẵn — làm theo yêu cầu trực tiếp cải thiện trang `/admin`. Chi
tiết kỹ thuật ở [06-frontend.md §7](../detail_design/06-frontend.md), mục roadmap ở
[02-roadmap.md P2.8](./02-roadmap.md).

| Hạng mục | Ghi chú |
|---|---|
| Số liệu ra quyết định | `activity` (nhịp 14 ngày), `school_coverage` (trường thiếu khối nào), `operations` (hàng chờ xử lý) — gộp vào `GET /api/v1/admin/dashboard/stats` sẵn có |
| Bỏ Recharts | `StatCard` tự vẽ sparkline SVG; `package.json` và `vite.config.ts` không còn `recharts`/`vendor-charts` |
| Bố cục mới | 4 ô chỉ số + 3 card khu vực (Sài Gòn/Cần Thơ/Vũng Tàu) thay ba biểu đồ cũ |

## 3. Đang làm dở 🚧

Không còn hạng mục nào dở dang ở nhánh này. Việc tiếp theo xem
[02-roadmap.md](./02-roadmap.md).

## 4. Nợ kỹ thuật ⚠️

Xếp theo mức độ ảnh hưởng thực tế:

### API key chặn cả trang public

Bật `API_REQUIRE_KEY=true` (production **bắt buộc**) sẽ áp middleware lên toàn bộ `/api/*`,
gồm cả `/api/v1/public/*`. Chỉ `/api/v1/health` được miễn (`middleware/apikey.go:19-22`).

**Hệ quả**: khách ẩn danh không xem được trang public khi cấu hình đúng chuẩn production.

**Cách sửa**: miễn trừ tiền tố `/api/v1/public/` trong middleware, giống cách đã làm cho
`/api/v1/health`.

### Lọc theo khu vực làm ở tầng ứng dụng

`HandleListFeatured` lấy tối đa 100 tác phẩm tiêu biểu rồi lọc `region` trong bộ nhớ, vì
`ArtworkFilter` không có trường `Region` (`public_handler.go:158-168`).

**Hệ quả**: nếu số tác phẩm tiêu biểu vượt 100, kết quả bị cắt **trước khi** lọc khu vực —
một số khu vực có thể thiếu tranh.

**Cách sửa**: thêm `Region` vào `ArtworkFilter`, dịch thành `school_id IN (SELECT id FROM
schools WHERE region = ?)`.

### `PublicHandler` gọi thẳng repository

Bỏ qua tầng service cho cảm xúc/bình luận/lượt xem. Chấp nhận được khi chỉ là CRUD một
bảng, nhưng sẽ thành vấn đề ngay khi thêm luật nghiệp vụ (lọc từ khoá, kiểm duyệt, thông báo).

### Hai bảng có thể lệch nhau

`UpdateArtwork` sửa `school_id`/`grade_level_id` trên `artworks` nhưng không đồng bộ ngược
về `students` (`artwork_service.go:264-270`).

### Chưa có API kiểm duyệt bình luận

Cột `is_hidden` có, repository lọc theo nó, nhưng **không có endpoint** để bật/tắt. Hiện
phải `UPDATE` bằng SQL tay.

### Chưa kiểm tra magic byte ở đường upload đơn

`ValidateFileContent` chỉ chạy ở đường chunked. Upload đơn tin vào đuôi file.

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
| Nhật ký thao tác admin | Không biết ai xoá tác phẩm nào lúc nào |
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

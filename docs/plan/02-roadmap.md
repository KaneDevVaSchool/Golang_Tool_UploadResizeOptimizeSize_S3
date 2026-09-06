# 02 — Lộ trình

Việc tiếp theo, xếp theo ưu tiên. Mỗi mục có **lý do**, **cách làm** và **tiêu chí hoàn
thành** để kiểm chứng được, không phải nhận định cảm tính.

Nguyên tắc xuyên suốt: **sửa cái đã biết hỏng trước khi thêm cái mới.**

---

## Ưu tiên 0 — Chặn đường lên production

Những mục này phải xong trước khi công bố tên miền rộng rãi.

### P0.1 — Miễn API key cho `/api/v1/public/*`

**Vấn đề.** Production **bắt buộc** `API_REQUIRE_KEY=true`, mà middleware API key áp lên
toàn bộ `/api/*` và chỉ miễn `/api/v1/health`. Khách ẩn danh sẽ nhận 401 trên toàn bộ trang
public.

**Cách làm.** Trong `middleware/apikey.go`, thêm miễn trừ cho tiền tố `/api/v1/public/`
đúng theo cách đã làm với `/api/v1/health`:

```go
if strings.HasPrefix(r.URL.Path, "/api/v1/public/") {
    next.ServeHTTP(w, r)
    return
}
```

Trang public vẫn được bảo vệ bằng rate limit riêng (20 req/phút cho ghi) và CSRF.

**Hoàn thành khi.** Với `APP_ENV=production` và `API_REQUIRE_KEY=true`:

- [ ] `curl https://<domain>/api/v1/public/artworks` trả `200` không cần header nào
- [ ] `curl https://<domain>/api/v1/upload` **không** có key vẫn trả `401`
- [ ] Mở trang public bằng trình duyệt ẩn danh, xem/thả cảm xúc/bình luận đều chạy

### ~~P0.2 — Commit khối lượng thay đổi đang treo~~ ✅ Xong 2026-09-06

Khối tồn đọng (~35 file sửa, ~17 file mới) đã được commit, tách theo giai đoạn thay vì dồn
một commit lớn:

- [x] `551c359` — backend: biến thể ảnh, gzip, migration `013`
- [x] `d0ec7c2` — frontend: dùng biến thể ảnh, tách chunk vendor, dựng lại trang public
- [x] `go build ./...` và `go test ./...` đều đạt sau khi commit
- [x] Việc gỡ `Dockerfile` được ghi rõ trong commit message

Chi tiết ở [01-current-state.md](./01-current-state.md) mục 6.

### P0.3 — Xoay vòng log

**Vấn đề.** Log ghi mỗi ngày một file, **không bao giờ tự xoá**. Đĩa đầy sẽ làm sập cả
MySQL lẫn ứng dụng.

**Cách làm.** Cấu hình `logrotate` — nội dung sẵn có trong
[deploys/03-operations.md](../deploys/03-operations.md).

**Hoàn thành khi.**

- [ ] `sudo logrotate -d /etc/logrotate.d/s3-upload-tool` chạy sạch
- [ ] Log cũ hơn 8 tuần được nén và xoá tự động

---

## Ưu tiên 1 — Đúng đắn và tin cậy

### P1.1 — Dọn `artwork_views` định kỳ

**Vấn đề.** Bảng chỉ tăng. Bản ghi cũ hơn 24 giờ vô dụng nhưng vẫn nằm đó, làm chậm dần
truy vấn chống trùng vốn chạy ở **mỗi lượt xem tranh**.

**Cách làm.** Hai lựa chọn:

- **Cron + SQL** (đơn giản, khuyến nghị): xoá theo lô mỗi đêm.
- **Goroutine trong ứng dụng**: theo đúng mẫu `startSessionCleanup` đã có sẵn ở
  `container.go:393`, nhất quán với phần còn lại của hệ thống.

**Hoàn thành khi.**

- [ ] Bản ghi cũ hơn 2 ngày bị xoá tự động
- [ ] Số dòng `artwork_views` ổn định sau một tuần theo dõi

### ~~P1.2 — Gom truy vấn học sinh theo lô~~ ✅ Xong 2026-09-06

Đặt tên `ListByIDs` thay vì `GetByIDs` cho khớp `awardRepo.ListByArtworkIDs` sẵn có.

- [x] `StudentRepository.ListByIDs` trả `map[int64]*Student` trong một truy vấn, có khử
      id trùng (nhiều tác phẩm cùng một học sinh chỉ lấy một lần)
- [x] `enrichArtworks` dùng nó — số truy vấn không còn tăng theo số tác phẩm
- [x] Dữ liệu hiển thị không đổi

Cùng đợt này cũng đóng luôn nợ **"Bảng vàng gọi lặp"**: thêm `HasAward` vào
`ArtworkFilter` để lấy toàn bộ tác phẩm có giải trong một lượt rồi tự nhóm theo giải, thay
vì gọi `ListArtworks` (kèm enrich) riêng cho từng giải.

### P1.3 — Lọc khu vực bằng SQL

**Vấn đề.** Lọc trong bộ nhớ sau khi đã cắt còn 100 bản ghi → một số khu vực có thể thiếu
tranh khi số tác phẩm tiêu biểu tăng.

**Cách làm.** Thêm `Region *string` vào `ArtworkFilter`, dịch thành
`school_id IN (SELECT id FROM schools WHERE region = ?)` — cùng mẫu với `EducationLevel`
đang dùng.

**Hoàn thành khi.**

- [ ] `/api/v1/public/artworks/featured?region=cantho` lọc ở tầng SQL
- [ ] Vẫn đúng khi tổng số tác phẩm tiêu biểu vượt 100

### P1.4 — Kiểm tra magic byte cho upload đơn

**Vấn đề.** Chỉ đường chunked kiểm tra nội dung thật; upload đơn tin vào đuôi file.

**Cách làm.** Gọi `utils.ValidateFileContent` trong `UploadService.UploadImage` sau khi ghi
file tạm, trước khi đẩy S3 — vị trí đã có sẵn `Seek(0,0)`.

**Hoàn thành khi.**

- [ ] File `.jpg` chứa nội dung không phải ảnh bị từ chối ở **cả hai** đường
- [ ] Ảnh hợp lệ vẫn upload bình thường

### ~~P1.5 — Quyết định số phận WebP~~ ✅ Xong 2026-09-06

**Vấn đề.** Bộ mã hoá cũ (`nativewebp`, chỉ lossless VP8L) sinh file lớn hơn JPEG ~10 lần.

**Đã chọn.** Phương án thứ hai trong bảng cân nhắc — đổi thư viện mã hoá — nhưng không
phải trả giá bằng cgo như dự đoán ban đầu: `github.com/gen2brain/webp` encode lossy VP8
qua WASM nhúng + purego, nên build tĩnh giữ nguyên.

**Kết quả.**

- [x] `go test ./...` đạt toàn bộ
- [x] `artworks.thumbnail_url` được điền (`internal/service/artwork_service.go:305`);
      lưới gallery dùng `<picture>` qua `web/src/lib/artworkImage.ts`
- [x] WebP nhỏ hơn JPEG 76–82% ở cả ba cỡ — số đo trong
      [01-current-state.md](./01-current-state.md)

---

## Ưu tiên 2 — Vận hành và trải nghiệm

### P2.1 — API kiểm duyệt bình luận

Cột `is_hidden` đã có và đã được lọc; chỉ thiếu endpoint.

**Cách làm.** `PATCH /api/v1/admin/artworks/{id}/comments/{commentID}` với thân
`{is_hidden: bool}`, cộng một màn hình danh sách bình luận trong admin.

**Hoàn thành khi.**

- [ ] Admin ẩn/hiện được bình luận từ giao diện, không cần chạm SQL
- [ ] Bình luận bị ẩn biến mất khỏi API public ngay

### P2.2 — Dọn ảnh mồ côi trên S3

**Cách làm.** Một lệnh CLI trong `cmd/` liệt kê object S3, đối chiếu với `artworks.s3_key`
và `uploads.s3_key`, **mặc định chỉ báo cáo** — muốn xoá thật phải thêm cờ `--delete`, và
chỉ xét object cũ hơn một khoảng an toàn (ví dụ 30 ngày).

**Hoàn thành khi.**

- [ ] Chạy được ở chế độ báo cáo, không đụng dữ liệu
- [ ] Có xác nhận rõ ràng trước khi xoá

### ~~P2.3 — Nối nén phản hồi vào chuỗi middleware~~ ✅ Xong 2026-09-06

- [x] `GzipMiddleware` đã nối tại `container.go:575`, nén `text/*` và JSON trên 1KB
- [x] Đặt **trong cùng** chuỗi (sát mux nhất) để thấy được `Content-Type` do handler đặt
- [x] Bỏ qua các định dạng đã nén sẵn — ảnh không bị nén lại

### ~~P2.4 — Trang 404 riêng~~ ✅ Xong 2026-09-06

- [x] `<Route path="*">` lồng trong cả nhánh public (`/`) và nhánh admin (`/admin`) —
  bắt mọi đường dẫn lạ ở cả hai khu vực, không chỉ ở gốc
- [x] `pages/public/NotFoundPage.tsx` — theme "khu vườn tổ tiên" dùng lại
  `FeaturedGardenScene`, mascot rồng (`vas-mascot-wave.png`) + biển gỗ "404", lối tắt về
  3 trang public chính
- [x] `pages/admin/NotFoundPage.tsx` — render trong `AdminLayout` (giữ sidebar/header),
  phong cách tối giản riêng của khu quản trị, không lặp theme khu vườn (xem
  [06-frontend.md §7](../detail_design/06-frontend.md))
- [x] Tôn trọng `prefers-reduced-motion` ở cả hai trang

### P2.5 — Xuất danh sách tác phẩm

Ban tổ chức cần bảng Excel/CSV để đối chiếu và in ấn. Có thể làm hoàn toàn ở frontend từ
dữ liệu đã tải.

### ~~P2.6 — Bulk-featured + chế độ xem Grid cho trang danh sách tác phẩm~~ ✅ Xong 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp cải thiện thao tác quản trị khi
số lượng tác phẩm lớn (chọn tay từng ảnh để đưa vào tiêu biểu không còn khả thi ở quy mô
hàng trăm tác phẩm). Chi tiết kỹ thuật ở
[06-frontend.md §10](../detail_design/06-frontend.md), API ở
[03-artwork-domain.md §8](../detail_design/03-artwork-domain.md) và [API.md](../API.md).

- [x] `PATCH /api/v1/admin/artworks/bulk-featured` — bật/tắt tiêu biểu hàng loạt bằng 1 câu
      `UPDATE ... WHERE id IN (...)`, thay vì frontend gọi lặp endpoint đơn lẻ
- [x] Trang `/admin/artworks` thêm chế độ xem Grid (lưới ảnh) cạnh List, nhớ lựa chọn qua
      `localStorage`
- [x] Chọn nhiều bằng checkbox (cả List lẫn Grid) + thanh thao tác hàng loạt
- [x] Nút "Đưa tác phẩm lên" rút gọn còn icon `+` (giữ `title`/`aria-label`)
- [x] Hover phóng ảnh tại chỗ (CSS `scale`), tắt dưới `prefers-reduced-motion` và trên
      thiết bị cảm ứng
- [x] Test service cho `SetFeaturedBatch` (danh sách rỗng, tham số đúng, lỗi repository
      được trả nguyên lên trên)
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

Không cần sửa gì ở trang public: `/tac-pham-tieu-bieu` và `/phong-trien-lam` đã tự động
đọc đúng theo cờ `is_featured`/`is_published` có sẵn (xem
[03-artwork-domain.md §1](../detail_design/03-artwork-domain.md)).

### ~~P2.7 — Trang tải tác phẩm lên: 2 chế độ + nhóm chủ đề + nhiều giải/tác phẩm~~ ✅ Xong 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp nâng cấp trang
`/admin/artworks/upload`: thêm chế độ đăng từng ảnh riêng lẻ (trước đây chỉ có đường hàng
loạt), đổi cách trình bày hàng loạt sang bảng nhập liệu trực tiếp, và mở rộng nghiệp vụ tác
phẩm sang nhóm chủ đề sáng tạo + nhiều giải mỗi tác phẩm. Chi tiết kỹ thuật ở
[06-frontend.md §11](../detail_design/06-frontend.md), schema ở
[01-database.md](../detail_design/01-database.md) (migration `015`, `016`), API ở
[API.md](../API.md).

- [x] `ArtworksUploadPage` (`/admin/artworks/upload`) có 2 tab: "Tải 1 ảnh" (preview lớn +
      form đầy đủ) và "Tải nhiều ảnh" (`ArtworkBulkTable` — bảng, mỗi ảnh 1 hàng, sửa trực
      tiếp trong ô, hover/focus thumbnail phóng to)
- [x] Đổi thứ tự upload: ảnh chỉ preview ở client (`URL.createObjectURL`), chỉ chạm S3 lúc
      bấm "Lưu"/"Lưu tất cả" — đánh đổi có ghi chú lại so với thiết kế bulk-upload gốc (xem
      [03-artwork-domain.md](../detail_design/03-artwork-domain.md))
- [x] Bảng `topic_categories` (migration `016`) + CRUD admin
      (`GET/POST/PUT/DELETE /api/v1/admin/topic-categories`, cùng mẫu `awards`) — nhóm chủ
      đề sáng tạo theo thể lệ, có thể giới hạn theo cấp học
- [x] Trang quản lý `/admin/topic-categories` (`TopicCategoriesPage`) — API CRUD ở trên có
      từ đầu nhưng ban đầu **không có giao diện nào gọi tới**, nên `topic_categories` luôn
      rỗng và dropdown "Nhóm chủ đề" ở `ArtworkMetaForm` không có gì để chọn. Bổ sung
      2026-09-06 sau khi phát hiện qua báo cáo dropdown trống, cùng mẫu `AwardsPage` — kéo-thả
      để sắp `display_order`, nhưng tách **3 khối độc lập theo cấp học** vì thứ tự chỉ có ý
      nghĩa trong cùng một cấp học (xem [06-frontend.md §12](../detail_design/06-frontend.md)).
- [x] Nút "+" cạnh dropdown "Nhóm chủ đề sáng tạo" trong `ArtworkMetaForm` mở
      `TopicCategoryQuickCreateModal` — tạo nhanh 1 nhóm ngay tại form đang nhập tác phẩm,
      không phải rời sang `/admin/topic-categories`. Bổ sung cùng đợt 2026-09-06.
- [x] `topic_categories.color_hex` (migration `017`) — màu tô icon nhóm chủ đề, cùng cơ chế
      `awards.color_hex`. Color picker (dải màu preset + input tuỳ ý, `topicCategoryColors.ts`)
      thêm vào cả `TopicCategoryQuickCreateModal` và `TopicCategoriesPage` để 2 nơi tạo/sửa
      nhóm không lệch bảng màu. Bổ sung cùng đợt 2026-09-06.
- [x] `/phong-trien-lam` (`GalleryPage`) thêm `GalleryTopicSection` — mỗi nhóm chủ đề đang
      active là 1 section, đặt **sau** 2 section cấp học có sẵn, tự ẩn nếu nhóm chưa có tác
      phẩm nào đã duyệt. Cần bổ sung parse `topic_category_id` ở
      `PublicHandler.HandleListArtworks` (tham số đã có trong `ArtworkFilter`, mới chỉ dùng ở
      nhánh admin) — xem [06-frontend.md §6](../detail_design/06-frontend.md).
- [x] `awards.grade_level_id` (migration `015`) — giải có thể gắn riêng cho 1 khối lớp thay
      vì chỉ dùng chung toàn hệ thống
- [x] Một tác phẩm nhận **nhiều giải cùng lúc** (`award_ids` thay `award_id` ở
      tạo/sửa tác phẩm) — bỏ giới hạn 1 giải/tác phẩm trước đây
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

⚠️ **Chưa làm trong đợt này** — nằm ngoài 3 mục được yêu cầu, cần hỏi lại nếu muốn triển
khai tiếp: khái niệm "cuộc thi" (contest) độc lập để phân tách 2 đợt thi Tiểu học/THCS-THPT
đang chạy chung một schema; vòng thi (sơ khảo/chung kết) và mốc thời gian; giới hạn số tác
phẩm mỗi cơ sở được chọn vào vòng sau; ràng buộc số lượng giải theo loại (vd đúng 1 Nhất,
2 Nhì — hiện admin tự tạo/gán không giới hạn); giới hạn số lượng tác phẩm tiêu biểu trưng
bày cố định (hiện `is_featured` vẫn là cờ không giới hạn số lượng).

### ~~P2.8 — Viết lại Dashboard: bỏ Recharts, thêm số liệu ra quyết định~~ ✅ Xong 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp cải thiện trang `/admin`. Ba biểu
đồ Recharts (nhịp hoạt động, phân bổ khối lớp, top tác phẩm dạng bảng) không giúp ban tổ
chức quyết định phải làm gì tiếp theo, chỉ mô tả quy mô. Chi tiết kỹ thuật ở
[06-frontend.md §7](../detail_design/06-frontend.md), response API ở
[API.md](../API.md#get-apiv1admindashboardstats).

- [x] `DashboardRepository`/`DashboardService` trả thêm ba khối: `activity` (nhịp 14 ngày
      gần nhất, đủ ngày kể cả ngày 0 hoạt động), `school_coverage` (mỗi trường đã có bài ở
      bao nhiêu khối trên tổng số khối), `operations` (bài chờ xuất bản, bình luận đã ẩn,
      tiến độ trao giải, tác phẩm chưa có tương tác) — gộp vào cùng response
      `GET /api/v1/admin/dashboard/stats`, không thêm endpoint mới
- [x] `Dashboard.tsx` đổi bố cục: 4 ô chỉ số (`StatCard` có sparkline SVG tự vẽ, không dùng
      thư viện biểu đồ) + 3 card khu vực (Sài Gòn/Cần Thơ/Vũng Tàu) thay cho ba biểu đồ cũ
- [x] Gỡ `recharts` khỏi `package.json` và nhánh `vendor-charts` khỏi `vite.config.ts` —
      thư viện chỉ dùng ở đúng 1 trang, nặng ~400KB, không còn lý do tồn tại
- [x] Test service cho `DashboardService.GetStats` (lỗi từng khối con được bọc ngữ cảnh và
      trả nguyên lên trên, tham số `days` truyền đúng xuống `ActivityTrend`)
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

### ~~P2.9 — Nâng cấp UI/UX trang tải tác phẩm lên + chuẩn validate dùng chung~~ ✅ Xong 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp nâng cấp UI/UX của
`/admin/artworks/upload` (cả 2 tab) và áp một chuẩn validate cho **toàn bộ form admin**:
dấu `*` bắt buộc + lỗi hiện ngay tại field. Chi tiết ở
[06-frontend.md §12](../detail_design/06-frontend.md).

- [x] `validateMetaForm()` — nguồn sự thật duy nhất cho field bắt buộc của
      `ArtworkMetaFormValues`, thay cho điều kiện rời rạc trước đây trong `isMetaFormValid()`
- [x] `RequiredMark` — component dấu `*` dùng chung, không mỗi form tự viết kiểu riêng
- [x] Lỗi hiện khi field "touched" (blur) hoặc khi bấm Lưu mà form còn thiếu
      (`showAllErrors`/`forceShowErrors`) — áp dụng cho cả 3 nơi dùng `ArtworkMetaForm`
      (tab 1 ảnh, `ArtworkBulkTable`, `ArtworkEditModal`) và form giải thưởng ở `AwardsPage`
- [x] Tab "Tải nhiều ảnh" hiện badge số ảnh đã chọn; panel có thanh tóm tắt (đã lưu/chưa điền đủ)
- [x] Preview ảnh tab "Tải 1 ảnh" có nút "Chọn ảnh khác" nổi trên góc, không chỉ ở footer
- [x] CSS lỗi dùng chung (`.form-field--error`, `.form-field-error-text`) + biến thể cho ô
      bảng hẹp (`.artwork-bulk-cell--error`, `.artwork-bulk-cell-error-text`)
- [x] `go build ./...`, `tsc --noEmit`, `vite build` đều sạch

### ~~P2.10 — Redesign trang tải tác phẩm lên: bỏ 2 tab, gộp 1 luồng lưới + panel~~ ✅ Xong 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp: giao diện 2 tab + bảng nhập liệu
của P2.7/P2.9 bị đánh giá "xấu, không tối ưu", yêu cầu thay hẳn chứ không remix. Chi tiết kỹ
thuật ở [06-frontend.md §11](../detail_design/06-frontend.md).

- [x] Bỏ 2 tab "Tải 1 ảnh"/"Tải nhiều ảnh" — gộp thành 1 luồng: `ArtworkPickerGrid` (lưới thẻ
      ảnh) bên trái + panel sửa bên phải, đổi theo số thẻ đang chọn
- [x] Xoá hẳn `ArtworkBulkTable.tsx` (bảng HTML mỗi ảnh 1 hàng, phải cuộn ngang dưới 768px) —
      thay bằng `ArtworkPickerGrid.tsx`, kiểu `BulkRow` chuyển sang `artworkUploadTypes.ts`
- [x] Chọn ≥2 thẻ → panel "áp dụng cho N ảnh": field điền thì ghi đè mọi thẻ đang chọn, field
      trống giữ nguyên riêng từng ảnh (`applyPatch()`) — tái dùng nguyên `ArtworkMetaForm`,
      không thêm prop mới vào component dùng chung đó
- [x] `AdminDropzone.tsx` mới — khung kéo-thả riêng cho khu quản trị, KHÔNG dùng chung
      `Dropzone.tsx` (component đó phục vụ `/upload` độc lập, theme tối/glassmorphism qua
      `index.css` nạp global). Bỏ hiệu ứng xoay 3D + viền chạy (framer-motion) khỏi bản admin,
      giữ nguyên `Dropzone.tsx`/`/upload` không đổi
- [x] Bỏ popover phóng to khi hover thumbnail (không còn cần — thẻ trong lưới đã đủ lớn)
- [x] `go build ./...` (không đổi Go), `tsc --noEmit`, `vite build` đều sạch

### ~~P2.11 — Dải card thống kê theo khu vực ở 3 trang quản trị~~ ✅ Xong 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp: thêm dải card nhỏ "số tác phẩm +
số học sinh theo khu vực" ở đầu trang Tác phẩm/Giải thưởng/Nhóm chủ đề quản trị. Dashboard
(`/admin`) giữ nguyên, không đổi. Chi tiết kỹ thuật ở
[06-frontend.md §10](../detail_design/06-frontend.md), response API ở
[API.md](../API.md#get-apiv1admindashboardregion-summary).

- [x] `DashboardRepository.RegionSummaries` — endpoint mới nhẹ
      `GET /api/v1/admin/dashboard/region-summary`, tách khỏi `/dashboard/stats` vì 3 trang
      này không cần activity/top_schools/school_coverage/operations
- [x] Số học sinh đếm **không dedupe theo tên** (`COUNT(*) students JOIN schools GROUP BY
      region`) — đúng quy ước ở [01-database.md](../detail_design/01-database.md)
- [x] `components/admin/RegionSummaryStrip.tsx` + `hooks/useRegionSummary.ts` (hook dùng
      chung 3 trang, tiền lệ `useAdminAuth.ts`) — tái dùng `StatCard.tsx` qua 3 tone khu vực
      mới thay vì viết component nhân bản; `Dashboard.tsx` không đổi
- [x] Test service cho `DashboardService.GetRegionSummary` (thứ tự cố định 3 khu vực, lan
      truyền lỗi từ repo)
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

---

## Ưu tiên 3 — Sau sự kiện

Không cần cho lần chạy này, ghi lại để không quên.

| Hạng mục | Ghi chú |
|---|---|
| Kiểm thử cho tầng service/repository | Cần khi codebase còn được phát triển tiếp |
| Nhật ký thao tác admin | Ai xoá/sửa gì lúc nào |
| Phân quyền theo vai trò | Cột `role` đã có sẵn chỗ |
| Chuyển tìm kiếm sang FULLTEXT | Chỉ khi `LIKE` thực sự chậm — **đo trước khi sửa** |
| Rate limit dùng chung (Redis) | Chỉ khi chạy nhiều instance |
| Khoá khi chạy migration | Chỉ khi chạy nhiều instance |
| Chỉ số dạng Prometheus | Nếu tích hợp hệ giám sát chung |

---

## Thứ tự thực hiện đề xuất

```text
✅ Đã xong    P0.2 commit việc tồn đọng
              P1.2 gom truy vấn học sinh (+ bảng vinh danh, index migration 014)
              P1.5 biến thể ảnh WebP
              P2.3 nén gzip phản hồi

Tuần này      P0.1 miễn API key public   ← chặn đường lên production
              P0.3 xoay vòng log

Tuần sau      P1.1 dọn artwork_views
              P1.4 magic byte upload đơn

Trước sự kiện P1.3 lọc khu vực bằng SQL
              P2.1 kiểm duyệt bình luận

Sau sự kiện   P2.2 dọn S3, P2.4, P2.5, và nhóm Ưu tiên 3
```

P0.1 nên làm đầu tiên: nó là lỗi **chặn đường**, sửa nhỏ, và nếu bỏ sót thì trang public sẽ
hỏng đúng vào lúc công bố.

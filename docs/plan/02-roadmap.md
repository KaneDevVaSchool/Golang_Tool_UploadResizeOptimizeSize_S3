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

**Vấn đề.** Bảng chỉ tăng, không bao giờ được dọn. Từ 2026-09-07 nó phình **nhanh hơn trước**:
bỏ chống trùng 24 giờ nghĩa là mỗi lần mở lightbox — kể cả mở lại cùng một tranh, kể cả tải
lại trang — đều ghi thêm một dòng. Đúng vào lúc công bố kết quả, khi lượng truy cập cao nhất,
bảng cũng lớn nhất và tăng nhanh nhất.

Truy vấn chống trùng từng chạy ở mỗi lượt xem đã không còn, nên áp lực đọc giảm; vấn đề còn
lại thuần tuý là **dung lượng lưu trữ và thời gian backup**.

**Cách làm.** Hai lựa chọn:

- **Cron + SQL** (đơn giản, khuyến nghị): xoá theo lô mỗi đêm.
- **Goroutine trong ứng dụng**: theo đúng mẫu `startSessionCleanup` đã có sẵn trong `container.go`,
  nhất quán với phần còn lại của hệ thống.

Cần quyết định trước **giữ bao lâu**: dữ liệu này giờ là nhật ký lượt xem thật, không còn là
bộ đệm chống trùng dùng xong bỏ. Giữ 30–90 ngày cho phép nhìn lại nhịp truy cập của cả kỳ
hội thi; giữ 2 ngày như dự tính ban đầu là quá ngắn cho mục đích đó.

**Hoàn thành khi.**

- [ ] Chốt thời gian lưu, ghi lý do vào [detail_design/01-database.md](../detail_design/01-database.md)
- [ ] Bản ghi quá hạn bị xoá tự động
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

### ~~P1.3 — Lọc khu vực bằng SQL~~ ✅ Xong 2026-09-06

**Vấn đề.** Lọc trong bộ nhớ sau khi đã cắt còn 100 bản ghi → một số khu vực có thể thiếu
tranh khi số tác phẩm tiêu biểu tăng.

**Đã làm.** Thêm `Region *string` vào `ArtworkFilter` (`internal/models/artwork.go`), dịch
thành `school_id IN (SELECT id FROM schools WHERE region = ?)` trong
`artworkRepository.List` — cùng mẫu với `EducationLevel` đang dùng. `models.IsKnownRegion`
chặn giá trị lạ trước khi vào SQL. Áp dụng cho cả ba endpoint đọc `region=`:
`/api/v1/public/artworks`, `/api/v1/public/artworks/featured`, `/api/v1/admin/artworks`.

**Kết quả.**

- [x] `/api/v1/public/artworks/featured?region=cantho` lọc ở tầng SQL, không còn cắt 100
      bản ghi rồi lọc trong bộ nhớ
- [x] Đúng bất kể tổng số tác phẩm tiêu biểu là bao nhiêu — không còn trần 100

### P1.4 — Kiểm tra magic byte cho upload ⚠️ nâng mức ưu tiên 2026-09-07

**Vấn đề.** Trước đây đây là chuyện *bất đối xứng*: đường chunked kiểm tra nội dung thật,
upload đơn thì không. Sau khi gỡ đường chunked (2026-09-07), lớp phòng thủ này **mất hẳn** —
không đường nào còn kiểm tra. Mục này đổi từ "cải thiện tính nhất quán" thành "bù lại một
lớp đã mất"; xem R13 trong [03-risks.md](./03-risks.md).

**Cách làm.** Gọi `utils.ValidateFileContent` trong `UploadService.UploadImage` sau khi ghi
file tạm, trước khi đẩy S3 — vị trí đã có sẵn `Seek(0,0)`. Hàm vẫn còn nguyên trong
`internal/utils`, chỉ mất chỗ gọi.

**Hoàn thành khi.**

- [ ] File `.jpg` chứa nội dung không phải ảnh bị từ chối
- [ ] Ảnh hợp lệ vẫn upload bình thường
- [ ] Có test khoá lại hành vi này, để lần dọn dẹp sau không lặng lẽ gỡ mất lần nữa

### ~~P1.5 — Quyết định số phận WebP~~ ✅ Xong 2026-09-06

**Vấn đề.** Bộ mã hoá cũ (`nativewebp`, chỉ lossless VP8L) sinh file lớn hơn JPEG ~10 lần.

**Đã chọn.** Phương án thứ hai trong bảng cân nhắc — đổi thư viện mã hoá — nhưng không
phải trả giá bằng cgo như dự đoán ban đầu: `github.com/gen2brain/webp` encode lossy VP8
qua WASM nhúng + purego, nên build tĩnh giữ nguyên.

**Kết quả.**

- [x] `go test ./...` đạt toàn bộ
- [x] `artworks.thumbnail_url` được điền (`artworkService.buildVariants`);
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
**và mọi key trong `artworks.variants`** (biến thể cũng là object thật trên S3 — bỏ sót cột
này là xoá nhầm ảnh đang dùng), **mặc định chỉ báo cáo** — muốn xoá thật phải thêm cờ
`--delete`, và chỉ xét object cũ hơn một khoảng an toàn (ví dụ 30 ngày).

`cmd/seed` đã có sẵn logic thu thập key từ `s3_key` + `variants` (`collectArtworkS3Keys`),
dùng lại thay vì viết mới.

> Phạm vi thu hẹp từ 2026-09-07: nguồn sinh rác "admin xoá tác phẩm" đã hết vì `DeleteArtwork`
> nay xoá luôn object S3. Chỉ còn bulk upload bỏ dở. Bảng `uploads` không cần đối chiếu nữa —
> nó đã chết và luôn rỗng.

**Hoàn thành khi.**

- [ ] Chạy được ở chế độ báo cáo, không đụng dữ liệu
- [ ] Có xác nhận rõ ràng trước khi xoá

### ~~P2.3 — Nối nén phản hồi vào chuỗi middleware~~ ✅ Xong 2026-09-06

- [x] `GzipMiddleware` đã nối trong `GetServerHandler`, nén `text/*` và JSON trên 1KB
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

### ~~P2.11 — Dải card thống kê theo khu vực ở 3 trang quản trị~~ ✅ Xong 2026-09-06, gỡ lại 2026-09-06

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp: thêm dải card nhỏ "số tác phẩm +
số học sinh theo khu vực" ở đầu trang Tác phẩm/Giải thưởng/Nhóm chủ đề quản trị. Dashboard
(`/admin`) giữ nguyên, không đổi. Chi tiết kỹ thuật ở
[06-frontend.md §10](../detail_design/06-frontend.md), response API ở
[API.md](../API.md#get-apiv1admindashboardregion-summary).

⚠️ **Đã gỡ khỏi frontend cùng ngày** — `RegionSummaryStrip.tsx`/`useRegionSummary.ts` xoá
khỏi cả 3 trang, xem [01-current-state.md](./01-current-state.md). Endpoint
`region-summary` ở backend vẫn còn, không xoá, chỉ không còn nơi gọi. Danh sách hoàn thành
dưới đây giữ nguyên làm lịch sử của lần triển khai ban đầu.

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

### ~~P2.12 — Nhịp hoạt động Dashboard: bộ lọc theo tháng/khoảng ngày~~ ✅ Xong 2026-09-07

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp: biểu đồ nhịp hoạt động ở P2.8 cố
định "14 ngày gần nhất", không xem lại được lịch sử xa hơn hay theo mốc tháng. Chi tiết kỹ
thuật ở [06-frontend.md §7](../detail_design/06-frontend.md), API ở
[API.md](../API.md#get-apiv1admindashboardstats).

- [x] `DashboardRepository.ActivityTrend` nhận khoảng `[from, to]` tường minh thay vì
      "N ngày gần nhất" — phục vụ được cả bộ lọc theo tháng lẫn theo khoảng ngày tuỳ ý
- [x] `GET /api/v1/admin/dashboard/stats` nhận thêm `from`/`to` tuỳ chọn (query, `YYYY-MM-DD`)
      — thiếu thì mặc định 14 ngày gần nhất, giữ nguyên hành vi cũ; khoảng > 366 ngày bị cắt
      để chặn truy vấn quét quá nhiều dữ liệu
- [x] `DashboardService.GetStats` cắt bỏ điểm 0 hoạt động ở **đầu/cuối** chuỗi
      (`trimLeadingTrailingZero`) — tháng đang chọn còn dở dang không còn vẽ một đoạn thẳng
      nằm ngang vô nghĩa; điểm 0 xen **giữa** hai ngày có dữ liệu vẫn giữ nguyên (tín hiệu
      thật, không phải khoảng trống cần bỏ)
- [x] `ActivityRangePicker` (trong `Dashboard.tsx`) — segmented control 3 chế độ: "14 ngày
      gần nhất" / "Theo tháng" (`<input type="month">`) / "Khoảng ngày" (2 `<input
      type="date">`)
- [x] `labelStepFor()` thay ngưỡng cứng "> 8 điểm thì nhảy 2" bằng bước nhảy tính theo đích
      ~60px/nhãn — không chồng chữ khi xem cả tháng hay khoảng ngày dài
- [x] `deltas` (biến động % ở 4 ô chỉ số) tổng quát từ "đúng 14 điểm" thành "≥ 4 điểm thì cắt
      đôi mà so sánh" — áp dụng được cho mọi độ dài khoảng
- [x] Test service cho `resolveActivityRange`/`trimLeadingTrailingZero` (mặc định 14 ngày,
      truyền khoảng tuỳ chỉnh, chặn khoảng quá dài, cắt đầu/cuối giữ nguyên xen giữa)
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

### ~~P2.13 — Trang danh sách tác phẩm: xoá hàng loạt, tải ảnh hàng loạt, dung lượng ảnh, toggle công khai~~ ✅ Xong 2026-09-07

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp nâng cấp `/admin/artworks`: xoá
hàng loạt, tải ảnh xuống hàng loạt, dòng nhỏ hiển thị dung lượng ảnh, và cờ quyết định tác
phẩm có qua được frontend public hay không. Không làm trang chi tiết riêng (quyết định sau
khi hỏi lại) — thay bằng nút "Tải ảnh gốc" ngay trong modal sửa. Chi tiết kỹ thuật ở
[06-frontend.md §10](../detail_design/06-frontend.md), API ở
[03-artwork-domain.md §4, §8](../detail_design/03-artwork-domain.md) và [API.md](../API.md).

- [x] **Cờ "công khai"**: dùng lại `artworks.is_published` có sẵn (đã đúng nghĩa "tắt thì ẩn
      khỏi mọi trang public") thay vì thêm cột `is_active` mới — toggle đặt trong
      `ArtworkEditModal`, tách khỏi `ArtworkMetaForm` vì đây là quyết định trạng thái, không
      phải metadata biên tập nội dung (cùng nguyên tắc đã áp dụng cho `is_featured`)
- [x] `DELETE /api/v1/admin/artworks/bulk-delete` — xoá hàng loạt, xoá **tuần tự** từng tác
      phẩm (kèm object S3 riêng, không gộp được thành 1 câu SQL như `bulk-featured`); 1 tác
      phẩm lỗi xoá S3 không chặn các tác phẩm còn lại trong lô
- [x] `GET /api/v1/admin/artworks/{id}/download` — proxy tải ảnh **gốc, không watermark,
      không ép `is_published`** (khác bản public), dùng cho cả nút tải đơn trong modal sửa
      lẫn tải hàng loạt
- [x] Tải hàng loạt đóng gói `.zip` ở phía client (`lib/artworkDownload.ts`, thư viện
      `fflate`) — bắt buộc vì ảnh nằm trên S3 (cross-origin), thẻ `<a download>` trỏ thẳng
      URL S3 sẽ mở ảnh thay vì tải; phải tải qua backend (same-origin blob) rồi tự nén
- [x] Dòng dung lượng ảnh (`formatBytes(item.file_size)`, tái dùng hàm có sẵn từ công cụ
      upload gốc) hiện ở cả 2 chế độ xem List và Grid
- [x] Test service cho `DeleteArtworkBatch` (1 item lỗi S3 không chặn cả lô, danh sách rỗng
      bị từ chối ở tầng service) — `artwork_bulk_delete_test.go`
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

### ~~P2.14 — Nhật ký ai tải ảnh gốc (admin + public)~~ ✅ Xong 2026-09-07

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp: ảnh gốc tải từ khu quản trị
không watermark và không để lại dấu vết gì, nên nếu bị phát tán sai mục đích thì không truy
được nguồn. Lấp **một phần** nợ "Nhật ký thao tác admin" ở Ưu tiên 3 dưới đây — chỉ phần
tải ảnh, chưa bao gồm xoá/sửa tác phẩm. Chi tiết schema ở
[01-database.md](../detail_design/01-database.md), API ở [API.md](../API.md).

- [x] Bảng `artwork_downloads` (migration `018`) — `artwork_id`, `admin_user_id` (NULL nếu
      tải từ public), `source` (`admin`/`public`), `ip_address`, `user_agent`,
      `downloaded_at`
- [x] `ArtworkService.LogDownload` — method dùng chung cho cả hai nguồn tải, gọi qua
      `ArtworkDownloadRepository.Create`
- [x] `ArtworkHandler.HandleDownload` (admin) ghi `source='admin'` kèm `admin_user_id` lấy
      từ session; `PublicHandler.HandleDownloadArtwork` (public) ghi `source='public'`,
      `admin_user_id` luôn `NULL` vì người xem ẩn danh không có tài khoản — dùng chung hàm
      `logArtworkDownload` (`internal/handlers/public_handler.go`)
- [x] Log là thao tác **phụ trợ**: ghi lỗi chỉ log cảnh báo, không chặn việc trả ảnh về —
      cùng nguyên tắc đã áp dụng cho watermark và sinh biến thể ảnh
- [x] Chưa làm giao diện xem lại nhật ký trong admin (quyết định có chủ đích, giữ phạm vi
      gọn) — tra cứu hiện tại bằng SQL trực tiếp vào `artwork_downloads`
- [x] Test service cho `LogDownload` (chuyển đúng dữ liệu xuống repo, lan truyền lỗi từ
      repo, báo lỗi rõ ràng khi chưa cấu hình `downloadRepo`) — `artwork_download_log_test.go`
- [x] `go build ./...`, `go vet ./...`, `go test ./...` đều sạch

### ~~P2.15 — SEO kỹ thuật cơ bản: sitemap.xml, robots.txt, meta động, JSON-LD, OG/Twitter nâng cao~~ ✅ Xong 2026-09-07

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp: trang public trước đây chỉ có
`<title>` tĩnh duy nhất trong `index.html` cho mọi route, không có `robots.txt`/`sitemap.xml`,
không có meta description/canonical/structured data nào — Googlebot crawl vào thấy cùng một
tiêu đề dù ở trang chủ hay trang bảng vàng, và không có cách nào để Google biết hết những URL
cần lập chỉ mục. Đã đánh giá và chọn KHÔNG làm SSR/prerender toàn phần (dự án chạy Go binary +
systemd, không có Node runtime ở production; Googlebot hiện đại render JS đủ tốt cho quy mô một
sự kiện trường học) — chỉ tối ưu crawl bằng sitemap + robots.txt + meta động, dựa trên nền các
tối ưu tải trang đã có (gzip, code-splitting, WebP).

- [x] `GET /sitemap.xml` — liệt kê 4 URL trang public cố định (`/`, `/tac-pham-tieu-bieu`,
      `/phong-trien-lam`, `/bang-vang`) + mọi tác phẩm đã `is_published=true` (trỏ về
      `/chia-se/tac-pham/{id}` đã SSR sẵn, không trỏ `?tranh={id}` để tránh trùng lặp nội
      dung). Origin dùng từ header request (`X-Forwarded-Proto` + `Host`), không thêm biến môi
      trường mới. `Cache-Control: public, max-age=900` — không cần ping Search Console/
      IndexNow, Google tự crawl lại theo lịch khi thấy `<lastmod>` mới.
- [x] `ArtworkService.ListPublishedForSitemap` — method mới, tự lặp trang để lấy TOÀN BỘ tác
      phẩm đã publish (không enrich, chỉ ID + `UpdatedAt`) — tránh chi phí JOIN học sinh/
      trường/giải không cần thiết cho việc build URL list.
- [x] `GET /robots.txt` — `Allow: /`, `Disallow: /admin/ /api/ /auth/`, `Sitemap: <URL tuyệt
      đối>`. Route Go động (không phải file tĩnh) vì cần domain thực tế suy từ request.
- [x] Trang chia sẻ tác phẩm (`/chia-se/tac-pham/{id}`) bổ sung `og:image:width/height`,
      `twitter:image:alt`, và JSON-LD `BreadcrumbList` (Trang chủ → Tác phẩm tiêu biểu → tên
      tác phẩm).
- [x] Hook `usePageMeta` (tự viết, không thêm `react-helmet-async`) cập nhật `<title>`,
      `<meta name="description">`, `<link rel="canonical">` theo từng route public — canonical
      của `/tac-pham-tieu-bieu`/`/phong-trien-lam` luôn cố định không kèm query để tránh Google
      coi mỗi biến thể lọc là một trang riêng.
- [x] Component `<JsonLd>` chèn structured data: `WebSite` (trang chủ), `CollectionPage` (tác
      phẩm tiêu biểu/phòng triển lãm/bảng vàng), `BreadcrumbList` (3 trang danh mục).
- [x] Test service cho `ListPublishedForSitemap` (phân trang lặp tới khi hết, lỗi từ repository
      lan truyền đúng, ép `IsPublished=true`, danh sách rỗng không lỗi).
- [x] `go build ./...`, `go test ./...`, `tsc --noEmit`, `vite build` đều sạch

⚠️ **Chưa làm trong đợt này** — nằm ngoài phạm vi đã chốt, cần hỏi lại nếu muốn triển khai
tiếp: Google Search Console API / IndexNow để chủ động "ping" khi có tác phẩm mới (người dùng
đã chọn rõ chỉ cần sitemap tự động, Google tự crawl lại theo lịch của họ); static
prerender/SSR cho trang chủ nếu sau này đo được qua Search Console là trang bị index chậm hoặc
lỗi crawl thật sự.

### ~~P2.16 — Nâng cấp bảo mật: giả mạo IP, chống tải trọn site, header phòng thủ~~ ✅ Xong 2026-09-07

Không nằm trong lộ trình gốc — làm theo yêu cầu trực tiếp "nâng cấp toàn diện chống bị tấn
công, DDoS, bảo mật, WinHTTrack". Phạm vi đã chốt với người yêu cầu: **chỉ tầng Go** (không
sửa file Nginx), chặn scraper nhưng giữ nguyên SEO của P2.15, và thêm rate limit riêng cho
tải ảnh. Chi tiết ở [detail_design/05-auth-security.md §6-§10](../detail_design/05-auth-security.md),
biến môi trường ở [deploys/02-configuration.md](../deploys/02-configuration.md).

- [x] **Sửa lỗ hổng giả mạo IP** (`middleware/clientip.go`) — `GetClientIP` cũ đọc thẳng
      `X-Forwarded-For` do client gửi, nên bất kỳ ai cũng vô hiệu hoá được **toàn bộ** rate
      limit theo IP bằng một header ngẫu nhiên mỗi request. Đây là lỗ hổng nghiêm trọng nhất
      tìm thấy trong đợt này: nó làm rỗng ruột cả bộ đếm 20 req/phút bảo vệ khu bình luận.
      Nay chỉ tin header khi chặng kết nối nằm trong `TRUSTED_PROXIES`, và duyệt chuỗi
      `X-Forwarded-For` từ **phải sang trái**. Tiện thể sửa luôn lỗi tách địa chỉ IPv6
      (`LastIndex(":")` cắt nhầm `[2001:db8::1]:54321`)
- [x] `ClientIPKey` gom IPv6 về khối `/64` — một thuê bao IPv6 được cấp cả khối, đếm theo
      địa chỉ đầy đủ thì chỉ cần đổi địa chỉ trong cùng khối là có bộ đếm mới
- [x] **Chống tải trọn site** (`middleware/botguard.go`) — ba lớp: User-Agent tự khai công cụ
      tải hàng loạt (HTTrack, wget, curl, Scrapy…) → 403; thiếu User-Agent trên đường HTML →
      403; nhịp duyệt giống máy quét → 429 tạm thời. Tiêu chí quyết định là **số đường dẫn
      khác nhau** trong một phút, không phải tổng số request: người thật xem đi xem lại vài
      trang, máy quét đi qua mỗi URL đúng một lần
- [x] **SEO không đổi** — Googlebot/bingbot/`facebookexternalhit`/coccocbot/Zalo được miễn
      hoàn toàn; `/robots.txt`, `/sitemap.xml`, `/api/v1/health` luôn phục vụ được kể cả với
      User-Agent nằm trong danh sách chặn. Có test khoá lại điều này, vì thêm `"bot"` chung
      chung vào danh sách chặn là sai lầm dễ mắc và sẽ xoá sổ toàn bộ P2.15
- [x] **Rate limit riêng cho tải ảnh gốc** (30/phút, `RATE_LIMIT_DOWNLOAD_REQUESTS`) — tách
      khỏi bộ đếm chung vì đây là thao tác đắt nhất trang public (đọc trọn object S3 + ghi
      `artwork_downloads`) và là đích ngắm chính khi muốn gom tranh hàng loạt
- [x] **Header bảo mật ở tầng ứng dụng** (`middleware/security_headers.go`) — CSP,
      `Permissions-Policy`, COOP/CORP, HSTS có điều kiện. Đặt trong Go chứ không chỉ ở Nginx
      vì file vhost thật trên VPS do Certbot sửa và người vận hành chỉnh tay, repo không kiểm
      soát được. Domain S3 tự suy từ cấu hình S3 sẵn có — bắt khai lại ở biến riêng là mời
      gọi việc quên đồng bộ, mà hậu quả là CSP chặn đúng ảnh tác phẩm
- [x] **Giới hạn body theo đường dẫn** (`middleware/bodylimit.go`) — Nginx đặt
      `client_max_body_size 200m` ở mức server nên endpoint bình luận cũng nhận được body
      200MB; nay còn 1MB (`MAX_JSON_BODY_KB`), riêng đường upload giữ nguyên trần cũ
- [x] **CSRF**: so sánh token bằng `crypto/subtle` (bản cũ so chuỗi thường, lộ độ dài tiền tố
      đúng qua thời gian phản hồi); không còn gọi `r.FormValue` với body multipart — hàm đó
      parse trọn body, nghĩa là file 200MB được ghi ra đĩa tạm **trước** khi kiểm tra token,
      biến chính lớp chống CSRF thành đường làm cạn tài nguyên
- [x] Bộ test mới cho `clientip`/`botguard`/`security_headers`
- [x] `go build ./...`, `go vet ./...`, `go test ./...` đều sạch

⚠️ **Chưa làm trong đợt này** — người yêu cầu chọn rõ phạm vi "chỉ sửa tầng Go", nên các
hạng mục sau vẫn để ngỏ: `limit_req`/`limit_conn` ở Nginx và fail2ban (tuyến biên, chặn
trước khi chạm tiến trình Go); đặt sau Cloudflare để chống DDoS phân tán thật sự; ghi log +
cảnh báo khi một IP tải ảnh bất thường; chặn hotlink ảnh theo `Referer`.

**Bổ sung 2026-09-07 — trả phần frontend mà phạm vi "chỉ sửa tầng Go" đã bỏ lại.** CSP đặt
xong ở tầng Go nhưng `index.html` vẫn còn hai thứ nó cấm, nên trên trình duyệt người dùng
có hai lỗi bị chặn thật sự (console báo, log server im lặng hoàn toàn):

- [x] **Font tự phục vụ** — `style-src 'self'` chặn stylesheet Google Fonts, cả trang tụt về
      font hệ thống. Đã tải Be Vietnam Pro + Fraunces về `web/public/fonts/`, sinh
      `web/src/styles/fonts.css` bằng `scripts/fetch-fonts.sh`. Chọn self-host thay vì nới
      CSP cho `fonts.googleapis.com`: nới CSP đổi lấy việc mỗi lượt xem trang gửi IP của học
      sinh và phụ huynh sang máy chủ bên thứ ba, và vẫn để giao diện phụ thuộc một dịch vụ
      ngoài. `unicode-range` giữ nguyên nên trình duyệt chỉ tải subset nó cần
- [x] **Script gỡ splash ra file riêng** (`web/public/splash.js`) — `script-src 'self'` chặn
      script nội tuyến, nghĩa là splash sẽ **không bao giờ tan** và trang đứng vĩnh viễn ở
      màn hình chờ. Không chọn cách băm `sha256` nhúng hash vào CSP: mỗi lần sửa script phải
      tính lại hash trong code Go, quên một lần là lỗi này quay lại nguyên vẹn
- [x] Loại `/splash.js` khỏi `isHTMLRoute` — tài nguyên tĩnh không cần CSP
- [x] 2 test mới khoá lại: font tự phục vụ phải chạy được (`font-src 'self'`), và CSP không
      đặt lên tài nguyên tĩnh

### ~~P2.17 — Gỡ chunk upload, upload-transaction và nhánh WordPress~~ ✅ Xong 2026-09-07

**Vấn đề.** Ba đường upload cùng tồn tại nhưng chỉ một đường được dùng thật. Đường chunked
giữ phiên trong bộ nhớ tiến trình nên không sống sót qua một lần restart, và giao diện duy
nhất gọi nó — công cụ `/upload` nội bộ — cũng không còn ai mở kể từ khi khu quản trị có
luồng bulk upload riêng. Nhánh WordPress (`/api/v1/wp-upload`, resize và tối ưu ảnh trên đĩa
local) chưa bao giờ chạy ở production.

**Đã làm.**

- [x] Xoá 7 file Go (~2000 dòng): `chunk_handler.go`, `wp_handler.go`, `chunk_upload.go`,
      `image_resize.go`, `image_optimizer.go`, `upload_repository.go`, `upload_record.go`
- [x] Xoá 12 file frontend của công cụ `/upload` (~3000 dòng) và ~2400 dòng CSS đi kèm
- [x] Gỡ cấu hình chết `WORDPRESS_*` + `IMAGE_*` khỏi `builder.go`, `config.go`,
      `.env.example` và systemd `ReadWritePaths` — giữ lại biến mà không code nào đọc chỉ
      khiến người deploy tưởng đang bật một tính năng có thật
- [x] Gỡ ràng buộc `AbsoluteMaxSize ≤ MaxSize × 64`, vốn tồn tại chỉ để giới hạn số chunk

**Hệ quả cần theo dõi.**

- ⚠️ **Mất lớp kiểm tra magic byte** — nó chỉ được gọi ở đường chunked. Xem P1.4 và R13.
- Bảng `uploads` (migration 001) thành bảng chết. Không sửa migration đã commit; dọn bằng
  một migration `DROP TABLE` khi tiện.
- Client cũ gọi các endpoint đã gỡ sẽ nhận 404.

---

## Ưu tiên 3 — Sau sự kiện

Không cần cho lần chạy này, ghi lại để không quên.

| Hạng mục | Ghi chú |
|---|---|
| Kiểm thử cho tầng service/repository | Cần khi codebase còn được phát triển tiếp |
| Nhật ký thao tác admin (phần còn lại) | Đã có phần **tải ảnh** — xem P2.14; ai xoá/sửa tác phẩm lúc nào thì vẫn chưa ghi lại |
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

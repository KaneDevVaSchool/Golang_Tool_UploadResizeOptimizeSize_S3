# 03 — Miền tác phẩm

Miền nghiệp vụ trung tâm: từ file ảnh thô đến tác phẩm hiển thị trên trang public.

**Code chính**: `internal/service/artwork_service.go`, `internal/handlers/artwork_handler.go`,
`internal/repository/artwork_repository.go`

## 1. Vòng đời một tác phẩm

```text
   [ảnh trên máy admin]
            │
            │  ① POST /admin/artworks/bulk-upload
            ▼
   ảnh đã nằm trên S3  ─── chưa có bản ghi DB, chưa ai thấy
            │
            │  ② POST /admin/artworks   (kèm metadata)
            ▼
   artworks(is_published=1, is_featured=0)  ─── hiện ở /phong-trien-lam
            │
            ├─ ③ PATCH .../featured=true  ──▶ hiện thêm ở /tac-pham-tieu-bieu
            ├─ ④ PUT ... {award_ids}      ──▶ hiện thêm ở /bang-vang
            ├─ ⑤ PUT ... {is_published:0} ──▶ ẩn khỏi mọi trang public
            └─ ⑥ DELETE                   ──▶ xoá bản ghi (⚠️ file S3 vẫn còn)
```

Bốn cờ quyết định tác phẩm xuất hiện ở đâu:

| Trang public | Điều kiện |
|---|---|
| `/phong-trien-lam` (phòng triển lãm) | `is_published = 1` |
| `/tac-pham-tieu-bieu` (tiêu biểu) | `is_published = 1` **và** `is_featured = 1` |
| `/bang-vang` (bảng vàng) | `is_published = 1` **và** có bản ghi trong `artwork_awards` |

`is_published` là công tắc chính: đặt `0` là biến mất khỏi **mọi** API public, kể cả khi đã
featured hoặc đã có giải. Mọi handler public đều ép `IsPublished=true` vào filter, không
tin tham số từ client (`public_handler.go:101-105`).

## 2. Vì sao upload chia hai bước

Admin chọn 30 ảnh cùng lúc rồi mới nhập tên tác phẩm/học sinh cho từng ảnh. Nếu gộp một
bước, người dùng phải điền xong 30 form rồi mới bắt đầu tải — chờ lâu, và một lỗi mạng giữa
chừng là mất hết công nhập liệu.

Tách hai bước:

```text
Bước ①  bulk-upload            Bước ②  create (lặp cho từng ảnh)
────────────────────────       ─────────────────────────────────
Ảnh lên S3 song song (5 luồng)  Admin nhập title/tên HS/trường/khối
Trả temp_key + s3_key + s3_url  Gửi kèm s3_key/s3_url từ bước ①
Đọc sẵn width/height            Tạo student + artwork trong 1 transaction
Lỗi từng file không hỏng cả lô  Gán giải (nếu có) sau khi commit
```

`temp_key` do **frontend tự sinh** (id của item preview), không phải ID trong DB. Nó chỉ
làm nhiệm vụ khớp kết quả trả về với đúng ô nhập liệu trên màn hình — backend không lưu
lại. Xem `BulkUploadItem` (`artwork_service.go:19-28`).

⚠️ **Hệ quả**: nếu admin upload xong bước ① rồi đóng trình duyệt, ảnh đã nằm trên S3 mà
không có bản ghi DB — rác S3 không ai biết. Xem [plan/03-risks.md](../plan/03-risks.md).

### Ranh giới transaction ở bước ②

```text
BEGIN
  ├─ INSERT students   (luôn tạo mới, kể cả trùng tên)
  └─ INSERT artworks   (tham chiếu student vừa tạo)
COMMIT
  └─ gán giải (artwork_awards) ← NGOÀI transaction, có chủ đích
```

Gán giải nằm ngoài transaction vì `artwork_awards` có khoá ngoại riêng và **không phải điều
kiện tiên quyết** để tác phẩm tồn tại. Nếu gán giải lỗi, code chỉ ghi log và vẫn trả về
tác phẩm đã tạo thành công (`artwork_service.go:246-250`) — admin gán lại giải sau, thay vì
mất toàn bộ dữ liệu tác phẩm chỉ vì lỗi phụ.

## 3. Cập nhật tác phẩm

`UpdateArtwork` (`artwork_service.go:255`) đọc bản ghi hiện tại → ghi đè các trường → lưu.

**Không đổi được ảnh.** `s3_key`/`s3_url` không nằm trong `UpdateArtworkRequest`. Muốn thay
ảnh phải xoá tác phẩm và tạo mới. Chủ đích: tránh trường hợp bản ghi trỏ tới ảnh này nhưng
lượt xem/bình luận lại thuộc về ảnh cũ.

**Ngữ nghĩa ba trạng thái của `AwardIDs`** (`artwork_service.go:76-80`):

| Giá trị | Ý nghĩa |
|---|---|
| `nil` (không gửi trường trong JSON) | Giữ nguyên giải hiện tại |
| `[]int64{}` (mảng rỗng) | Gỡ hết giải |
| Danh sách khác rỗng | Gỡ giải cũ rồi gắn **toàn bộ** danh sách này |

Một tác phẩm có thể nhận nhiều giải cùng lúc (vd giải chính Nhất/Nhì/Ba + giải Đặc biệt
phụ) — `ArtworkMetaForm` cho chọn nhiều giải qua checkbox (`awardIds: number[]`), không còn
giới hạn một giải/tác phẩm như bản UI ban đầu. Giới hạn cũ (single-select) chưa từng nằm ở
schema: `artwork_awards` (migration 009) vốn đã là bảng nối N:N; giới hạn chỉ do UI cũ chọn
1 giải áp lên request. Phân biệt "không gửi trường" với "gửi mảng rỗng" dựa vào hành vi có
sẵn của `encoding/json` trên slice thường (`AwardIDs []int64` trong `updateArtworkBody`,
`artwork_handler.go:221`): key vắng mặt trong JSON giữ `AwardIDs` là `nil`, còn `"award_ids":
[]` giải mã ra slice rỗng khác `nil` — không cần kiểu con trỏ.

⚠️ `UpdateArtwork` sửa `school_id`/`grade_level_id` trên bảng `artworks` nhưng **không đồng
bộ ngược** về `students`. Hai bảng có thể lệch nhau. Chi tiết trong
[01-database.md](./01-database.md).

## 4. Xoá tác phẩm

```go
func (s *artworkService) DeleteArtwork(ctx, id) error {
    artwork, _ := s.artworkRepo.GetByID(ctx, id)
    for _, key := range collectArtworkS3Keys(artwork, s.uploadService.ObjectKeyFromURL) {
        s.uploadService.DeleteObject(ctx, key)   // ảnh gốc + mọi biến thể/thumbnail
    }
    return s.artworkRepo.Delete(ctx, id)
}
```

Xoá bản ghi `artworks` sẽ **CASCADE** xoá reaction/comment/view/artwork_awards liên quan.
File trên S3 **cũng bị xoá** — đảo ngược quyết định ban đầu (từng cố ý giữ lại S3 để tránh
mất file gốc do bấm nhầm nút xoá). `collectArtworkS3Keys` (`artwork_service.go`) gom key ảnh
gốc (`S3Key`) và mọi URL trong `Variants`/`ThumbnailURL`, dịch ngược URL → key bằng
`utils.ParseS3ObjectKey` (đối chiếu bucket đã cấu hình, hỗ trợ cả dạng virtual-host,
path-style và endpoint tuỳ chỉnh) trước khi gọi `s3Repo.Delete` cho từng key — thứ tự xoá S3
**trước** DB row để nếu xoá dở giữa chừng, còn s3_key trong DB mà tra lại được, không mồ côi
ngược. Có test khoá hành vi thu thập key ở `artwork_delete_test.go`
(`TestCollectArtworkS3Keys`).

### Xoá hàng loạt

`DeleteArtworkBatch` (`artwork_service.go`) phục vụ thao tác chọn nhiều ở trang danh sách
quản trị — gọi `DeleteArtwork` **tuần tự cho từng id**, không phải 1 câu SQL `IN (...)` như
`SetFeaturedBatch`, vì mỗi tác phẩm còn cần xoá kèm object S3 riêng (không gộp được thành 1
lệnh). Tác phẩm nào xoá S3 lỗi (mất mạng, key đã mất...) thì **giữ nguyên bản ghi DB của
riêng nó** và tiếp tục sang tác phẩm kế tiếp, thay vì để 1 lỗi chặn cả lô — admin chọn 50
ảnh xoá, 1 ảnh lỗi mạng không nên khiến 49 ảnh còn lại cũng không xoá được. Response trả số
lượng đã xoá thành công (`deleted`) so với số lượng yêu cầu (`requested`); `deleted <
requested` nghĩa là còn tác phẩm chưa xoá được, cần chọn lại đúng chúng để thử lại. Có test
khoá hành vi này ở `artwork_bulk_delete_test.go`
(`TestDeleteArtworkBatchMotItemLoiKhongChanCaLo`).

## 5. Lọc và tìm kiếm

`ArtworkFilter` (`models/artwork.go:29`) dùng chung cho **cả admin và public** — khác biệt
duy nhất là public luôn ép `IsPublished=true`.

| Trường | Cách dịch sang SQL |
|---|---|
| `Search` | `(artworks.title LIKE ? OR students.full_name LIKE ?)` — thêm JOIN `students` |
| `SchoolID` | `artworks.school_id = ?` |
| `Region` | `artworks.school_id IN (SELECT id FROM schools WHERE region = ?)`; `models.IsKnownRegion` chặn giá trị lạ trước khi vào SQL |
| `GradeLevelID` | `artworks.grade_level_id = ?` |
| `EducationLevel` | `grade_level_id IN (SELECT id FROM grade_levels WHERE education_level = ?)` |
| `TopicCategoryID` | `artworks.topic_category_id = ?` |
| `AwardID` | `id IN (SELECT artwork_id FROM artwork_awards WHERE award_id = ?)` |
| `IsFeatured` / `IsPublished` | So sánh trực tiếp |

Ba điểm về thiết kế truy vấn:

**JOIN có điều kiện.** `students` chỉ được JOIN khi có `Search` — các truy vấn khác không
gánh chi phí thừa (`artwork_repository.go:181`).

**Dùng `LIKE` chứ không `FULLTEXT`.** Index `FULLTEXT(title)` đã tồn tại nhưng không dùng.
Lý do ghi tại `artwork_repository.go:172-175`: tokenizer FULLTEXT của MySQL với tiếng Việt
có dấu cho kết quả không đáng tin. `LIKE '%...%'` chậm hơn nhưng đúng, và với quy mô vài
nghìn tác phẩm thì chênh lệch không đáng kể. Đây là đánh đổi có ý thức, không phải bỏ sót.

**Phân trang có trần cứng.** `pageSize` bị kẹp tối đa **100** ở tầng repository
(`artwork_repository.go:238-240`), bất kể client gửi gì. Chặn việc một request kéo toàn bộ
bảng.

Sắp xếp cố định `ORDER BY artworks.created_at DESC` — mới nhất trước, chưa cho tuỳ chỉnh.

## 6. Enrich — tránh N+1

`ArtworkWithMeta` gộp tác phẩm với dữ liệu từ nhiều bảng khác. Nếu truy vấn ngây thơ, hiển
thị 24 tác phẩm sẽ tốn hàng trăm truy vấn.

`enrichArtworks()` (`artwork_service.go`) gom lại thành **truy vấn theo lô**:

```text
1 query  → awards theo danh sách artwork_id     (ListByArtworkIDs)
1 query  → đếm reaction theo lô                 (CountByArtworkBatch)
1 query  → đếm comment theo lô                  (CountByArtworkBatch)
1 query  → toàn bộ schools          (ít dòng, nạp hết rồi map trong bộ nhớ)
1 query  → toàn bộ grade_levels     (12 dòng, tương tự)
1 query  → toàn bộ topic_categories (vài chục dòng, tương tự)
N query  → student theo từng artwork   ⚠️ CHƯA gom lô
```

⚠️ **Vòng lặp student còn sót lại**: mỗi tác phẩm vẫn tốn một truy vấn lấy tên học sinh. Với
`page_size` 24 là 24 truy vấn thêm mỗi lần tải trang. Đây là nợ kỹ thuật đã xác định — cần
thêm `StudentRepository.GetByIDs()`. Xem [plan/02-roadmap.md](../plan/02-roadmap.md).

`schools`, `grade_levels`, và `topic_categories` được nạp **toàn bộ** rồi map trong bộ nhớ
vì chúng là danh mục tĩnh rất nhỏ (5, 12, và vài chục dòng) — rẻ hơn nhiều so với JOIN hay
truy vấn theo lô. Cache dùng chung một TTL (`refCacheTTL`, 5 phút) cho cả ba.

## 7. Bảng vàng

`HandleBillboard` (`public_handler.go:615`) dựng danh sách theo thứ hạng giải:

```text
lấy danh sách awards (đã sắp theo rank_order)
  cho mỗi award:
      ListArtworks(filter{AwardID: award.ID, IsPublished: true, PageSize: 100})
      gắn thông tin award vào từng tác phẩm
```

Thứ tự kết quả bám theo `awards.rank_order` — Giải Nhất trước, Khuyến khích sau — vì vòng
lặp chạy theo thứ tự danh sách giải.

⚠️ **Hai hạn chế**: (a) mỗi giải là một lần gọi `ListArtworks` kèm enrich riêng, số truy
vấn tăng tuyến tính theo số giải; (b) trần 100 tác phẩm mỗi giải, vượt quá sẽ bị cắt âm
thầm. Với quy mô hội thi hiện tại thì chấp nhận được, nhưng cần biết giới hạn.

### `rank_order` là 0-based, do trang admin/awards quyết định

Backend chỉ lưu và sắp theo `rank_order` thô — không có ràng buộc DB nào ép nó bắt đầu từ 0
hay 1. Giá trị thực tế do trang `/admin/awards` ghi xuống: kéo-thả đổi thứ tự hiển thị, vị
trí trong danh sách MỚI là dữ liệu, `rank_order` chỉ là hệ quả tính ra khi lưu — **0-based**
(giải đứng đầu danh sách có `rank_order = 0`, xem `AwardsPage.tsx:249-278`).

Hai trang public suy luận "đây là hạng Nhất/Nhì/Ba hay giải chuyên đề" từ `rank_order` để
tô màu bục/podium: `HallOfFamePage.tsx` (`tierOf()`) và `FeaturedArtworksPage.tsx`
(`podiumRank()`). Cả hai đều phải cộng 1 để quy đổi `rank_order` 0-based sang bậc 1-based
(`position = rank_order + 1`) trước khi so `position <= 3`. Thiếu bước +1 này từng khiến
giải Nhì (`rank_order=1`) hiện thành hạng Nhất, giải Ba (`rank_order=2`) hiện thành hạng
Nhì — lỗi đã sửa (2026-09-06). `cmd/seed/data.go` cũng từng seed `rank_order` bắt đầu từ 1
kèm "Giải Đặc biệt" chen trước "Giải Nhất", không khớp quy ước trên; đã sửa lại 0-based với
"Giải Nhất" đứng đầu.

Cả hai hàm còn có nhánh dự phòng: nếu `position` nằm ngoài 1..3, đoán bậc theo `slug`/`name`
chứa "nhat"/"nhi"/"giai ba" — dữ liệu cũ có award chưa gán `rank_order` chuẩn nhưng tên vẫn
đúng "Giải Nhất/Nhì/Ba". Đổi thứ tự giải ở `/admin/awards` sau này vẫn an toàn miễn `rank_order`
tiếp tục là 0-based liên tục; nếu sau này đổi quy ước (vd cho phép số âm, hoặc không liên
tục), phải sửa đồng thời cả hai hàm suy luận bậc này.

## 8. Bề mặt API của miền này

### Admin (yêu cầu session)

| Method | Đường dẫn | Việc |
|---|---|---|
| POST | `/api/v1/admin/artworks/bulk-upload` | Bước ① — đẩy nhiều ảnh lên S3 |
| POST | `/api/v1/admin/artworks` | Bước ② — tạo bản ghi kèm metadata |
| GET | `/api/v1/admin/artworks` | Danh sách có lọc/phân trang |
| GET | `/api/v1/admin/artworks/{id}` | Chi tiết |
| PUT | `/api/v1/admin/artworks/{id}` | Cập nhật metadata |
| DELETE | `/api/v1/admin/artworks/{id}` | Xoá bản ghi |
| DELETE | `/api/v1/admin/artworks/bulk-delete` | Xoá hàng loạt — `{ids}`, xoá tuần tự từng tác phẩm (kèm S3), 1 lỗi không chặn cả lô |
| GET | `/api/v1/admin/artworks/{id}/download` | Tải ảnh gốc, không watermark, không ép `is_published`, ghi nhật ký vào `artwork_downloads`. Chỉ có ở khu quản trị — không có phiên bản public (đã gỡ, xem [03-risks.md](../plan/03-risks.md)) |
| PATCH | `/api/v1/admin/artworks/{id}/featured` | Bật/tắt tiêu biểu (1 tác phẩm) |
| PATCH | `/api/v1/admin/artworks/bulk-featured` | Bật/tắt tiêu biểu hàng loạt — `{ids, featured}`, 1 câu `UPDATE ... WHERE id IN (...)` |

`ParseMultipartForm(maxUploadSize × 20)` ở bulk-upload (`artwork_handler.go:55`) cho phép
lô lớn nằm trong bộ nhớ trước khi ghi đĩa — với mặc định 20MB là 400MB bộ nhớ đệm.
⚠️ Cần cân nhắc lại nếu RAM của VPS eo hẹp.

### Public (không cần đăng nhập)

| Method | Đường dẫn | Việc |
|---|---|---|
| GET | `/api/v1/public/artworks` | Danh sách đã publish |
| GET | `/api/v1/public/artworks/featured` | Tiêu biểu, lọc theo `region` |
| GET | `/api/v1/public/artworks/{id}` | Chi tiết + ghi nhận lượt xem |
| GET | `/api/v1/public/billboard` | Bảng vàng |
| GET | `/api/v1/awards` | Danh mục giải thưởng đang hoạt động |
| GET | `/api/v1/topic-categories` | Danh mục nhóm chủ đề sáng tạo đang hoạt động |

`ArtworkService.ListPublishedForSitemap` không phải REST endpoint riêng — dùng nội bộ bởi
`PublicHandler.HandleSitemap` (`GET /sitemap.xml`, xem [API.md](../API.md)) để liệt kê ID +
thời điểm cập nhật của mọi tác phẩm đã publish, không enrich (không cần tên học sinh/trường/
giải), tự lặp trang để vượt trần `page_size=100` của `ArtworkRepository.List`.

### Giải thưởng và nhóm chủ đề (admin, yêu cầu session)

CRUD giống nhau về hình dạng, tách 2 handler/service/repository riêng vì hai domain độc lập
(`internal/handlers/award_handler.go`, `internal/handlers/topic_category_handler.go`):

| Method | Đường dẫn | Việc |
|---|---|---|
| GET | `/api/v1/admin/awards` | Danh sách (gồm cả giải đã tắt) |
| POST | `/api/v1/admin/awards` | Tạo giải mới |
| PUT | `/api/v1/admin/awards/{id}` | Cập nhật |
| DELETE | `/api/v1/admin/awards/{id}` | Xoá — 409 nếu đang gắn cho tác phẩm |
| GET | `/api/v1/admin/topic-categories` | Danh sách (gồm cả nhóm đã tắt) |
| POST | `/api/v1/admin/topic-categories` | Tạo nhóm mới |
| PUT | `/api/v1/admin/topic-categories/{id}` | Cập nhật |
| DELETE | `/api/v1/admin/topic-categories/{id}` | Xoá — 409 nếu đang gắn cho tác phẩm |

Lọc theo `region` làm ở **tầng SQL** từ P1.3 (xem [02-roadmap.md](../plan/02-roadmap.md)):
`HandleListFeatured` (`public_handler.go:149-172`) gán `filter.Region` thẳng vào
`ArtworkFilter`, dịch thành `school_id IN (SELECT id FROM schools WHERE region = ?)` trong
`artworkRepository.List` — không còn cắt 100 bản ghi rồi lọc trong bộ nhớ như thiết kế ban
đầu. `models.IsKnownRegion` chặn giá trị lạ trước khi vào SQL
(`public_handler.go:163`).

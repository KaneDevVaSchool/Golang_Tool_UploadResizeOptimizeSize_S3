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
            ├─ ④ PUT ... {award_id}       ──▶ hiện thêm ở /bang-vang
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

**Ngữ nghĩa ba trạng thái của `AwardID`** (`artwork_service.go:65`):

| Giá trị | Ý nghĩa |
|---|---|
| `nil` (không gửi trường) | Giữ nguyên giải hiện tại |
| `0` | Gỡ hết giải |
| `> 0` | Gỡ giải cũ rồi gắn giải này |

Việc "gỡ hết rồi gắn lại" phản ánh giới hạn của UI (một giải mỗi tác phẩm) trên một schema
vốn hỗ trợ N:N. Nếu sau này UI cho nhiều giải, chỗ này phải sửa lại.

⚠️ `UpdateArtwork` sửa `school_id`/`grade_level_id` trên bảng `artworks` nhưng **không đồng
bộ ngược** về `students`. Hai bảng có thể lệch nhau. Chi tiết trong
[01-database.md](./01-database.md).

## 4. Xoá tác phẩm

```go
func (s *artworkService) DeleteArtwork(ctx, id) error {
    return s.artworkRepo.Delete(ctx, id)   // chỉ xoá DB
}
```

Xoá bản ghi `artworks` sẽ **CASCADE** xoá reaction/comment/view/artwork_awards liên quan.
File trên S3 **cố ý giữ lại**.

Lý do (ghi trong `artwork_service.go:89-92`): bấm nhầm nút xoá là chuyện thường; mất bản ghi
DB có thể nhập lại, mất file gốc của học sinh thì không. Đánh đổi: S3 tích rác theo thời
gian, cần quy trình dọn định kỳ có đối chiếu.

## 5. Lọc và tìm kiếm

`ArtworkFilter` (`models/artwork.go:29`) dùng chung cho **cả admin và public** — khác biệt
duy nhất là public luôn ép `IsPublished=true`.

| Trường | Cách dịch sang SQL |
|---|---|
| `Search` | `(artworks.title LIKE ? OR students.full_name LIKE ?)` — thêm JOIN `students` |
| `SchoolID` | `artworks.school_id = ?` |
| `GradeLevelID` | `artworks.grade_level_id = ?` |
| `EducationLevel` | `grade_level_id IN (SELECT id FROM grade_levels WHERE education_level = ?)` |
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

`ArtworkWithMeta` gộp tác phẩm với dữ liệu từ 6 bảng khác. Nếu truy vấn ngây thơ, hiển thị
24 tác phẩm sẽ tốn ~150 truy vấn.

`enrichArtworks()` (`artwork_service.go:354`) gom lại thành **truy vấn theo lô**:

```text
1 query  → awards theo danh sách artwork_id     (ListByArtworkIDs)
1 query  → đếm reaction theo lô                 (CountByArtworkBatch)
1 query  → đếm comment theo lô                  (CountByArtworkBatch)
1 query  → toàn bộ schools      (ít dòng, nạp hết rồi map trong bộ nhớ)
1 query  → toàn bộ grade_levels (12 dòng, tương tự)
N query  → student theo từng artwork   ⚠️ CHƯA gom lô
```

⚠️ **Vòng lặp student còn sót lại** (`artwork_service.go:397`): mỗi tác phẩm vẫn tốn một
truy vấn lấy tên học sinh. Với `page_size` 24 là 24 truy vấn thêm mỗi lần tải trang. Đây là
nợ kỹ thuật đã xác định — cần thêm `StudentRepository.GetByIDs()`. Xem
[plan/02-roadmap.md](../plan/02-roadmap.md).

`schools` và `grade_levels` được nạp **toàn bộ** rồi map trong bộ nhớ vì chúng là danh mục
tĩnh rất nhỏ (5 và 12 dòng) — rẻ hơn nhiều so với JOIN hay truy vấn theo lô.

## 7. Bảng vàng

`HandleBillboard` (`public_handler.go:493`) dựng danh sách theo thứ hạng giải:

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
| PATCH | `/api/v1/admin/artworks/{id}/featured` | Bật/tắt tiêu biểu |

`ParseMultipartForm(maxUploadSize × 20)` ở bulk-upload (`artwork_handler.go:44`) cho phép
lô lớn nằm trong bộ nhớ trước khi ghi đĩa — với mặc định 20MB là 400MB bộ nhớ đệm.
⚠️ Cần cân nhắc lại nếu RAM của VPS eo hẹp.

### Public (không cần đăng nhập)

| Method | Đường dẫn | Việc |
|---|---|---|
| GET | `/api/v1/public/artworks` | Danh sách đã publish |
| GET | `/api/v1/public/artworks/featured` | Tiêu biểu, lọc theo `region` |
| GET | `/api/v1/public/artworks/{id}` | Chi tiết + ghi nhận lượt xem |
| GET | `/api/v1/public/billboard` | Bảng vàng |

⚠️ **Lọc theo `region` làm ở tầng ứng dụng, không phải SQL.** `HandleListFeatured` lấy về
tối đa 100 tác phẩm featured rồi lọc trong bộ nhớ theo `region`
(`public_handler.go:158-168`), vì `ArtworkFilter` chỉ có `SchoolID` chứ không có `Region`.
Đúng với quy mô hiện tại (số tác phẩm tiêu biểu ít), nhưng nếu vượt 100 thì kết quả sẽ
thiếu **trước khi** lọc region. Ghi chú này có trong code tại `public_handler.go:112-116`.

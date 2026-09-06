# 01 — Thiết kế cơ sở dữ liệu

**Hệ quản trị**: MySQL 8+ · InnoDB · `utf8mb4` / `utf8mb4_unicode_ci`
**Driver**: `github.com/go-sql-driver/mysql`
**Migration**: `internal/database/migrations/001..014_*.sql`, chạy tự động lúc khởi động

> `utf8mb4` là bắt buộc, không phải tuỳ chọn: dữ liệu chứa tiếng Việt có dấu và emoji
> (bình luận, tên tác phẩm). `utf8` của MySQL chỉ 3 byte và sẽ làm hỏng emoji.

## Sơ đồ quan hệ

```text
   admin_users ─────1:N────▶ admin_sessions
        │                     (id = chính token, xoá theo CASCADE)
        │ created_by (SET NULL)
        ▼
   ┌─────────────────────────────────────────────────┐
   │                  artworks                       │
   │  bảng trung tâm                                 │
   └─┬────┬────┬─────────┬─────────┬─────────┬───────┘
     │    │    │         │         │         │
     │    │    │         │         │         └──N:1──▶ uploads (SET NULL)
     │    │    │         │         │
     │    │    │         │         └──1:N──▶ artwork_views      (CASCADE)
     │    │    │         └──1:N──▶ artwork_comments             (CASCADE)
     │    │    └──1:N──▶ artwork_reactions                      (CASCADE)
     │    │
     │    └──N:N──▶ artwork_awards ──N:1──▶ awards              (CASCADE cả hai)
     │
     └──N:1──▶ students ──N:1──▶ schools
                       └──N:1──▶ grade_levels
```

Chính sách xoá được chọn có chủ đích:

| Quan hệ | Chính sách | Lý do |
|---|---|---|
| `artworks` → tương tác (reaction/comment/view) | `CASCADE` | Xoá tranh thì cảm xúc/bình luận về nó vô nghĩa |
| `artworks` → `uploads` | `SET NULL` | Lịch sử upload là bản ghi audit, giữ lại dù tranh đã xoá |
| `artworks` → `admin_users` (created_by) | `SET NULL` | Xoá tài khoản admin không được xoá tác phẩm họ đã đăng |
| `artworks` → `students`/`schools`/`grade_levels` | **Không** cascade (RESTRICT mặc định) | Chặn xoá nhầm một trường đang có tác phẩm |
| `admin_users` → `admin_sessions` | `CASCADE` | Vô hiệu hoá tài khoản là phải cắt mọi phiên |

## Bảng theo nhóm

### Nhóm hạ tầng upload

#### `uploads` — lịch sử upload (migration 001)

Ghi nhận mọi lần upload qua `/api/v1/upload-transaction`, dùng cho audit và đối soát.

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | BIGINT UNSIGNED AI | PK |
| `filename` / `original_name` | VARCHAR(255) | Tên sau chuẩn hoá / tên gốc người dùng gửi |
| `file_size` | BIGINT | Byte |
| `content_type` | VARCHAR(100) | Suy ra từ đuôi file |
| `s3_key` / `s3_url` | VARCHAR(500) / (1000) | NULL khi còn `pending` |
| `status` | VARCHAR(20) | `pending` → `completed` \| `failed` |
| `error` | TEXT | Thông điệp lỗi khi `failed` |

**Index**: `status`, `created_at DESC`, `s3_key`.

Trạng thái chuyển theo đúng transaction: bản ghi `pending` được tạo *trước* khi đẩy S3,
cập nhật `completed`/`failed` sau. Nếu tiến trình chết giữa chừng, bản ghi kẹt ở `pending` —
đó là tín hiệu để dò file mồ côi trên S3.

### Nhóm danh mục (seed sẵn)

#### `schools` — 5 cơ sở, 3 khu vực (migration 004)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `name` | VARCHAR(255) | Tên cơ sở |
| `region` | ENUM(`saigon`,`cantho`,`vungtau`) | **3 khu vực trưng bày** |
| `display_order` | INT | Thứ tự hiển thị |
| `is_active` | TINYINT(1) | Ẩn cơ sở không tham gia |

Dữ liệu seed sẵn trong migration:

| Cơ sở | Khu vực |
|---|---|
| Bình Thới - Tân Bình | `saigon` |
| Thống Tây Hội | `saigon` |
| Phú Định | `saigon` |
| Vũng Tàu | `vungtau` |
| Cần Thơ | `cantho` |

⚠️ **5 cơ sở vật lý ≠ 3 khu vực trưng bày.** Ba cơ sở tại TP.HCM gộp thành một khu vực
`saigon` theo yêu cầu ban tổ chức. Khi đọc code thấy `region`, đó là *khu vực trưng bày*,
không phải địa điểm. Trang public dùng `region` để chia tab; admin dùng `school_id` để
nhập liệu chính xác cơ sở.

#### `grade_levels` — 12 khối, 2 cấp (migration 005)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `education_level` | ENUM(`primary`,`secondary`) | Tiểu học / Trung học |
| `grade_number` | TINYINT UNSIGNED | 1–12 |
| `label` | VARCHAR(50) | "Khối 1"… "Khối 12" |

**Unique**: `(education_level, grade_number)`.

Seed: khối 1–5 = `primary`, khối 6–12 = `secondary`. Lưu ý `secondary` gộp **cả THCS và
THPT** — hệ thống chỉ phân hai cấp, không tách ba.

#### `awards` — cấu hình giải thưởng (migration 008)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `name` | VARCHAR(100) | "Giải Nhất"… |
| `slug` | VARCHAR(100) UNIQUE | Định danh ổn định |
| `rank_order` | INT | Thứ tự xếp hạng, nhỏ = cao |
| `color_hex` | VARCHAR(7) | Màu badge, mặc định `#c49c57` |
| `icon_key` | VARCHAR(50) | Khoá icon FE tự map |
| `is_active` | TINYINT(1) | Ẩn giải không dùng |

Không seed — admin tự tạo qua `/admin/awards`. `rank_order` quyết định thứ tự bảng vàng.

### Nhóm nghiệp vụ chính

#### `students` — học sinh (migration 006)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `full_name` | VARCHAR(255) | Họ tên |
| `school_id` / `grade_level_id` | FK | Tại thời điểm nộp bài |
| `class_name` | VARCHAR(100) NULL | Ví dụ "5A2" |

⚠️ **Không phải hệ thống tài khoản học sinh.** Mỗi lần tạo tác phẩm sẽ tạo **một bản ghi
student mới**, kể cả trùng tên (`artwork_service.go:204`). Không khử trùng lặp. Hệ quả:
một học sinh nộp 3 bài sẽ có 3 dòng trong `students`. Đây là lựa chọn có chủ đích — hội thi
không có định danh học sinh đáng tin (không mã học sinh, tên trùng nhiều), gộp nhầm hai
em cùng tên còn tệ hơn để trùng lặp.

#### `artworks` — bảng trung tâm (migration 007)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `title` | VARCHAR(255) | Tên tác phẩm |
| `student_id` | FK | Tác giả |
| `school_id`, `grade_level_id` | FK | **Denormalize** từ `students` |
| `s3_key` | VARCHAR(500) | Không lộ ra JSON (`json:"-"`) |
| `s3_url` | VARCHAR(1000) | Trả FE dưới tên `image_url` |
| `thumbnail_url` | VARCHAR(1000) NULL | Trỏ `thumb_jpg`; `NULL` với ảnh cũ hoặc ảnh gốc nhỏ hơn cỡ thumb |
| `variants` | JSON NULL | Migration `013` — map `"<cỡ>_<định dạng>"` → URL |
| `file_size`, `width`, `height` | | `width`/`height` đọc từ ảnh lúc upload |
| `is_featured` | TINYINT(1) | Hiện ở trang "Tác phẩm tiêu biểu" |
| `is_published` | TINYINT(1) | `0` = ẩn khỏi mọi API public |
| `view_count` | BIGINT UNSIGNED | Đếm dồn, chống trùng qua `artwork_views` |
| `upload_id`, `created_by` | FK NULL | Truy vết nguồn gốc |

**Index**: `school_id`, `grade_level_id`, `is_featured`, `is_published`, và
`FULLTEXT(title)`.

Hai điểm thiết kế quan trọng:

**Denormalize `school_id`/`grade_level_id`.** Có thể suy ra qua `students` nhưng vẫn lưu
trực tiếp, vì dashboard đếm theo khu vực/khối là truy vấn chạy thường xuyên nhất và
JOIN qua `students` mỗi lần là lãng phí. Giá phải trả: nếu sửa trường/khối của học sinh thì
phải cập nhật cả hai bảng — `UpdateArtwork` hiện cập nhật trên `artworks`
(`artwork_service.go:264-270`), không đồng bộ ngược về `students`. ⚠️ Đây là điểm cần lưu ý
khi thêm tính năng sửa thông tin học sinh.

**`FULLTEXT(title)` chưa được dùng.** Index đã tạo nhưng `ArtworkFilter.Search` hiện thực
hiện bằng `LIKE` (xem ghi chú tại `models/artwork.go:31`). Chuyển sang `MATCH...AGAINST`
là cải tiến sẵn sàng làm khi lượng tác phẩm tăng.

**Cột `variants` dùng JSON, không phải 6 cột riêng** (migration `013`). Lý do: số biến thể
mỗi ảnh **không cố định** — ảnh gốc nhỏ hơn cỡ đích thì cỡ đó không được sinh (`skipVariant`
trong `internal/service/image_variants.go`) — và bộ cỡ còn có thể đổi về sau mà không phải
migrate lại schema.

Dạng dữ liệu: `{"thumb_webp":"https://…","thumb_jpg":"https://…", …}`.

`ArtworkVariants` (`models/artwork.go:60`) cài `Scan`/`Value` để đọc-ghi cột JSON. `NULL`
(tác phẩm cũ chưa sinh biến thể) trả về map rỗng **chứ không phải lỗi** — phía hiển thị đã
có đường lui về ảnh gốc. `thumbnail_url` vẫn được ghi song song để code cũ và trang admin
không gãy.

**Index `(is_published, created_at DESC)`** (migration `014`). Mọi truy vấn công khai đều
lọc `is_published = 1` rồi `ORDER BY created_at DESC`, nhưng `created_at` trước đó không có
index nào. MySQL vì thế chọn `idx_artworks_is_published` — gần như toàn bảng đều bằng 1 nên
độ chọn lọc gần bằng không — rồi filesort lại toàn bộ kết quả cho **mỗi trang**. Composite
này vừa lọc vừa cho sẵn thứ tự, nên MySQL đọc đúng số dòng của trang rồi dừng.

#### `artwork_awards` — nối N:N (migration 009)

| Cột | Ghi chú |
|---|---|
| `artwork_id`, `award_id` | FK, `CASCADE` cả hai chiều |
| `awarded_at` | DATETIME NULL |

**Unique**: `(artwork_id, award_id)` — không gán trùng một giải hai lần.

UI hiện chỉ cho một giải mỗi tác phẩm, nhưng schema thiết kế N:N để mở rộng (một tác phẩm
nhận nhiều giải ở nhiều hạng mục). Service phản ánh giới hạn UI bằng cách gỡ hết giải cũ
trước khi gắn giải mới (`artwork_service.go:275-291`).

### Nhóm tương tác ẩn danh

Ba bảng dưới đây định danh người dùng bằng `visitor_token` — UUID sinh ở trình duyệt, lưu
`localStorage`. **Đây không phải xác thực.** Xoá localStorage là mất quyền sửa/xoá bình
luận của chính mình. Chấp nhận được vì yêu cầu nghiệp vụ là tương tác không rào cản.

#### `artwork_reactions` (migration 010)

| Cột | Kiểu |
|---|---|
| `reaction_type` | ENUM(`like`,`love`,`haha`,`wow`,`sad`,`angry`) |
| `visitor_token` | VARCHAR(64) |
| `ip_address` | VARCHAR(45) — đủ chứa IPv6 |

**Unique**: `(artwork_id, visitor_token, reaction_type)`.

Ràng buộc này cho phép **một người thả nhiều loại cảm xúc khác nhau** trên cùng tác phẩm,
nhưng mỗi loại chỉ một lần. Repository dùng upsert nên bấm lại không tạo bản ghi mới.

`ip_address` chỉ dùng cho rate-limit/audit nội bộ, **không bao giờ trả ra JSON**
(`models/reaction.go` đánh `json:"-"`).

#### `artwork_comments` (migration 011)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `display_name` | VARCHAR(100) | Người xem tự nhập |
| `content` | VARCHAR(1000) | Giới hạn khớp hằng số ở handler |
| `is_hidden` | TINYINT(1) | Ẩn spam mà không xoá hẳn |

**Index**: `artwork_id`, `created_at DESC`, và `(artwork_id, is_hidden, created_at DESC)`
(migration `014`). Truy vấn thật là `WHERE artwork_id = ? AND is_hidden = 0 ORDER BY
created_at DESC`; với hai index rời rạc, MySQL chỉ dùng được một rồi filesort phần còn lại.
Composite phủ trọn cả ba mệnh đề.

Giới hạn độ dài được **đồng bộ hai nơi**: cột VARCHAR ở đây và hằng số
`maxCommentContentLength = 1000` / `maxDisplayNameLength = 100` ở
`public_handler.go:21-25`. Sửa một nơi phải sửa nơi kia, nếu không handler sẽ nhận dữ liệu
mà MySQL từ chối.

⚠️ `is_hidden` đã có trong schema và repository lọc theo nó, nhưng **chưa có API admin để
bật/tắt**. Kiểm duyệt hiện phải làm bằng SQL thủ công. Xem [plan/02-roadmap.md](../plan/02-roadmap.md).

#### `artwork_views` (migration 012)

| Cột | Kiểu |
|---|---|
| `artwork_id`, `visitor_token`, `viewed_at` | |

**Index**: `(artwork_id, visitor_token, viewed_at)` — composite, đúng thứ tự cho truy vấn
"trình duyệt này đã xem tranh này trong 24h qua chưa".

Bảng này tồn tại chỉ để chống thổi phồng lượt xem. Trước khi tăng `artworks.view_count`,
service kiểm tra bảng này trước. Bảng sẽ phình theo thời gian — ⚠️ cần job dọn bản ghi cũ
hơn 24h, hiện **chưa có** (xem [plan/03-risks.md](../plan/03-risks.md)).

### Nhóm xác thực

#### `admin_users` (migration 002)

| Cột | Ghi chú |
|---|---|
| `google_sub` | VARCHAR(255) UNIQUE — định danh ổn định từ Google |
| `email` | VARCHAR(255) UNIQUE |
| `name`, `avatar_url` | Từ hồ sơ Google |
| `role` | Mặc định `admin` — 📋 chưa phân quyền theo vai trò |
| `is_active` | Vô hiệu hoá không cần xoá |
| `last_login_at` | |

Không có cột mật khẩu — đăng nhập hoàn toàn qua Google. `google_sub` là khoá định danh
thật (email có thể đổi), nhưng cả hai đều unique.

#### `admin_sessions` (migration 003)

| Cột | Ghi chú |
|---|---|
| `id` | VARCHAR(64) PK — **chính là token** đặt trong cookie HttpOnly |
| `admin_user_id` | FK, `CASCADE` |
| `expires_at` | Dọn định kỳ mỗi giờ |

Token 32 byte ngẫu nhiên, mã base64 url-safe (`auth/session.go:33-38`). Lưu DB thay vì
JWT để **thu hồi được ngay**: vô hiệu hoá một admin là xoá dòng, phiên chết lập tức —
JWT sẽ vẫn hợp lệ đến khi hết hạn.

## Cơ chế migration

Hai đường chạy migration, dùng chung định dạng bảng `schema_migrations` nhưng độc lập:

| Đường | Khi nào dùng |
|---|---|
| Tự động lúc khởi động | Mặc định (`DATABASE_AUTO_MIGRATE=true`), chạy trong `container.go:163-170`, timeout 2 phút |
| `cmd/migrate` CLI | Khi muốn kiểm soát riêng, chạy tay, kiểm tra trạng thái |

Migration là **idempotent**: mọi file dùng `CREATE TABLE IF NOT EXISTS`, và
`schema_migrations` chặn chạy lại file đã áp dụng. Không có cơ chế `down`/rollback tự động —
muốn lùi phải viết SQL tay.

⚠️ **Nhiều instance cùng khởi động** có thể chạy migration song song. Hiện chỉ deploy một
instance nên chưa thành vấn đề; nếu scale ngang thì phải thêm khoá (advisory lock) hoặc
tách migration khỏi luồng khởi động.

## Chuẩn hoá DSN

`database.NormalizeMySQLDSN()` tự bổ sung ba tham số nếu người dùng quên:

| Tham số | Vì sao bắt buộc |
|---|---|
| `parseTime=true` | Không có nó, `DATETIME` trả về `[]byte` thay vì `time.Time` → mọi scan vào `time.Time` sẽ lỗi |
| `charset=utf8mb4` | Tiếng Việt có dấu + emoji |
| `multiStatements=true` | File migration chứa nhiều câu lệnh (CREATE + INSERT seed) |

Giá trị người dùng đã tự đặt thì **không bị ghi đè**. Nhờ vậy `.env` chỉ cần
`root:pass@tcp(localhost:3306)/dbname` là đủ.

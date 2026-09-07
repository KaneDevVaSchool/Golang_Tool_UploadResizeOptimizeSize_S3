# Tham chiếu API

Base URL: `https://pictures.vaschools.edu.vn` (dev: `http://localhost:8080`).

Đối chiếu code tại commit `d0ec7c2`. Nguồn sự thật cho danh sách route:
`GetServerHandler()` trong `internal/container/container.go`.

## Định dạng phản hồi chung

Mọi endpoint JSON dùng chung một vỏ bọc (`handlers/base_handler.go`):

```json
// thành công — luôn HTTP 200
{ "success": true, "data": { } }

// lỗi
{ "success": false, "error": { "code": "MÃ_LỖI", "message": "Mô tả tiếng Việt" } }
```

Thông báo lỗi viết bằng tiếng Việt, hướng tới người dùng cuối. Mã lỗi (`code`) ổn định,
dùng để xử lý bằng chương trình.

## Xác thực

| Cơ chế | Áp cho | Cách gửi |
|---|---|---|
| API key | Toàn bộ `/api/*` khi `API_REQUIRE_KEY=true` | Header `X-API-Key` |
| Session | `/api/v1/admin/*` | Cookie `vas_admin_session` (tự động) |
| CSRF | Mọi method ghi khi `CSRF_ENABLED=true` | Header `X-CSRF-Token` khớp cookie `csrf_token` |

Ngoại lệ: `/api/v1/health` **luôn** miễn API key.

⚠️ **Cảnh báo quan trọng.** Bật `API_REQUIRE_KEY=true` (bắt buộc ở production) sẽ áp API key
lên **cả** `/api/v1/public/*`, khiến trang public không hoạt động với khách ẩn danh. Xem
[plan/03-risks.md](./plan/03-risks.md) mục R1.

### Lấy CSRF token cho client không phải trình duyệt

```bash
# 1. Gọi GET bất kỳ để nhận cookie csrf_token
curl -c cookies.txt http://localhost:8080/api/v1/health

# 2. Trích token và gửi kèm ở request ghi
TOKEN=$(grep csrf_token cookies.txt | awk '{print $7}')
curl -b cookies.txt -H "X-CSRF-Token: $TOKEN" -X POST ...
```

Hoặc đặt `CSRF_ENABLED=false` nếu triển khai thuần API, không có trình duyệt.

## Giới hạn tần suất

| Phạm vi | Giới hạn | Ghi chú |
|---|---|---|
| Toàn cục | 100 req/phút/IP | Cấu hình được |
| `/api/v1/metrics` | 10 req/phút/IP | Cố định trong code |
| `POST`/`DELETE` ở `/api/v1/public/*` | 20 req/phút/IP | Cố định; `GET` không bị giới hạn riêng |

Vượt giới hạn trả **429**.

---

# 1. Hệ thống

## `GET /api/v1/health`

Không cần xác thực. Dùng cho giám sát và load balancer.

```json
{
  "status": "ok",
  "service": "s3-upload-api",
  "max_size": 20971520,
  "max_size_formatted": "20.0 MB",
  "absolute_max_size": 209715200,
  "absolute_max_size_formatted": "200.0 MB",
  "request_id": "…",
  "checks": { "disk": { }, "database": { } }
}
```

Trả **503** khi kiểm tra sẵn sàng thất bại (ví dụ mất kết nối DB).

## `GET /api/v1/metrics`

Số request, độ trễ, phân bố mã trạng thái. Giới hạn 10 req/phút.

---

# 2. Upload

## `POST /api/v1/upload`

Upload một file, tối đa `UPLOAD_MAX_SIZE_MB` (mặc định 20MB).

**Request**: `multipart/form-data`, trường file tên `file`.

```json
{
  "success": true,
  "data": {
    "url": "https://bucket.s3.ap-southeast-1.amazonaws.com/images/tranh-20260906-143022-Ab3xK9mQ2pLr.jpg",
    "key": "images/tranh-20260906-143022-Ab3xK9mQ2pLr.jpg",
    "size": 1234567,
    "name": "tranh.jpg"
  }
}
```

| Mã HTTP | `code` | Ý nghĩa |
|---|---|---|
| 400 | `INVALID_FORM` | Form multipart không hợp lệ |
| 400 | `FILE_NOT_FOUND` | Thiếu file trong request |
| 400 | `INVALID_FILE_TYPE` | Đuôi file không được hỗ trợ |
| 400 | `FILE_TOO_LARGE` | Vượt `UPLOAD_MAX_SIZE_MB` |
| 405 | `METHOD_NOT_ALLOWED` | Không phải POST |
| 5xx | `UPLOAD_FAILED` | Lỗi S3 hoặc lỗi hệ thống |

Định dạng cho phép: ảnh (jpg, jpeg, png, gif, webp, bmp, svg, ico), tài liệu (pdf, doc,
docx, xls, xlsx, ppt, pptx, txt, rtf, odt, ods, odp), video (mp4, avi, mov, wmv, flv, webm,
mkv, m4v, 3gp), âm thanh (mp3, wav, ogg, flac, aac, m4a, wma, opus), nén (zip, rar, 7z,
tar, gz, bz2, xz).

> **Đã gỡ (2026-09-07)**: `POST /api/v1/upload-transaction`, nhóm chunked
> `POST /api/v1/upload/{init,chunk,complete,abort}`, `POST /api/v1/wp-upload` và static
> `/wp-content/uploads/`. Xem [detail_design/02-upload-pipeline.md](./detail_design/02-upload-pipeline.md)
> để biết lý do. Client cũ gọi các đường này sẽ nhận 404.

---

# 3. Dữ liệu nền (công khai)

Không cần đăng nhập — dùng cho cả form admin lẫn bộ lọc trang public.

## `GET /api/v1/schools`

Danh sách cơ sở đang hoạt động (hiện là 16 cơ sở: 8 Sài Gòn, 5 Vũng Tàu, 3 Cần Thơ).

```json
{
  "success": true,
  "data": [
    { "id": 1, "name": "Bình Thới - Tân Bình", "region": "saigon",
      "display_order": 1, "is_active": true, "created_at": "…", "updated_at": "…" }
  ]
}
```

`region` là một trong `saigon` / `cantho` / `vungtau` — **3 khu vực trưng bày**, không phải
5 địa điểm vật lý.

## `GET /api/v1/grade-levels`

Danh sách 12 khối. Lọc tuỳ chọn: `?education_level=primary|secondary`.

```json
{ "id": 1, "education_level": "primary", "grade_number": 1, "label": "Khối 1", "display_order": 1 }
```

## `GET /api/v1/awards`

Danh sách giải thưởng **đang hoạt động** (bản public chỉ trả `is_active=true`). Mỗi giải có
thể mang `grade_level_id` — `null`/vắng mặt nghĩa là giải dùng chung toàn hệ thống (vd
"Đặc biệt"); một số nguyên nghĩa là giải chỉ áp dụng cho đúng khối lớp đó (vd hội thi chia
giải riêng theo khối: mỗi khối Tiểu học có 1 Nhất/1 Nhì/2 Ba).

## `GET /api/v1/topic-categories`

Danh sách nhóm chủ đề sáng tạo **đang hoạt động** (bản public chỉ trả `is_active=true`).
Mỗi nhóm có thể mang `education_level` (`primary`/`secondary`) để giới hạn nhóm đó chỉ áp
dụng cho một cấp học — thể lệ Tiểu học và THCS-THPT dùng bộ nhóm chủ đề khác nhau; `null`
nghĩa là nhóm dùng chung mọi cấp.

---

# 4. API công khai — trang triển lãm

Không cần đăng nhập. Ghi dữ liệu bị giới hạn 20 req/phút/IP và cần CSRF token.

## `GET /api/v1/public/artworks`

Danh sách tác phẩm đã xuất bản. **Luôn ép `is_published=true`** — không thể xem bản nháp
qua API này.

| Tham số | Kiểu | Mặc định | Ghi chú |
|---|---|---|---|
| `search` | string | — | Khớp tên tác phẩm hoặc tên học sinh; **cắt còn 80 ký tự** |
| `school_id` | int | — | |
| `region` | string | — | `saigon` / `cantho` / `vungtau`; giá trị lạ bị bỏ qua (không lọc) |
| `grade_level_id` | int | — | |
| `education_level` | string | — | `primary` / `secondary` |
| `topic_category_id` | int | — | |
| `page` | int | 1 | |
| `page_size` | int | 24 | **Trần cứng 100** |

```json
{
  "success": true,
  "data": {
    "items": [ /* ArtworkWithMeta */ ],
    "total_count": 350,
    "page": 1,
    "page_size": 24
  }
}
```

### Cấu trúc `ArtworkWithMeta`

```json
{
  "id": 42,
  "title": "Mùa xuân quê em",
  "student_id": 108,
  "school_id": 1,
  "grade_level_id": 3,
  "image_url": "https://…",          // ← s3_url; s3_key KHÔNG bao giờ được trả về
  "thumbnail_url": "https://…",      // trỏ thumb_jpg; null với ảnh cũ hoặc ảnh gốc quá nhỏ
  "variants": {                      // các cỡ sinh lúc upload; có thể vắng mặt
    "thumb_webp": "https://…", "thumb_jpg": "https://…",
    "medium_webp": "https://…", "medium_jpg": "https://…",
    "large_webp": "https://…", "large_jpg": "https://…"
  },
  "file_size": 2458000,
  "width": 1920,
  "height": 1080,
  "is_featured": true,
  "is_published": true,
  "view_count": 1250,
  "created_at": "2026-09-01T10:30:00Z",
  "updated_at": "2026-09-01T10:30:00Z",
  "student_name": "Nguyễn Văn A",
  "school_name": "Bình Thới - Tân Bình",
  "region": "saigon",
  "grade_label": "Khối 3",
  "education_level": "primary",
  "topic_category_id": 5,               // có thể vắng mặt - tác phẩm cũ chưa gán nhóm chủ đề
  "topic_category_name": "Mái trường Việt Mỹ - Nơi những điều đẹp đẽ được lắng nghe",
  "class_name": "3A2",
  "comment_count": 15,
  "reaction_counts": { "like": 30, "love": 45 },
  "awards": [                            // 1 tác phẩm có thể nhận nhiều giải cùng lúc
    { "id": 1, "name": "Giải Nhất", "slug": "giai-nhat",
      "grade_level_id": 3, "rank_order": 1, "color_hex": "#c49c57" },
    { "id": 9, "name": "Giải Đặc biệt - Nét vẽ Việt Mỹ", "slug": "dac-biet-net-ve",
      "rank_order": 9, "color_hex": "#725139" }
  ]
}
```

⚠️ `s3_url` xuất hiện dưới tên **`image_url`**; `s3_key`, `upload_id`, `created_by` bị ẩn
khỏi JSON.

## `GET /api/v1/public/artworks/featured`

Tác phẩm tiêu biểu. Tham số tuỳ chọn `region=saigon|cantho|vungtau`, lọc bằng SQL (JOIN
`schools`) giống `GET /api/v1/public/artworks` — không còn giới hạn 100 bản ghi rồi lọc
trong bộ nhớ.

```json
{ "success": true, "data": { "items": [ ], "total_count": 12 } }
```

## `GET /api/v1/public/artworks/{id}`

Chi tiết một tác phẩm, **kèm ghi nhận lượt xem**.

| Tham số | Ghi chú |
|---|---|
| `visitor_token` | Tuỳ chọn; có thì mới tính lượt xem |

⚠️ **Mỗi lần gọi kèm `visitor_token` đều +1 `view_count`** — không chống trùng theo thời
gian. Gọi lại nhiều lần (tải lại trang, mở lại lightbox) sẽ tăng số nhiều lần.

Trả **404** nếu không tồn tại **hoặc** chưa xuất bản — không phân biệt hai trường hợp.

## `GET /api/v1/public/artworks/{id}/download`

Tải ảnh **gốc** (không phải thumbnail/variant), chèn watermark logo VAS góc dưới-phải trước
khi trả về — stream qua API thay vì trỏ thẳng URL S3 (same-origin, không mở tab rời).
`Content-Disposition: attachment` kèm tên file lấy từ tiêu đề tác phẩm.

Trả **404** nếu tác phẩm không tồn tại, chưa xuất bản, hoặc không có `s3_key`. Nếu chèn
watermark lỗi (không đọc được ảnh, thiếu file logo …), trả **ảnh gốc không watermark** kèm
ghi log — lỗi khâu phụ trợ không chặn việc tải ảnh.

Mỗi lượt tải thành công được ghi vào bảng `artwork_downloads` (`source='public'`,
`admin_user_id` luôn `NULL` vì người xem ẩn danh không có tài khoản) — phục vụ truy vết nếu
ảnh bị phát tán sai mục đích. Đây cũng là log phụ trợ: ghi lỗi chỉ log cảnh báo, không chặn
việc trả ảnh. Xem [01-database.md](./detail_design/01-database.md).

## `GET /api/v1/public/billboard`

Bảng vàng: các tác phẩm đạt giải, sắp theo `rank_order` (Nhất trước).

```json
{ "success": true, "data": [ { /* ArtworkWithMeta */, "award": { } } ] }
```

⚠️ Trần 100 tác phẩm mỗi giải.

## Cảm xúc

### `POST /api/v1/public/artworks/{id}/reactions`

```json
{ "reaction_type": "love", "visitor_token": "uuid" }
```

`reaction_type` ∈ `like` `love` `haha` `wow` `sad` `angry`.

Idempotent — gửi lại cùng loại không tạo bản ghi mới, không báo lỗi.

```json
{ "success": true, "data": { "reaction_counts": { "like": 30, "love": 46 } } }
```

### `DELETE /api/v1/public/artworks/{id}/reactions/{type}?visitor_token=…`

Trả về bảng đếm mới nhất, cùng cấu trúc như trên.

| `code` | Ý nghĩa |
|---|---|
| `INVALID_REACTION_TYPE` | Loại cảm xúc không hợp lệ |
| `MISSING_VISITOR_TOKEN` | Thiếu định danh trình duyệt |
| `REACTION_FAILED` | Lỗi ghi dữ liệu |

## Bình luận

### `GET /api/v1/public/artworks/{id}/comments?visitor_token=…`

Chỉ trả bình luận **chưa bị ẩn**.

```json
{
  "success": true,
  "data": [
    { "id": 1, "artwork_id": 42, "display_name": "Phụ huynh A",
      "content": "Bé vẽ đẹp quá!", "created_at": "…", "can_delete": true }
  ]
}
```

`can_delete` = `true` khi `visitor_token` gửi lên trùng với người đã viết bình luận đó.

### `POST /api/v1/public/artworks/{id}/comments`

```json
{ "display_name": "Phụ huynh A", "content": "Bé vẽ đẹp quá!", "visitor_token": "uuid" }
```

| Trường | Ràng buộc |
|---|---|
| `display_name` | Bắt buộc, ≤ 100 **ký tự** |
| `content` | Bắt buộc, ≤ 1000 **ký tự** |
| `visitor_token` | Bắt buộc |

Độ dài đếm theo ký tự Unicode, không phải byte.

| `code` | Ý nghĩa |
|---|---|
| `MISSING_DISPLAY_NAME` / `DISPLAY_NAME_TOO_LONG` | Tên hiển thị |
| `MISSING_CONTENT` / `CONTENT_TOO_LONG` | Nội dung |
| `MISSING_VISITOR_TOKEN` | Thiếu định danh |
| `COMMENT_FAILED` | Lỗi ghi dữ liệu |

### `DELETE /api/v1/public/artworks/{id}/comments/{commentID}?visitor_token=…`

Chỉ xoá được bình luận do chính trình duyệt đó tạo.

Trả **404 `COMMENT_NOT_FOUND`** cho **cả** trường hợp không tồn tại lẫn không phải của mình —
cố ý không phân biệt, để không ai dò được quyền sở hữu bình luận.

---

# 5. API quản trị

Toàn bộ yêu cầu cookie session hợp lệ. Thiếu/hết hạn → **401 `UNAUTHORIZED`**.

## Xác thực

| Endpoint | Ghi chú |
|---|---|
| `GET /auth/google/login` | Bắt đầu luồng OAuth (redirect trình duyệt, **ngoài** `/api`) |
| `GET /auth/google/callback` | Google gọi về; xử lý xong redirect `/admin` |
| `GET /api/v1/admin/auth/me` | Thông tin admin đang đăng nhập |
| `POST /api/v1/admin/auth/logout` | **Không** qua middleware auth — luôn xoá được cookie |

`/auth/google/*` trả **503 `OAUTH_NOT_CONFIGURED`** nếu chưa cấu hình Client ID/Secret.

Khi đăng nhập thất bại, người dùng bị chuyển về `/admin/login?error=<mã>` với các mã:
`invalid_state`, `missing_code`, `exchange_failed`, `userinfo_failed`, `incomplete_profile`,
`email_not_verified`, `email_not_allowed`, `server_error`.

## Tác phẩm

### `POST /api/v1/admin/artworks/bulk-upload`

Bước ① — đẩy nhiều ảnh lên S3, **chưa tạo bản ghi**.

**Request**: `multipart/form-data`, trường `files` (nhiều file).

```json
{
  "success": true,
  "data": {
    "items": [
      { "temp_key": "tranh1.jpg", "file_name": "tranh1.jpg",
        "s3_key": "images/…", "s3_url": "https://…",
        "file_size": 2458000, "width": 1920, "height": 1080 },
      { "temp_key": "tranh2.jpg", "file_name": "tranh2.jpg",
        "error": "file quá lớn: 25.0 MB (giới hạn: 20.0 MB)" }
    ],
    "open_errors": ["hỏng.jpg"]
  }
}
```

Lỗi từng file **không** làm hỏng cả lô: item lỗi có trường `error`. `open_errors` liệt kê
file không mở/không hợp lệ ngay từ đầu.

### `POST /api/v1/admin/artworks`

Bước ② — tạo bản ghi từ ảnh đã có trên S3.

```json
{
  "title": "Mùa xuân quê em",
  "student_name": "Nguyễn Văn A",
  "school_id": 1,
  "grade_level_id": 3,
  "topic_category_id": 5,
  "class_name": "3A2",
  "s3_key": "images/…",
  "s3_url": "https://…",
  "file_size": 2458000,
  "width": 1920,
  "height": 1080,
  "award_ids": [1, 9]
}
```

Bắt buộc: `title`, `student_name`, `school_id`, `grade_level_id`, `s3_key`, `s3_url`.
Thiếu → **400 `VALIDATION_ERROR`** kèm thông báo tiếng Việt cụ thể. `topic_category_id` và
`award_ids` đều tuỳ chọn — `award_ids` cho phép gán nhiều giải ngay lúc tạo (vd giải chính
+ giải Đặc biệt phụ); vắng mặt hoặc rỗng nghĩa là chưa gán giải nào.

### `GET /api/v1/admin/artworks`

Như bản public nhưng **không** ép `is_published`, và có thêm `is_featured`, `is_published`,
`award_id`, `topic_category_id` trong bộ lọc.

### `GET /api/v1/admin/artworks/{id}`

### `PUT /api/v1/admin/artworks/{id}`

```json
{
  "title": "…", "student_id": 108, "school_id": 1, "grade_level_id": 3,
  "topic_category_id": 5,
  "is_featured": true, "is_published": true, "award_ids": [2, 9]
}
```

Ngữ nghĩa `award_ids`: **không gửi trường** (key vắng mặt trong JSON) = giữ nguyên giải hiện
tại · gửi `[]` = gỡ hết giải · gửi danh sách = thay **toàn bộ** giải hiện tại bằng danh sách
đó. Không còn giới hạn 1 giải/tác phẩm — một tác phẩm có thể nhận nhiều giải cùng lúc (vd
giải chính Nhất/Nhì/Ba + giải Đặc biệt phụ).

⚠️ **Không đổi được ảnh** qua endpoint này.

### `DELETE /api/v1/admin/artworks/{id}`

Xoá bản ghi và các dữ liệu liên quan (cảm xúc/bình luận/lượt xem/giải) theo CASCADE.
⚠️ **File trên S3 cũng bị xoá** — ảnh gốc và mọi biến thể/thumbnail trong `variants`, xoá
trên S3 trước khi xoá bản ghi DB. Xem [03-artwork-domain.md §4](./detail_design/03-artwork-domain.md).

### `GET /api/v1/admin/artworks/{id}/download`

Tải ảnh **gốc, không watermark, không ép `is_published`** — công cụ nội bộ để admin lưu
trữ/in ấn, khác `GET /api/v1/public/artworks/{id}/download` ở hai điểm này. Cùng cơ chế
proxy qua backend (same-origin, không cần CORS trên bucket).

Mỗi lượt tải thành công được ghi vào bảng `artwork_downloads` (`source='admin'`, kèm
`admin_user_id` của admin đang đăng nhập) — vì không watermark, đây là nơi duy nhất định
danh được người tải khi cần truy vết. Log phụ trợ, không chặn việc tải nếu ghi lỗi. Xem
[01-database.md](./detail_design/01-database.md).

### `DELETE /api/v1/admin/artworks/bulk-delete`

Xoá nhiều tác phẩm cùng lúc — thao tác chọn nhiều trên trang danh sách quản trị. Dùng
method `DELETE` với thân JSON `{ids}` thay vì query string vì số lượng id có thể lớn.

```json
{ "ids": [12, 15, 22] }
```

Khác `PATCH /bulk-featured` (1 câu `UPDATE ... WHERE id IN (...)`), endpoint này xoá **từng
tác phẩm một** ở tầng service vì mỗi tác phẩm còn cần xoá kèm object S3 riêng, không gộp
được thành 1 lệnh SQL. Tác phẩm nào xoá S3 lỗi (mất mạng, key đã mất...) thì **giữ nguyên**
bản ghi DB của riêng nó và tiếp tục sang tác phẩm tiếp theo — không để 1 lỗi chặn cả lô.

```json
{ "success": true, "data": { "deleted": 21, "requested": 22 } }
```

`deleted < requested` nghĩa là có tác phẩm xoá S3 lỗi và chưa bị xoá — chọn lại đúng những
tác phẩm đó để thử lại. Xem [03-artwork-domain.md §4](./detail_design/03-artwork-domain.md).

### `PATCH /api/v1/admin/artworks/{id}/featured`

```json
{ "featured": true }
```

Body dùng khoá `featured` (không phải `is_featured`) - khác tên với trường trên chính
`Artwork`, xem `artwork_handler.go` `HandleSetFeatured`.

### `PATCH /api/v1/admin/artworks/bulk-featured`

Bật/tắt tiêu biểu cho nhiều tác phẩm cùng lúc - 1 câu `UPDATE ... WHERE id IN (...)` thay vì
gọi lặp endpoint đơn lẻ ở trên, dùng cho thao tác chọn nhiều trên trang danh sách quản trị.

```json
{ "ids": [12, 15, 22], "featured": true }
```

```json
{ "success": true, "data": { "updated": 3, "is_featured": true } }
```

`ids` rỗng → **400 `VALIDATION_ERROR`**. Id không tồn tại bị bỏ qua lặng lẽ, không coi là lỗi.

## Giải thưởng

| Method | Đường dẫn |
|---|---|
| GET | `/api/v1/admin/awards` (gồm cả giải đã tắt) |
| POST | `/api/v1/admin/awards` |
| PUT | `/api/v1/admin/awards/{id}` |
| DELETE | `/api/v1/admin/awards/{id}` |

```json
{ "name": "Giải Nhất", "slug": "giai-nhat", "grade_level_id": 3, "rank_order": 1,
  "color_hex": "#c49c57", "icon_key": "trophy", "is_active": true }
```

`slug` là **duy nhất**. `rank_order` quyết định thứ tự trên bảng vàng (nhỏ = hạng cao).
`grade_level_id` tuỳ chọn: `null`/vắng mặt = giải dùng chung toàn hệ thống (vd "Đặc biệt");
một số nguyên = giải chỉ áp dụng cho đúng khối lớp đó (hội thi chia giải riêng theo khối).

DELETE trả **409 `AWARD_IN_USE`** nếu giải đang gắn cho tác phẩm nào đó (ràng buộc khoá
ngoại `artwork_awards`).

## Nhóm chủ đề sáng tạo

| Method | Đường dẫn |
|---|---|
| GET | `/api/v1/admin/topic-categories` (gồm cả nhóm đã tắt) |
| POST | `/api/v1/admin/topic-categories` |
| PUT | `/api/v1/admin/topic-categories/{id}` |
| DELETE | `/api/v1/admin/topic-categories/{id}` |

```json
{ "name": "Mái trường Việt Mỹ - Nơi những điều đẹp đẽ được lắng nghe",
  "slug": "mai-truong-viet-my", "color_hex": "#9a0036", "education_level": "secondary",
  "display_order": 3, "is_active": true }
```

`slug` là **duy nhất**. `color_hex` tuỳ chọn (mặc định `#725139` nếu bỏ trống) — tô icon
nhóm trong dropdown `ArtworkMetaForm` và trang quản lý, cùng cơ chế `awards.color_hex`.
`education_level` tuỳ chọn: `null`/vắng mặt = nhóm dùng chung mọi cấp học; `primary`/
`secondary` = nhóm chỉ áp dụng cho đúng cấp đó (thể lệ Tiểu học và THCS-THPT dùng bộ nhóm
chủ đề khác nhau).

DELETE trả **409 `TOPIC_CATEGORY_IN_USE`** nếu nhóm đang gắn cho tác phẩm nào đó.

## Bảng điều khiển

### `GET /api/v1/admin/dashboard/stats?from=YYYY-MM-DD&to=YYYY-MM-DD`

`from`/`to` tuỳ chọn, chỉ ảnh hưởng khối `activity` — thiếu một trong hai (hoặc cả hai, hoặc
giá trị không parse được) thì tự áp mặc định **14 ngày gần nhất**, giữ đúng hành vi trước khi
có bộ lọc này. Khoảng dài hơn 366 ngày bị cắt về đúng 366 ngày, tính lùi từ `to`, để chặn
truy vấn quét quá nhiều dữ liệu.

```json
{
  "success": true,
  "data": {
    "total_artworks": 350,
    "total_by_region": { "saigon": 200, "cantho": 80, "vungtau": 70 },
    "total_by_grade": [ { } ],
    "top_schools": [ { } ],
    "top_artworks": [ { } ],
    "activity": [
      { "date": "2026-09-06", "uploads": 12, "views": 210, "reactions": 44, "comments": 9 }
    ],
    "school_coverage": [
      {
        "school_id": 1, "name": "Phú Định", "region": "saigon",
        "artworks": 57, "grades_covered": 9, "total_grades": 12, "awarded": 4
      }
    ],
    "operations": {
      "pending_artworks": 23, "hidden_comments": 7, "total_comments": 264,
      "awarded_artworks": 27, "active_awards": 4, "featured_artworks": 6,
      "silent_artworks": 14, "total_views": 9142, "total_reactions": 1863
    }
  }
}
```

Gộp toàn bộ số liệu vào **một** lần gọi, để frontend không phải gọi nhiều endpoint rời rạc.

Ba khối bổ sung phục vụ việc **ra quyết định**, không chỉ mô tả quy mô:

| Khối | Ý nghĩa |
|---|---|
| `activity` | Nhịp từng ngày trong khoảng `[from, to]` (mặc định 14 ngày gần nhất — xem tham số ở trên). Trả đủ ngày liên tục **trong đoạn có dữ liệu**: ngày 0 hoạt động xen giữa hai ngày có hoạt động vẫn có mặt (tín hiệu thật — thiếu ngày sẽ làm biểu đồ đường vẽ sai độ dốc), nhưng phần đầu/cuối chuỗi toàn số 0 (vd tháng chưa hết) bị cắt bỏ ở tầng service trước khi trả ra. `views` đếm từ `artwork_views` chứ không lấy `artworks.view_count` (cột đó là tổng tích luỹ, không tách được theo ngày). |
| `school_coverage` | Mỗi cơ sở đã có bài ở bao nhiêu khối trên tổng số khối. Trường **chưa có tác phẩm nào vẫn xuất hiện** với số 0 — đó chính là nơi ban tổ chức cần nhắc. |
| `operations` | Các con số cần hành động: bài chờ xuất bản, bình luận đã ẩn, tiến độ trao giải, tác phẩm chưa có tương tác nào. |

### `GET /api/v1/admin/dashboard/region-summary`

```json
{
  "success": true,
  "data": [
    { "region": "saigon", "artworks": 200, "students": 235 },
    { "region": "cantho", "artworks": 80, "students": 92 },
    { "region": "vungtau", "artworks": 70, "students": 78 }
  ]
}
```

Phiên bản nhẹ của `/dashboard/stats`, chỉ trả đúng hai con số mỗi khu vực, không kéo theo
`activity`/`top_schools`/`top_artworks`/`school_coverage`/`operations`. Luôn trả đủ 3 khu vực
theo thứ tự cố định `saigon`, `cantho`, `vungtau`.

⚠️ **Hiện không có client nào gọi.** Endpoint sinh ra cho dải card thống kê ở đầu trang Tác
phẩm/Giải thưởng/Nhóm chủ đề quản trị; dải đó đã gỡ ngày 2026-09-07 vì lặp lại số liệu có
sẵn trên Dashboard. Endpoint vẫn để nguyên (rẻ, không phụ thuộc gì) nhưng nếu không dùng lại
trong đợt tới thì nên gỡ cả handler, service và repository cùng lúc.

`artworks` đếm như `total_by_region` ở trên (JOIN `artworks`-`schools`, `WHERE is_published =
1`). `students` đếm **mọi** bản ghi bảng `students` JOIN `schools` theo `region`, **không
dedupe theo tên** — đúng quy ước ở
[01-database.md](detail_design/01-database.md#students--học-sinh-migration-006): mỗi lần tạo
tác phẩm luôn tạo một bản ghi học sinh mới, không có mã định danh học sinh chính thức nên
không gộp trùng.

---

# 6. Trang không phải JSON

| Đường dẫn | Trả về |
|---|---|
| `GET /chia-se/tac-pham/{id}` | HTML có thẻ Open Graph + JSON-LD `BreadcrumbList`, tự chuyển hướng về SPA |
| `GET /sitemap.xml` | XML: 4 trang public cố định (`/`, `/tac-pham-tieu-bieu`, `/phong-trien-lam`, `/bang-vang`) + mọi tác phẩm đã publish (trỏ `/chia-se/tac-pham/{id}`). `Cache-Control: public, max-age=900` |
| `GET /robots.txt` | `Allow: /`, `Disallow: /admin/ /api/ /auth/`, trỏ `Sitemap:` tới `/sitemap.xml` bằng URL tuyệt đối. `Cache-Control: public, max-age=3600` |
| `GET /wp-content/uploads/*` | File tĩnh (đã chặn liệt kê thư mục) |
| `GET /*` | SPA React; fallback `index.html`. Chưa build `dist` thì trả JSON thông tin API |

---

# 7. Bảng mã lỗi

| `code` | HTTP | Ý nghĩa |
|---|---|---|
| `METHOD_NOT_ALLOWED` | 405 | Sai HTTP method |
| `INVALID_FORM` | 400 | Form multipart hỏng |
| `FILE_NOT_FOUND` | 400 | Thiếu file |
| `INVALID_FILE_TYPE` | 400 | Đuôi file không hỗ trợ |
| `FILE_TOO_LARGE` | 400 | Vượt giới hạn dung lượng |
| `NO_FILES` / `ALL_FILES_INVALID` | 400 | Bulk upload không có file hợp lệ |
| `INVALID_BODY` | 400 | JSON không hợp lệ |
| `VALIDATION_ERROR` | 400 | Thiếu trường bắt buộc |
| `INVALID_ID` / `INVALID_COMMENT_ID` | 400 | ID không hợp lệ |
| `INVALID_REACTION_TYPE` | 400 | Loại cảm xúc sai |
| `MISSING_VISITOR_TOKEN` | 400 | Thiếu định danh trình duyệt |
| `MISSING_DISPLAY_NAME` / `DISPLAY_NAME_TOO_LONG` | 400 | Tên hiển thị |
| `MISSING_CONTENT` / `CONTENT_TOO_LONG` | 400 | Nội dung bình luận |
| `MISSING_API_KEY` | 401 | Thiếu header `X-API-Key` |
| `UNAUTHORIZED` | 401 | Session không hợp lệ/hết hạn |
| `INVALID_API_KEY` | 403 | Sai API key |
| `INVALID_CSRF` | 403 | Token CSRF sai/thiếu |
| `NOT_FOUND` / `COMMENT_NOT_FOUND` | 404 | Không tìm thấy |
| `AWARD_IN_USE` | 409 | Xoá giải đang gắn cho tác phẩm |
| `TOPIC_CATEGORY_IN_USE` | 409 | Xoá nhóm chủ đề đang gắn cho tác phẩm |
| — | 429 | Vượt giới hạn tần suất |
| `CREATE_FAILED` / `UPLOAD_FAILED` / `REACTION_FAILED` / `COMMENT_FAILED` | 5xx | Lỗi xử lý |
| `INTERNAL_ERROR` | 500 | Lỗi không xác định |
| `OAUTH_NOT_CONFIGURED` | 503 | Chưa cấu hình Google OAuth |
| `CSRF_INIT_FAILED` | 500 | Không sinh được token CSRF |

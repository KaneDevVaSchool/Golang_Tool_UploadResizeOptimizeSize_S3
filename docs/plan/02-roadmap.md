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

### P2.4 — Trang 404 riêng

Hiện mọi đường dẫn lạ đều rơi vào SPA và hiện trang chủ, gây bối rối.

### P2.5 — Xuất danh sách tác phẩm

Ban tổ chức cần bảng Excel/CSV để đối chiếu và in ấn. Có thể làm hoàn toàn ở frontend từ
dữ liệu đã tải.

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

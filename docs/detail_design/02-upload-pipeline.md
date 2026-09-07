# 02 — Pipeline upload

Chỉ còn **một** đường upload dùng chung cho mọi tình huống:

| Đường | Endpoint | Dùng khi | Giới hạn |
|---|---|---|---|
| Đơn | `POST /api/v1/upload` | Mọi file | `UPLOAD_MAX_SIZE_MB` (20MB) |
| Bulk (admin) | `ArtworkService.BulkUploadToS3` | Admin đẩy nhiều tranh cùng lúc | `UPLOAD_ABSOLUTE_MAX_MB` (200MB) mỗi file |

> **Đã gỡ (2026-09-07)**: đường upload có transaction (`POST /api/v1/upload-transaction`) và
> đường chunked (`POST /api/v1/upload/{init,chunk,complete,abort}`). Chunk session giữ trong
> bộ nhớ tiến trình nên không sống sót qua restart, và giao diện duy nhất dùng nó — công cụ
> `/upload` nội bộ — cũng bị gỡ trong cùng đợt. Bảng `uploads` (migration 001) còn trong CSDL
> nhưng không code nào đọc/ghi nữa; xem [01-database.md](./01-database.md).

## 1. Nhiều lớp kiểm tra

Một file phải qua ba lớp trước khi lên S3. Mỗi lớp chặn một kiểu tấn công/lỗi khác nhau —
không lớp nào thừa:

```text
Lớp 1 — Tên file          SanitizeFilename()
                          chống path traversal: bỏ "..", "/", "\", ký tự điều khiển,
                          < > : " | ? * ; cắt còn 255 ký tự
                          ↓
Lớp 2 — Đuôi file         IsAllowedFileType()
                          whitelist theo nhóm (ảnh/tài liệu/video/audio/nén)
                          ↓
Lớp 3 — Dung lượng        ValidateFileSizeFromHeader() rồi LimitReader(maxSize+1)
                          kiểm tra 2 lần: theo header khai báo, và theo byte thực đọc
```

⚠️ **Không còn lớp kiểm tra magic byte.** `ValidateFileContent()` (đọc 512 byte đầu,
`http.DetectContentType`, đối chiếu đuôi file) trước đây **chỉ chạy ở đường chunked**, và
đường đó đã bị gỡ. Nghĩa là hiện **không đường upload nào** kiểm tra nội dung thật: file
`.jpg` chứa nội dung khác vẫn lên được S3.

Rủi ro thực tế vẫn thấp vì bucket chỉ phục vụ ảnh tĩnh và không thực thi nội dung, nhưng
đây không còn là "điểm không nhất quán" như trước — nó là một lớp phòng thủ đã **mất hẳn**.
Xem mục P1.4 trong [plan/02-roadmap.md](../plan/02-roadmap.md).

### Vì sao kiểm tra dung lượng hai lần

Header `Content-Length` do client khai báo, có thể nói dối. Nên:

1. Kiểm tra header trước — chặn sớm, không tốn I/O.
2. Đọc qua `io.LimitReader(file, maxSize+1)` — cái `+1` là mấu chốt: nếu đọc được đúng
   `maxSize+1` byte thì biết file thực sự vượt hạn mức chứ không phải vừa khít
   (`upload_service.go:152-166`).

## 2. Upload đơn — chi tiết

```text
POST /api/v1/upload  (multipart/form-data)
  │
  ├─ ParseMultipartForm → lấy file → validate type + size (header)
  │
  ├─ UploadService.UploadImage
  │    ├─ os.CreateTemp(UPLOAD_DIR, "upload-*<ext>")
  │    │   defer: xoá temp file  ← luôn chạy, kể cả khi lỗi
  │    │
  │    ├─ io.Copy có kiểm soát thời gian
  │    │   timeout = 60s + 2s mỗi MB, trần 5 phút
  │    │   chạy trong goroutine vì io.Copy KHÔNG nhận context
  │    │   quá hạn → đóng file để cắt I/O → chờ goroutine thoát tối đa 3s
  │    │
  │    ├─ kiểm tra byte thực: > maxSize hoặc == maxSize+1 → lỗi FILE_TOO_LARGE
  │    ├─ Seek(0,0) → GenerateS3Key → GetContentType
  │    ├─ S3Repository.Upload   part 8MB, concurrency 4, LeavePartsOnError=false
  │    └─ sinh URL (object URL hoặc presigned)
  │
  └─ 200 {url, key, size, name}
```

**Vì sao timeout co giãn theo dung lượng.** Một timeout cố định hoặc quá ngắn cho file lớn
trên mạng chậm, hoặc quá dài để phát hiện kết nối chết. Công thức `60s + 2s/MB` cho file
20MB được 100 giây — đủ rộng cho mạng chậm, đủ chặt để không treo tài nguyên.

**Vì sao phải đóng file khi quá hạn.** `io.Copy` không nhận `context` và không dừng giữa
chừng được. Cách duy nhất cắt nó là đóng file bên dưới để lệnh ghi trả lỗi. Code chờ
goroutine thoát sạch tối đa 3 giây rồi mới bỏ qua — tránh rò rỉ goroutine
(`upload_service.go:130-147`).

**`LeavePartsOnError=false`.** Nếu multipart lên S3 hỏng giữa chừng, AWS SDK tự huỷ các
part đã tải. Không có nó, các part mồ côi vẫn tính tiền lưu trữ mà không ai thấy.

## 3. Sinh S3 key

`utils.GenerateS3Key()` (`file_utils.go:208`) tạo key theo mẫu:

```text
[basePath/]<category>/<tên-file>-<YYYYMMDD-HHMMSS>-<12 ký tự ngẫu nhiên><ext>

ví dụ:  vaschools-uploads/images/tranh-mua-xuan-20260906-143022-Ab3xK9mQ2pLr.jpg
```

| Thành phần | Vai trò |
|---|---|
| `basePath` | Tiền tố tuỳ chọn (`S3_BASE_PATH`), gom toàn bộ file dự án vào một nhánh trong bucket dùng chung |
| `category` | `images`/`documents`/`videos`/`audio`/`archives`/`files` — suy từ đuôi file |
| Tên file | Khoảng trắng đổi thành `-`, giữ nguyên phần còn lại để người vận hành đọc được |
| Timestamp | Sắp xếp theo thời gian khi duyệt bucket |
| 12 ký tự ngẫu nhiên | Chống trùng khi nhiều người upload cùng giây — dùng `crypto/rand` |

Nếu `crypto/rand` lỗi, fallback dùng `UnixNano()`. Chống trùng yếu hơn nhưng không bao giờ
để upload thất bại chỉ vì thiếu entropy.

Key được **escape từng đoạn** khi dựng URL (`s3_url.go:escapeS3Key`) — tên file tiếng Việt
có dấu vẫn ra URL hợp lệ, và dấu `/` phân cấp không bị mã hoá nhầm.

## 4. Sinh URL trả về

`BuildS3ObjectURL()` xử lý ba dạng hạ tầng:

| Trường hợp | URL sinh ra |
|---|---|
| S3 thật, virtual-host (mặc định) | `https://<bucket>.s3.<region>.amazonaws.com/<key>` |
| S3 thật, `S3_FORCE_PATH_STYLE=true` | `https://s3.<region>.amazonaws.com/<bucket>/<key>` |
| Endpoint tuỳ chỉnh (MinIO/LocalStack) | `<endpoint>/<bucket>/<key>` — luôn path-style |

⚠️ **Cảnh báo về presigned URL.** Khi `S3_USE_PRESIGNED_URL=true`, URL sinh ra **được lưu
vào `artworks.s3_url`** và sẽ hết hạn sau `S3_PRESIGNED_URL_EXPIRY` phút — ảnh trên trang
public sẽ trả 403 sau đó. Với kho ảnh công khai lâu dài, **phải để `false`** và mở quyền
đọc bằng bucket policy (xem [S3-PUBLIC-READ.md](../S3-PUBLIC-READ.md)). Cảnh báo này cũng
được ghi trong `.env.example`.

## 5. Bulk upload (dùng cho admin)

`ArtworkService.BulkUploadToS3` (`internal/service/artwork_service.go`) upload nhiều file
song song:

- **Giới hạn đồng thời 5** qua semaphore — tránh áp đảo S3/băng thông khi admin chọn hàng
  chục ảnh.
- **Lỗi từng file không làm hỏng cả lô**: mỗi item có trường `Error` riêng; FE hiển thị
  file nào hỏng, file nào xong.
- **Đọc kích thước ảnh trước** (`DecodeImageDimensions`) rồi `Seek(0)` để dùng lại reader
  cho việc upload — không đọc file hai lần từ đĩa.
- Chỉ đẩy lên S3, **chưa ghi bảng `artworks`**. Xem
  [03-artwork-domain.md](./03-artwork-domain.md) cho bước 2.
- **Sinh biến thể ảnh ngay trong bước này** (hàm `buildVariants`) — xem mục 6.

## 6. Sinh biến thể ảnh

Vấn đề đã giải quyết: lưới gallery trước đây tải **ảnh gốc** cho từng ô, nên một trang 36
tranh dự thi cỡ vài MB kéo hàng chục MB.

Mỗi ảnh upload qua bulk-upload nay được sinh sẵn bộ biến thể, đẩy lên S3 **cạnh ảnh gốc**:

| Cỡ | Cạnh dài tối đa | Dùng ở | Chất lượng JPEG |
|---|---|---|---|
| `thumb` | 400px | Ô lưới gallery (rộng ~260–320px × 1.5 cho retina) | 78 |
| `medium` | 1000px | Lightbox trên laptop | cao hơn |
| `large` | 1600px | Màn hình lớn / retina; trên mức này dùng ảnh gốc | cao hơn |

Mỗi cỡ sinh ở **hai định dạng**: WebP và JPEG.

Bốn quyết định thiết kế đáng chú ý:

**WebP lossy không cần cgo.** `golang.org/x/image/webp` chỉ có bộ **giải mã**. Bản đầu dùng
`nativewebp` nhưng thư viện này chỉ làm lossless (VP8L), cho ra file **lớn hơn JPEG ~10 lần**
với tranh vẽ và ảnh chụp. Đã đổi sang `github.com/gen2brain/webp` — encode lossy VP8 qua
WASM nhúng + purego, nên khâu build tĩnh (`CGO_ENABLED=0`) **giữ nguyên**. Kết quả đo:
WebP nhỏ hơn JPEG **76–82%** ở cả ba cỡ.

**Không phóng to ảnh nhỏ.** `skipVariant` bỏ qua cỡ nào lớn hơn ảnh gốc — tranh gốc nhỏ sẽ
có ít biến thể hơn, và không bao giờ có biến thể "to hơn bản gốc" vô nghĩa.

**Sinh song song theo số CPU**, vì đây là công việc nặng CPU thuần tuý.

**Lỗi sinh biến thể không làm hỏng upload.** Nếu khâu này lỗi, ảnh gốc vẫn lên S3 và tác
phẩm vẫn tạo được — chỉ là không có biến thể, và phía hiển thị tự lui về ảnh gốc.

Kết quả lưu ở cột JSON `artworks.variants`; `thumbnail_url` vẫn được ghi song song (trỏ
`thumb_jpg`) để code cũ và trang admin không gãy. Xem
[01-database.md](./01-database.md).

Phía frontend, `web/src/lib/artworkImage.ts` gom toàn bộ logic chọn ảnh vào một chỗ, với
thứ tự lui rõ ràng: **biến thể đúng cỡ → `thumbnail_url` → `image_url`**. WebP đi qua
`<source>` trong `<picture>`, `src` luôn là JPEG hoặc ảnh gốc để trình duyệt nào cũng đọc
được.

## 7. Tải ảnh về (đường ngược lại)

Ảnh không chỉ đi lên — cả admin lẫn khách đều tải được ảnh gốc, và cả hai đều **đi qua
backend** thay vì trỏ thẳng vào S3:

| Endpoint | Watermark | Đòi `is_published` | Dùng cho |
|---|---|---|---|
| `GET /api/v1/public/artworks/{id}/download` | **Có** | **Có** | Khách xem triển lãm |
| `GET /api/v1/admin/artworks/{id}/download` | Không | Không | Admin cần file gốc sạch |

Vì sao proxy qua backend chứ không trả link S3: cùng origin nên không phụ thuộc CORS của
bucket, và đó cũng là chỗ duy nhất chèn được watermark cùng ghi nhật ký lượt tải (bảng
`artwork_downloads`).

⚠️ **Bẫy vận hành**: ảnh mốc watermark đọc theo **đường dẫn tương đối**
(`web/public/images/vas-white-mark.png`, lui về `web/dist/images/...`). Chạy binary từ thư
mục khác thì watermark bị bỏ qua **im lặng** — chỉ ghi log cảnh báo, ảnh vẫn trả về bình
thường (fail-open, cố ý: một khâu trang trí hỏng không đáng làm hỏng cả lượt tải). Đây là lý
do systemd unit bắt buộc đặt `WorkingDirectory`; xem
[deploys/00-tu-dau-den-cuoi.md](../deploys/00-tu-dau-den-cuoi.md).

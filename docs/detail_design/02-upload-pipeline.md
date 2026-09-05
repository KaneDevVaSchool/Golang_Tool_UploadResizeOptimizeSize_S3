# 02 — Pipeline upload

Ba đường upload cùng tồn tại, phục vụ ba tình huống khác nhau:

| Đường | Endpoint | Dùng khi | Giới hạn |
|---|---|---|---|
| Đơn | `POST /api/v1/upload` | File thường, phổ biến nhất | `UPLOAD_MAX_SIZE_MB` (20MB) |
| Có transaction | `POST /api/v1/upload-transaction` | Cần audit trail trong DB | 20MB, yêu cầu DB bật |
| Chunked | `POST /api/v1/upload/{init,chunk,complete,abort}` | File lớn | `UPLOAD_ABSOLUTE_MAX_MB` (200MB) |

## 1. Nhiều lớp kiểm tra

Một file phải qua bốn lớp trước khi lên S3. Mỗi lớp chặn một kiểu tấn công/lỗi khác nhau —
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
                          ↓
Lớp 4 — Nội dung thật     ValidateFileContent()  ⭐ chỉ áp dụng cho chunked
                          đọc 512 byte đầu, http.DetectContentType, đối chiếu đuôi file
```

⚠️ **Bất đối xứng đáng chú ý**: lớp 4 (magic byte) hiện **chỉ chạy ở đường chunked**
(`chunk_upload.go`, bước Complete). Đường upload đơn không kiểm tra nội dung thật — file
`.jpg` chứa nội dung khác vẫn lên được S3. Rủi ro thực tế thấp vì bucket chỉ phục vụ ảnh
tĩnh và không thực thi nội dung, nhưng đây là điểm không nhất quán nên biết.
Xem [plan/02-roadmap.md](../plan/02-roadmap.md).

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

## 5. Upload có transaction

Khác biệt duy nhất so với upload đơn: bọc trong transaction MySQL và ghi bản ghi audit.

```text
BEGIN
  ├─ INSERT uploads(status='pending')            ← trước khi đụng S3
  ├─ ghi temp file → upload S3
  │    lỗi ở bất kỳ bước nào → UPDATE status='failed', error=... → ROLLBACK
  └─ UPDATE uploads(status='completed', s3_key, s3_url)
COMMIT
```

Thứ tự này cho phép phát hiện bất thường: bản ghi kẹt ở `pending` nghĩa là tiến trình chết
giữa chừng, có thể có file mồ côi trên S3.

Xử lý panic đúng cách (`upload_service.go:229-244`): `defer` bắt `recover()`, rollback,
rồi **panic lại** — không nuốt panic, nhưng cũng không để transaction treo.

⚠️ Transaction chỉ bảo vệ **phía DB**. Nếu S3 upload thành công nhưng commit thất bại, file
đã nằm trên S3 mà không có bản ghi — rác S3. Đây là giới hạn cố hữu khi phối hợp hai hệ
thống không chung transaction; chấp nhận và bù bằng việc dọn rác định kỳ.

## 6. Upload chunked

### Vòng đời phiên

```text
init ──▶ [nhận chunk 0..N-1 theo thứ tự bất kỳ] ──▶ complete ──▶ (phiên bị xoá)
  │                                                     │
  └──────────────── abort ◀─────────────────────────────┘
                     hoặc hết TTL 45 phút
```

### Tham số cố định (`chunk_upload.go:22-27`)

| Hằng số | Giá trị | Ý nghĩa |
|---|---|---|
| `chunkSessionTTL` | 45 phút | Phiên quá hạn bị dọn |
| `chunkCleanupEvery` | 5 phút | Chu kỳ quét |
| `maxChunkSessions` | 64 | Trần phiên đồng thời toàn server |
| `maxChunksPerUpload` | 64 | Trần số phần mỗi file |

`maxChunksPerUpload` liên đới với config: `ConfigBuilder.WithUpload()` tự **hạ**
`AbsoluteMaxSize` xuống `maxSize × 64` nếu người dùng đặt tỷ lệ vượt quá
(`builder.go:98-103`). Nghĩa là đặt `UPLOAD_MAX_SIZE_MB=20` thì trần tuyệt đối không bao giờ
vượt 1280MB dù `.env` ghi lớn hơn.

### Kiểm tra từng phần

Mỗi chunk phải đúng kích thước kỳ vọng: `chunkSize` cho mọi phần, riêng phần cuối là
`totalSize - index × chunkSize`. Có ba tình huống được xử lý riêng:

- **Client không khai báo size** (`FileHeader.Size == 0` ở một số client): bỏ qua kiểm tra
  khai báo, vẫn kiểm tra byte thực ghi được.
- **Ghi thiếu byte**: xoá file tạm, báo lỗi — không để chunk hỏng nằm lại.
- **Ghi nguyên tử**: ghi ra `part_%06d.tmp` rồi `os.Rename` sang `part_%06d`. Rename là
  nguyên tử trên cùng filesystem, nên không bao giờ tồn tại chunk ghi dở mang tên thật.

### Ghép và hoàn tất

```text
Complete:
  ├─ đánh dấu Completing=true (chặn nhận chunk mới, chặn Complete song song)
  ├─ kiểm tra đủ chunk 0..N-1
  ├─ ghép tuần tự vào file tạm "assembled-*"
  ├─ đối chiếu tổng byte == totalSize khai báo
  ├─ ValidateFileContent  ⭐ magic byte — chỉ đường này có
  ├─ upload S3
  └─ xoá phiên + thư mục chunks/<id>
```

Nếu bất kỳ bước nào lỗi, cờ `Completing` được **đặt lại false** (`chunk_upload.go:290-297`)
để client thử lại được — không khoá cứng phiên vì một lần mạng chập chờn.

### Vì sao phiên nằm trong bộ nhớ

Chunk session **cố ý không lưu DB**, trái ngược với session admin. Lý do: vòng đời chỉ vài
phút, mất khi restart là chấp nhận được (client upload lại), và ghi DB cho mỗi chunk là chi
phí không đáng. Đánh đổi rõ ràng: ⚠️ deploy giữa lúc ai đó đang upload file 200MB sẽ làm
họ mất công. Với tần suất deploy hiện tại, chấp nhận được.

## 7. Bulk upload (dùng cho admin)

`ArtworkService.BulkUploadToS3` (`artwork_service.go:136`) upload nhiều file song song:

- **Giới hạn đồng thời 5** qua semaphore — tránh áp đảo S3/băng thông khi admin chọn hàng
  chục ảnh.
- **Lỗi từng file không làm hỏng cả lô**: mỗi item có trường `Error` riêng; FE hiển thị
  file nào hỏng, file nào xong.
- **Đọc kích thước ảnh trước** (`DecodeImageDimensions`) rồi `Seek(0)` để dùng lại reader
  cho việc upload — không đọc file hai lần từ đĩa.
- Chỉ đẩy lên S3, **chưa ghi bảng `artworks`**. Xem
  [03-artwork-domain.md](./03-artwork-domain.md) cho bước 2.
- **Sinh biến thể ảnh ngay trong bước này** (`artwork_service.go:178`) — xem mục 9.

## 9. Sinh biến thể ảnh

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

## 8. Pipeline resize kiểu WordPress

Đường độc lập hoàn toàn với S3 — lưu **đĩa local**, phục vụ qua `/wp-content/uploads/`.

```text
POST /api/v1/wp-upload
  → ImageResizeService: sinh nhiều kích thước theo WORDPRESS_IMAGE_SIZES
  → ImageOptimizer (nếu bật): JPEG/PNG quality, WebP tuỳ chọn
  → lưu wp-uploads/, trả danh sách URL theo từng kích thước
```

Chất lượng JPEG **thích ứng theo dung lượng** (`service/constants.go`): file lớn hơn 1MB
giảm 15 điểm chất lượng, trên 500KB giảm 10, dưới 100KB tăng 5 — luôn kẹp trong khoảng
70–95. Mục tiêu: file lớn nén mạnh hơn vì ở đó tiết kiệm được nhiều nhất, file nhỏ giữ nét.

Tính năng này bật mặc định (`WORDPRESS_ENABLED=true`) nhưng **không phục vụ trang public** —
trang public chỉ dùng ảnh S3. Nếu không tích hợp WordPress, có thể tắt để giảm bề mặt tấn
công và không phải bảo vệ thư mục static.

# 03 — Rủi ro

Rủi ro đã xác định, xếp theo **mức độ ảnh hưởng × khả năng xảy ra**. Mỗi mục ghi rõ dấu
hiệu nhận biết sớm và cách xử lý.

Bối cảnh chi phối: sự kiện có **thời hạn cứng** và **đỉnh truy cập ngắn** vào lúc công bố
kết quả. Rủi ro nào có thể phát tác đúng thời điểm đó được ưu tiên cao hơn.

---

## R1 — Trang public 401 vì API key ở production 🔴

**Khả năng**: Cao — xảy ra ngay khi làm đúng theo hướng dẫn production.
**Ảnh hưởng**: Nghiêm trọng — toàn bộ trang public ngừng hoạt động với khách.

Production **bắt buộc** `API_REQUIRE_KEY=true`, mà middleware chỉ miễn `/api/v1/health`.
Khách ẩn danh nhận 401 trên mọi lệnh gọi `/api/v1/public/*`.

**Dấu hiệu**: mở trang public bằng tab ẩn danh thấy trang trắng hoặc lỗi tải dữ liệu.

**Xử lý**: P0.1 trong [02-roadmap.md](./02-roadmap.md) — miễn trừ tiền tố
`/api/v1/public/`.

**Tạm thời**: kiểm tra trang public bằng trình duyệt ẩn danh **trước** khi công bố tên
miền. Đừng chỉ kiểm tra bằng máy đã đăng nhập admin.

---

## R2 — Đĩa đầy vì log không được dọn 🔴

**Khả năng**: Trung bình–Cao theo thời gian.
**Ảnh hưởng**: Nghiêm trọng — đĩa đầy làm sập cả MySQL lẫn ứng dụng.

Log ghi mỗi ngày một file và **không bao giờ tự xoá**. Thư mục `storage/logs` hiện đã có
log từ tháng 8.

**Dấu hiệu**: `df -h` cho thấy phân vùng gần đầy; ghi DB bắt đầu lỗi.

**Xử lý**: P0.3 — cấu hình `logrotate`.

**Theo dõi**: đặt cảnh báo khi đĩa vượt 80%.

---

## R3 — Mất dữ liệu do chưa có sao lưu tự động 🔴

**Khả năng**: Thấp.
**Ảnh hưởng**: Rất nghiêm trọng — mất toàn bộ metadata tác phẩm, bình luận, cảm xúc.

Ảnh nằm trên S3 nên an toàn, nhưng **toàn bộ metadata nằm trong MySQL**. Hiện chỉ có một
bản dump thủ công trong `storage/backups/`.

Nếu mất database: ảnh vẫn còn trên S3 nhưng **không biết tranh nào của ai** — coi như mất
sạch, vì tên file S3 không chứa thông tin học sinh.

**Xử lý**: cron sao lưu hằng ngày (xem [deploys/03-operations.md](../deploys/03-operations.md)).

⚠️ **Sao lưu chưa từng thử phục hồi thì không phải sao lưu.** Hãy thử phục hồi vào một
database tạm ít nhất một lần.

---

## R4 — Ảnh mồ côi tích tụ trên S3 🟡

**Khả năng**: Cao — đã đang xảy ra.
**Ảnh hưởng**: Trung bình — tốn phí lưu trữ, gây nhầm lẫn khi đối soát.

Hai nguồn sinh rác:

1. **Xoá tác phẩm** — cố ý không xoá file S3 (an toàn hơn khi bấm nhầm).
2. **Bulk upload bỏ dở** — admin đẩy ảnh lên S3 rồi đóng trình duyệt trước khi nhập metadata.

**Dấu hiệu**: số object trên S3 nhiều hơn hẳn số dòng `artworks`.

**Xử lý**: P2.2 — công cụ đối soát, mặc định chỉ báo cáo.

⚠️ **Không tự động xoá.** Rủi ro xoá nhầm ảnh vừa upload chưa kịp tạo bản ghi lớn hơn lợi
ích tiết kiệm dung lượng. Luôn có người xem lại danh sách.

---

## R5 — `artwork_views` phình to làm chậm trang 🟡

**Khả năng**: Cao khi lượt truy cập tăng.
**Ảnh hưởng**: Trung bình — mỗi lượt xem tranh chậm dần.

Bảng chỉ tăng, và truy vấn chống trùng chạy ở **mỗi lượt xem chi tiết tranh**. Đúng vào lúc
công bố kết quả — khi lượng truy cập cao nhất — bảng cũng lớn nhất.

**Dấu hiệu**: `SELECT COUNT(*) FROM artwork_views` tăng nhanh; API chi tiết tranh chậm dần.

**Xử lý**: P1.1 — dọn định kỳ.

---

## R6 — Đỉnh truy cập lúc công bố kết quả 🟡

**Khả năng**: Chắc chắn xảy ra.
**Ảnh hưởng**: Trung bình–Cao tuỳ mức chuẩn bị.

Khi công bố giải, nhiều phụ huynh vào cùng lúc. Ba điểm nghẽn theo thứ tự khả năng:

| Điểm nghẽn | Vì sao |
|---|---|
| Rate limit chung 100 req/phút/IP | ⚠️ **Nguy hiểm nhất** — trường học dùng chung một IP ra ngoài |
| Bảng vàng gọi lặp theo từng giải | Số truy vấn tăng theo số giải, kèm enrich mỗi lần |
| N+1 truy vấn học sinh | 24 truy vấn thừa mỗi lần tải danh sách |

**Chuẩn bị trước sự kiện**:

- [ ] Sửa P1.2 (N+1) và cân nhắc thêm cache ngắn cho bảng vàng
- [ ] **Xem lại giới hạn rate limit** — cân nhắc nâng, vì nhiều người sau cùng một IP NAT
- [ ] Kiểm tra Nginx truyền `X-Forwarded-For` (nếu thiếu, mọi người tính chung một IP →
      cả trang bị khoá)
- [ ] Thử tải trước với công cụ đo tải
- [ ] Cân nhắc đặt CDN/Cloudflare trước tên miền

---

## R7 — Cấu hình sai lúc triển khai 🟡

**Khả năng**: Trung bình.
**Ảnh hưởng**: Trung bình — phát hiện được nhanh nếu theo đúng danh sách kiểm tra.

Các lỗi hay gặp nhất, theo thứ tự:

| Lỗi | Hậu quả | Phòng tránh |
|---|---|---|
| `GOOGLE_REDIRECT_URL` lệch một ký tự | Không đăng nhập admin được | Chép dán, không gõ tay |
| `S3_USE_PRESIGNED_URL=true` | Ảnh public hỏng sau 60 phút | Danh sách kiểm tra |
| Thiếu `X-Forwarded-For` | Rate limit khoá nhầm cả trang | Dùng vhost mẫu trong repo |
| `client_max_body_size` nhỏ hơn giới hạn app | Upload lỗi 413 | Dùng vhost mẫu |
| Còn vhost cũ chiếm tên miền | Thấy giao diện cũ | `ls /etc/nginx/sites-enabled/` |
| Quên `CSRF_SECURE_COOKIE=true` sau khi bật HTTPS | Cookie kém an toàn | Danh sách kiểm tra |

**Xử lý**: theo danh sách trong [deploys/README.md](../deploys/README.md). Bốn lỗi nghiêm
trọng nhất đã được server **tự chặn khởi động**.

---

## R8 — Lạm dụng tương tác ẩn danh 🟡

**Khả năng**: Trung bình.
**Ảnh hưởng**: Trung bình — số liệu sai lệch, bình luận không phù hợp.

`visitor_token` giả mạo được. Người có kỹ thuật cơ bản có thể thổi phồng lượt cảm xúc, hoặc
đăng bình luận không phù hợp dưới tên bất kỳ.

**Đang có**: rate limit 20 req/phút/IP cho thao tác ghi; ràng buộc UNIQUE chống trùng; giới
hạn độ dài.

**Chưa có**: kiểm duyệt bình luận qua giao diện (P2.1), lọc từ khoá, chặn theo IP.

⚠️ **Đặc biệt nhạy cảm vì đối tượng là học sinh.** Nên có người trực theo dõi bình luận
trong những ngày cao điểm, và P2.1 nên xong **trước** khi công bố kết quả để xử lý nhanh
khi cần.

---

## R9 — Mất phiên upload khi triển khai 🟢

**Khả năng**: Thấp.
**Ảnh hưởng**: Thấp — người dùng upload lại.

Phiên chunk lưu trong bộ nhớ, mất khi khởi động lại.

**Xử lý**: triển khai vào giờ thấp điểm. Không cần sửa code — chi phí lưu phiên vào DB lớn
hơn lợi ích.

---

## R10 — Nhiều instance cùng chạy migration 🟢

**Khả năng**: Rất thấp — hiện chỉ chạy một instance.
**Ảnh hưởng**: Cao nếu xảy ra — schema hỏng.

Migration chạy lúc khởi động, không có khoá. Hai instance khởi động cùng lúc có thể chạy
song song.

**Xử lý**: chỉ cần khi mở rộng nhiều instance. Khi đó thêm advisory lock hoặc tách migration
khỏi luồng khởi động.

---

## R11 — `ADD COLUMN IF NOT EXISTS` gây lỗi cú pháp trên MySQL 8.1 (ServBay) 🟡

**Khả năng**: Chắc chắn xảy ra nếu còn dùng cú pháp này — không phải rủi ro xác suất, mà là
lỗi đã xảy ra thật (migration 015, 016 khiến server không khởi động được cho tới khi sửa).
**Ảnh hưởng**: Cao khi xảy ra — server không start được, toàn bộ API 404/không phản hồi.

MySQL chính thức hỗ trợ `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` từ 8.0.29, nhưng bản
MySQL 8.1.0 cài qua ServBay dùng ở máy dev báo lỗi cú pháp 1064 với cú pháp này (đã kiểm
chứng trực tiếp bằng `mysql` CLI, không phải lỗi driver Go). Chưa xác định được đây là đặc
thù build ServBay hay khác biệt phiên bản/`sql_mode` nào khác — chỉ biết là đo được thật
trên môi trường dev hiện tại. `DROP COLUMN IF EXISTS` cũng lỗi tương tự.

**Xử lý**: không dùng `ADD COLUMN IF NOT EXISTS` / `DROP COLUMN IF EXISTS` trong migration
mới. Dùng `ADD COLUMN` trần, dựa vào bảng `schema_migrations` để đảm bảo idempotency (không
chạy lại file đã áp dụng) — cùng cách `014_add_perf_indexes.sql` đã làm với `CREATE INDEX`.
Trước khi viết migration có DDL mới, thử chạy trực tiếp qua `mysql` CLI trên môi trường dev
thật thay vì tin vào tài liệu phiên bản MySQL chính thức.

---

## Bảng tổng hợp

| Mã | Rủi ro | Mức | Xử lý |
|---|---|---|---|
| R1 | API key chặn trang public | 🔴 | P0.1 |
| R2 | Đĩa đầy vì log | 🔴 | P0.3 |
| R3 | Chưa sao lưu tự động | 🔴 | Cron sao lưu |
| R4 | Ảnh mồ côi trên S3 | 🟡 | P2.2 |
| R5 | `artwork_views` phình to | 🟡 | P1.1 |
| R6 | Đỉnh truy cập | 🟡 | P1.2 + xem lại rate limit |
| R7 | Cấu hình sai | 🟡 | Danh sách kiểm tra |
| R8 | Lạm dụng ẩn danh | 🟡 | P2.1 + trực theo dõi |
| R9 | Mất phiên upload | 🟢 | Chấp nhận |
| R10 | Migration song song | 🟢 | Chấp nhận ở quy mô hiện tại |
| R11 | `ADD COLUMN IF NOT EXISTS` lỗi cú pháp MySQL 8.1 | 🟡 | Đã sửa 015/016, tránh cú pháp này về sau |

## Ba việc cần làm trước khi công bố

Nếu chỉ làm được ba việc, hãy làm ba việc này:

1. **Sửa R1** — không thì trang public không chạy được với cấu hình production.
2. **Bật sao lưu tự động (R3)** — không thì một sự cố là mất sạch dữ liệu hội thi.
3. **Xem lại rate limit và `X-Forwarded-For` (R6)** — không thì trang có thể tự khoá đúng
   lúc đông người nhất.

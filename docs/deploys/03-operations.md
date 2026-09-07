# 03 — Vận hành

Việc cần làm sau khi hệ thống đã chạy: theo dõi, sao lưu, xử lý sự cố, quay lui.

## 1. Lệnh dùng hằng ngày

```bash
# Trạng thái và log
sudo systemctl status s3-upload-tool
sudo journalctl -u s3-upload-tool -f              # theo dõi trực tiếp
sudo journalctl -u s3-upload-tool -n 200 --no-pager
tail -f /opt/s3-upload-tool/storage/logs/app-$(date +%F).log

# Vòng đời service
sudo systemctl restart s3-upload-tool
sudo systemctl stop s3-upload-tool
sudo systemctl start s3-upload-tool

# Kiểm tra sức khoẻ
curl http://127.0.0.1:8080/api/v1/health
curl https://pictures.vaschools.edu.vn/api/v1/health
```

## 2. Theo dõi

### Điểm cuối sức khoẻ

`GET /api/v1/health` — không cần xác thực, dùng cho giám sát tự động.

### Chỉ số

`GET /api/v1/metrics` — số request, độ trễ, phân bố mã trạng thái. Giới hạn **10 req/phút**
(cố định trong code) và **cần API key** nếu đã bật.

### Nhật ký

Ghi ra hai nơi cùng lúc:

| Nơi | Cách xem | Xoay vòng |
|---|---|---|
| systemd journal | `journalctl -u s3-upload-tool` | journald tự quản lý |
| File theo ngày | `storage/logs/app-YYYY-MM-DD.log` | Tạo file mới mỗi ngày, ⚠️ **không tự xoá file cũ** |

Tiền tố log giúp lọc nhanh:

```bash
grep '\[Container\]'    storage/logs/app-$(date +%F).log   # khởi động, tắt máy
grep '\[UploadService\]' storage/logs/app-$(date +%F).log  # upload đơn
grep '\[AdminAuth\]'     storage/logs/app-$(date +%F).log  # đăng nhập, từ chối truy cập
grep '\[ArtworkService\]' storage/logs/app-$(date +%F).log # nghiệp vụ tác phẩm
grep '\[BotGuard\]'      storage/logs/app-$(date +%F).log  # chặn công cụ tải trọn site
grep 'watermark'         storage/logs/app-$(date +%F).log  # cảnh báo không tìm thấy ảnh mốc
```

Mọi dòng log kèm request ID để lần theo một request qua nhiều tầng.

### ⚠️ Dọn log — cần thiết lập thủ công

Không có cơ chế tự xoá. Thêm `logrotate`:

```bash
sudo tee /etc/logrotate.d/s3-upload-tool > /dev/null <<'EOF'
/opt/s3-upload-tool/storage/logs/app-*.log {
    weekly
    rotate 8
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    su appuser appuser
}
EOF

sudo logrotate -d /etc/logrotate.d/s3-upload-tool   # thử trước, không ghi gì
```

Dùng `copytruncate` vì ứng dụng giữ file mở suốt — đổi tên file sẽ khiến log tiếp tục chảy
vào file cũ đã bị đổi tên.

## 3. Sao lưu

### Cơ sở dữ liệu

Thư mục `storage/backups/` đã có sẵn cho mục đích này (hiện chứa một bản dump thủ công
trước thao tác xoá dữ liệu).

```bash
# Sao lưu thủ công trước mọi thao tác rủi ro
mysqldump -u vasapp -p va_stu_pic_db_prd \
  > /opt/s3-upload-tool/storage/backups/backup-$(date +%Y%m%d-%H%M%S).sql
```

Sao lưu tự động hằng ngày:

```bash
sudo crontab -e
```

```cron
0 2 * * * mysqldump -u vasapp -p'<mật-khẩu>' va_stu_pic_db_prd | gzip > /opt/s3-upload-tool/storage/backups/auto-$(date +\%Y\%m\%d).sql.gz
0 3 * * * find /opt/s3-upload-tool/storage/backups -name 'auto-*.sql.gz' -mtime +30 -delete
```

⚠️ Mật khẩu trong crontab lộ qua `ps`. An toàn hơn: dùng file `~/.my.cnf` với quyền `600`.

Phục hồi:

```bash
sudo systemctl stop s3-upload-tool
mysql -u vasapp -p va_stu_pic_db_prd < backup-YYYYMMDD-HHMMSS.sql
sudo systemctl start s3-upload-tool
```

### Ảnh trên S3

Ảnh nằm trên S3, đã bền vững sẵn. Nên bật thêm:

- **Versioning** trên bucket — khôi phục được khi xoá nhầm.
- **Lifecycle policy** chuyển ảnh cũ sang lớp lưu trữ rẻ hơn nếu chi phí thành vấn đề.

### `.env`

Chứa bí mật, **không nằm trong git**. Lưu một bản ở nơi an toàn (trình quản lý mật khẩu của
tổ chức). Mất file này là phải cấu hình lại toàn bộ OAuth và khoá API.

## 4. Chẩn đoán sự cố

### Service không khởi động

```bash
sudo journalctl -u s3-upload-tool -n 50 --no-pager
```

| Thông báo | Nguyên nhân | Cách xử lý |
|---|---|---|
| `production requires API_REQUIRE_KEY=true...` | `APP_ENV=production` nhưng thiếu khoá | Đặt `API_KEY` và `API_REQUIRE_KEY=true` |
| `production requires an explicit CORS_ORIGINS...` | `CORS_ORIGINS=*` | Ghi rõ tên miền |
| `failed to initialize database` | MySQL chưa chạy / sai thông tin đăng nhập | `systemctl status mysql`, kiểm tra `DATABASE_URL` |
| `failed to run database migrations` | Thiếu quyền hoặc SQL lỗi | Kiểm tra quyền của `vasapp` trên database |
| `bind: address already in use` | Cổng 8080 đã bị chiếm | `sudo lsof -i :8080` |
| `TRUSTED_PROXIES có giá trị không hợp lệ` | Sai định dạng CIDR trong `.env` | Sửa thành dạng `127.0.0.1/32`; app vẫn chạy nhưng bỏ qua dòng sai |
| `không đọc được ảnh mốc watermark` | Chạy sai thư mục, hoặc chưa build `web/` | Kiểm tra `WorkingDirectory` của systemd; chạy `npm run build` trong `web/` |

### Nginx trả 502

Nghĩa là Nginx chạy nhưng ứng dụng phía sau không trả lời.

```bash
curl http://127.0.0.1:8080/api/v1/health   # ứng dụng có sống không?
sudo systemctl status s3-upload-tool
```

Nếu ứng dụng khoẻ mà vẫn 502, kiểm tra `proxy_pass` trong vhost trỏ đúng `127.0.0.1:8080`.

### Vẫn thấy giao diện cũ

Theo thứ tự khả năng:

1. **Vhost khác đang chiếm tên miền** — nguyên nhân phổ biến nhất:

   ```bash
   ls /etc/nginx/sites-enabled/
   sudo rm /etc/nginx/sites-enabled/<vhost-cũ>
   sudo systemctl reload nginx
   ```

2. **Trình duyệt cache** — thử tab ẩn danh hoặc `Ctrl+Shift+R`.
3. **CDN/Cloudflare cache** — xoá cache ở đó.
4. **DNS chưa trỏ đúng** — `dig +short pictures.vaschools.edu.vn`.
5. **`web/dist` chưa build lại** — chạy lại `npm run build`.

### Upload thất bại

| Triệu chứng | Nguyên nhân | Cách xử lý |
|---|---|---|
| 413 từ Nginx | `client_max_body_size` nhỏ hơn file | Nâng trong vhost, phải ≥ `UPLOAD_ABSOLUTE_MAX_MB` |
| Ngắt kết nối khi upload file lớn | `proxy_read_timeout` quá ngắn | Nâng lên `300s` |
| `FILE_TOO_LARGE` từ ứng dụng | Vượt `UPLOAD_MAX_SIZE_MB` | Nâng `UPLOAD_MAX_SIZE_MB` (và `client_max_body_size` của Nginx cho khớp) |
| Ảnh tải về không có watermark | Chạy sai thư mục hoặc chưa build `web/` | Xem dòng cảnh báo `watermark` trong log — lỗi này **không** làm hỏng lượt tải nên dễ bỏ sót |
| Khách bị chặn 403 `AUTOMATED_ACCESS_BLOCKED` | BotGuard nhận nhầm | Kiểm tra `TRUSTED_PROXIES` trước tiên: sai dòng đó thì mọi khách gộp thành một IP và cùng vượt ngưỡng |
| Lỗi ký request S3 | Sai region/khoá | Xem log `[Container]` dòng region đã dò được |
| `403` khi xem ảnh | Bucket chưa mở quyền đọc | Xem [S3-PUBLIC-READ.md](../S3-PUBLIC-READ.md) |
| Ảnh hiển thị một lúc rồi hỏng | Đang bật presigned URL | Đặt `S3_USE_PRESIGNED_URL=false` |

### Không đăng nhập admin được

| Triệu chứng | Nguyên nhân |
|---|---|
| 503 `OAUTH_NOT_CONFIGURED` | Thiếu `GOOGLE_CLIENT_ID`/`SECRET` |
| Google báo `redirect_uri_mismatch` | `GOOGLE_REDIRECT_URL` không khớp **chính xác** Console |
| Quay lại login kèm `email_not_allowed` | Email ngoài danh sách cho phép |
| Quay lại login kèm `email_not_verified` | Tài khoản Google chưa xác minh email |
| Quay lại login kèm `invalid_state` | Cookie state bị chặn/quá 5 phút; thử lại |
| Đăng nhập xong lại bị đăng xuất | `CSRF_SECURE_COOKIE=true` nhưng site chạy HTTP |

Log `[AdminAuth]` ghi rõ email bị từ chối và lý do.

### Trang public trả 401

Gần như chắc chắn do `API_REQUIRE_KEY=true` áp lên cả `/api/v1/public/*`. Xem phần cảnh báo
trong [02-configuration.md](./02-configuration.md).

### Lỗi 429 quá nhiều

Kiểm tra Nginx có truyền `X-Forwarded-For` không:

```bash
grep -r 'X-Forwarded-For' /etc/nginx/sites-enabled/
```

⚠️ Thiếu header này thì **mọi** request trông như đến từ `127.0.0.1`, và một người truy cập
nhiều sẽ làm cả trang bị khoá với tất cả mọi người.

## 5. Quay lui

Cách nhanh nhất là đổi lại binary đã giữ từ lần deploy trước — không cần build lại:

```bash
cd /opt/s3-upload-tool
sudo systemctl stop s3-upload-tool
sudo -u appuser cp server.prev server
sudo systemctl start s3-upload-tool
curl -sf http://127.0.0.1:8080/api/v1/health && echo OK
```

Nếu cần quay lui cả mã nguồn và build lại, xem giai đoạn K trong
[00-tu-dau-den-cuoi.md](./00-tu-dau-den-cuoi.md):

```bash
cd /opt/s3-upload-tool
git log --oneline -10
sudo -u appuser git checkout <commit-ổn-định>
# rồi build lại theo H1-H3 và restart service
```

⚠️ **Migration không quay lui tự động.** Không có cơ chế `down`. Nếu bản mới đã thêm bảng/cột
thì việc quay lui code có thể gặp lỗi lệch schema. Với các migration chỉ *thêm* bảng mới
(trường hợp hiện tại), quay lui code thường an toàn vì code cũ đơn giản là không dùng bảng
mới đó.

Cách chắc chắn nhất: phục hồi bản sao lưu database tương ứng với thời điểm của code cũ.

## 6. Bảo trì định kỳ

| Việc | Tần suất | Ghi chú |
|---|---|---|
| Kiểm tra `certbot renew --dry-run` | Hằng tháng | Certbot tự gia hạn, đây chỉ là kiểm tra |
| Xem lại dung lượng đĩa | Hằng tháng | `df -h`, chú ý `storage/logs` và `uploads` |
| Kiểm tra bản sao lưu phục hồi được | Hằng quý | Sao lưu chưa thử phục hồi không phải là sao lưu |
| Cập nhật gói hệ thống | Hằng quý | `apt update && apt upgrade` |
| Dọn `artwork_views` cũ | 📋 Chưa có | Xem bên dưới |
| Dọn ảnh mồ côi trên S3 | 📋 Chưa có | Xem bên dưới |

### Hai việc dọn dẹp chưa được tự động hoá

**Bảng `artwork_views`** chỉ tăng, không bao giờ được xoá. Bản ghi cũ hơn 24 giờ không còn
tác dụng:

```sql
DELETE FROM artwork_views WHERE viewed_at < NOW() - INTERVAL 2 DAY;
```

Chạy vào giờ thấp điểm; nếu bảng đã rất lớn, xoá theo lô để tránh khoá bảng lâu.

**Ảnh mồ côi trên S3** giờ chỉ còn **một** nguồn: bulk upload bị bỏ dở giữa chừng (admin đẩy
ảnh lên S3 rồi đóng trình duyệt trước khi nhập thông tin tác phẩm). Nguồn thứ hai trước đây —
xoá tác phẩm cố ý giữ lại file — đã hết từ 2026-09-07: `DeleteArtwork` nay xoá cả object S3
(ảnh gốc, mọi biến thể và thumbnail) trước khi xoá bản ghi. Cách rà soát an toàn:

```sql
SELECT s3_key FROM artworks;   -- xuất ra file, đối chiếu với danh sách object trên S3
```

⚠️ **Luôn xem lại danh sách trước khi xoá.** Đối chiếu cả `uploads.s3_key`, và cân nhắc chỉ
xoá các object cũ hơn vài tuần để tránh xoá nhầm ảnh vừa upload mà chưa kịp tạo bản ghi.

Chi tiết hai rủi ro này trong [plan/03-risks.md](../plan/03-risks.md).

## 7. Thay đổi cấu hình

```bash
sudo nano /opt/s3-upload-tool/.env
sudo systemctl restart s3-upload-tool
sudo journalctl -u s3-upload-tool -n 30 --no-pager   # xác nhận khởi động sạch
```

`.env` chỉ đọc **một lần** lúc khởi động — không có nạp lại nóng.

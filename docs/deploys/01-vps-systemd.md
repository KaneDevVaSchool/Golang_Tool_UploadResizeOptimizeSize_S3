# 01 — Triển khai VPS: binary + systemd + Nginx

Quy trình đầy đủ từ một VPS trắng đến hệ thống chạy HTTPS.

**Đích đến**: `https://pictures.vaschools.edu.vn`
**Thư mục ứng dụng**: `/opt/s3-upload-tool`
**Tài khoản chạy**: `appuser` (tài khoản hệ thống, không có shell)

## 0. Điều kiện tiên quyết

| Yêu cầu | Cách kiểm tra |
|---|---|
| DNS trỏ đúng IP VPS | `dig +short pictures.vaschools.edu.vn` |
| Go 1.24+ | `go version` |
| Node 20+ | `node -v` |
| MySQL 8+ đang chạy | `sudo systemctl status mysql` |
| Nginx đã cài | `nginx -v` |
| Bucket S3 + quyền truy cập | `aws s3 ls s3://<bucket>` |

⚠️ **Kiểm tra DNS trước tiên.** Nếu DNS chưa trỏ đúng, bước xin chứng chỉ HTTPS ở mục 5 sẽ
thất bại và phải chờ thêm thời gian để bản ghi lan truyền.

Cài phần thiếu:

```bash
sudo apt update
sudo apt install -y nginx mysql-server git curl
# Go và Node cài theo hướng dẫn chính thức của từng bên
```

## 1. Chuẩn bị mã nguồn và cơ sở dữ liệu

```bash
sudo mkdir -p /opt/s3-upload-tool
sudo git clone <repo-url> /opt/s3-upload-tool
cd /opt/s3-upload-tool
```

Tạo database và tài khoản riêng cho ứng dụng:

```sql
CREATE DATABASE va_stu_pic_db_prd
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE USER 'vasapp'@'localhost' IDENTIFIED BY '<mật-khẩu-mạnh>';
GRANT ALL PRIVILEGES ON va_stu_pic_db_prd.* TO 'vasapp'@'localhost';
FLUSH PRIVILEGES;
```

⚠️ `CHARACTER SET utf8mb4` là **bắt buộc**, không phải tuỳ chọn — dữ liệu có tiếng Việt có
dấu và emoji trong bình luận. Dùng `utf8` (3 byte) sẽ làm hỏng emoji.

Không cần chạy migration thủ công: ứng dụng tự tạo bảng khi khởi động lần đầu
(`DATABASE_AUTO_MIGRATE=true`).

## 2. Tạo `.env`

```bash
sudo cp .env.example .env
sudo nano .env
```

Xem [02-configuration.md](./02-configuration.md) cho ý nghĩa từng biến. Tối thiểu phải sửa:

```env
APP_ENV=production

AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=<bucket-thật>
S3_USE_PRESIGNED_URL=false

DATABASE_ENABLED=true
DATABASE_URL=vasapp:<mật-khẩu>@tcp(localhost:3306)/va_stu_pic_db_prd

API_REQUIRE_KEY=true
API_KEY=<chuỗi ngẫu nhiên: openssl rand -base64 32>
CORS_ORIGINS=https://pictures.vaschools.edu.vn
CSRF_SECURE_COOKIE=true

GOOGLE_CLIENT_ID=<từ Google Cloud Console>
GOOGLE_CLIENT_SECRET=<từ Google Cloud Console>
GOOGLE_REDIRECT_URL=https://pictures.vaschools.edu.vn/auth/google/callback
ADMIN_ALLOWED_EMAILS=<danh sách email ban tổ chức, cách nhau bởi dấu phẩy>
```

Bảo vệ file:

```bash
sudo chmod 600 /opt/s3-upload-tool/.env
```

## 3. Thiết lập Google OAuth

1. Vào [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials).
2. Tạo **OAuth 2.0 Client ID**, loại "Web application".
3. Thêm **Authorized redirect URI** — phải khớp **từng ký tự** với `GOOGLE_REDIRECT_URL`:

   ```text
   https://pictures.vaschools.edu.vn/auth/google/callback
   ```

4. Chép Client ID/Secret vào `.env`.

⚠️ Sai lệch dù chỉ một ký tự (thiếu `s` trong `https`, thừa dấu `/` cuối) sẽ khiến Google
trả lỗi `redirect_uri_mismatch`. Đây là lỗi hay gặp nhất khi thiết lập.

Server vẫn khởi động bình thường nếu chưa cấu hình OAuth — chỉ `/auth/google/*` trả 503.

## 4. Triển khai

### Cách nhanh — dùng script

```bash
cd /opt/s3-upload-tool
sudo bash deploy/deploy.sh
```

Script thực hiện 6 bước và chạy lại được nhiều lần:

| Bước | Việc |
|---|---|
| 1 | `git pull` |
| 2 | `npm ci && npm run build` trong `web/` |
| 3 | Build binary Go tĩnh (`CGO_ENABLED=0`, `-trimpath -ldflags="-s -w"`) |
| 4 | Tạo `appuser` nếu chưa có, đặt quyền sở hữu thư mục |
| 5 | Cài + bật + khởi động lại systemd service |
| 6 | Cài + bật vhost Nginx, `nginx -t`, reload |

Cuối cùng script tự kiểm tra sức khoẻ qua `127.0.0.1:8080`. Nếu thất bại, nó thoát với mã
lỗi và chỉ ra lệnh xem log.

Script build ra `server.new` rồi mới `mv` đè lên `server` — thao tác đổi tên là nguyên tử,
nên không bao giờ tồn tại binary ghi dở.

### Cách thủ công — khi cần gỡ lỗi

```bash
# Build
cd /opt/s3-upload-tool/web && npm ci && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server ./cmd/server

# Người dùng và quyền
sudo useradd -r -s /sbin/nologin appuser
sudo chown -R appuser:appuser /opt/s3-upload-tool
sudo chmod +x server

# systemd
sudo cp deploy/systemd/s3-upload-tool.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now s3-upload-tool
sudo systemctl status s3-upload-tool     # phải là active (running)

# Ứng dụng phải trả lời trên localhost TRƯỚC khi đụng đến Nginx
curl http://127.0.0.1:8080/api/v1/health
```

⚠️ **Không cấu hình Nginx khi bước này chưa xong.** Đặt reverse proxy lên một service chưa
chạy chỉ tạo ra lỗi 502 và làm việc chẩn đoán rối thêm. Xem log:
`sudo journalctl -u s3-upload-tool -f`.

### Cài Nginx

```bash
sudo cp deploy/nginx/pictures.vaschools.edu.vn.conf /etc/nginx/sites-available/
sudo ln -sf /etc/nginx/sites-available/pictures.vaschools.edu.vn.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

⚠️ **Gỡ vhost cũ đang chiếm tên miền.** Đây là nguyên nhân phổ biến nhất của hiện tượng
"vẫn thấy giao diện cũ":

```bash
ls /etc/nginx/sites-enabled/
sudo rm /etc/nginx/sites-enabled/<vhost-cũ>       # kể cả 'default' nếu nó là default_server
sudo systemctl reload nginx
```

Những tham số quan trọng trong vhost và lý do:

| Tham số | Giá trị | Vì sao |
|---|---|---|
| `client_max_body_size` | `200m` | Phải ≥ `UPLOAD_ABSOLUTE_MAX_MB`, nếu không Nginx chặn trước khi ứng dụng thấy request |
| `proxy_read_timeout` | `300s` | Upload file lớn vượt xa mặc định 60s |
| `X-Forwarded-For` | bắt buộc | Thiếu thì rate limit tính mọi người là `127.0.0.1` — **một người spam khoá cả trang** |
| `X-Forwarded-Proto` | bắt buộc | Trang chia sẻ Open Graph dùng để dựng `og:url` đúng `https` |

## 5. Bật HTTPS

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d pictures.vaschools.edu.vn
```

Certbot tự thêm khối `listen 443 ssl` và chuyển hướng HTTP→HTTPS vào file vhost.

Sau khi có HTTPS, bật cookie bảo mật:

```bash
# trong /opt/s3-upload-tool/.env
CSRF_SECURE_COOKIE=true

sudo systemctl restart s3-upload-tool
```

Kiểm tra việc tự gia hạn chứng chỉ:

```bash
sudo certbot renew --dry-run
```

## 6. Nghiệm thu

```bash
# Sức khoẻ ứng dụng
curl https://pictures.vaschools.edu.vn/api/v1/health

# Trang public tải được
curl -I https://pictures.vaschools.edu.vn/

# Trang chia sẻ có thẻ Open Graph (thay 1 bằng id tác phẩm thật)
curl -s https://pictures.vaschools.edu.vn/chia-se/tac-pham/1 | grep 'og:image'
```

Kiểm tra bằng tay:

- [ ] Mở `https://pictures.vaschools.edu.vn` — thấy trang triển lãm, có ổ khoá HTTPS
- [ ] Vào `/admin/login`, đăng nhập Google thành công bằng email trong danh sách
- [ ] Đăng nhập bằng email **ngoài** danh sách → bị từ chối
- [ ] Upload thử một ảnh qua `/admin/artworks/upload`
- [ ] Ảnh hiển thị được trên trang public (kiểm tra quyền đọc S3)
- [ ] Thả cảm xúc và bình luận trên trang public

## 7. Cập nhật code về sau

```bash
cd /opt/s3-upload-tool
sudo bash deploy/deploy.sh
```

Không cần đụng Nginx khi chỉ cập nhật mã ứng dụng — vhost chỉ phải sửa nếu đổi cổng, tên
miền, hoặc giới hạn kích thước upload.

Ứng dụng tắt an toàn: `systemctl restart` gửi `SIGTERM`, ứng dụng chờ các request đang chạy
xong (tối đa `SERVER_SHUTDOWN_TIMEOUT_SECONDS`) rồi mới đóng DB.

⚠️ Người đang upload file lớn dở dang sẽ mất phiên chunk (lưu trong bộ nhớ). Nên deploy vào
giờ thấp điểm.

## 8. Khi gặp sự cố

Xem [03-operations.md](./03-operations.md) cho bảng chẩn đoán đầy đủ. Ba lệnh dùng nhiều nhất:

```bash
sudo journalctl -u s3-upload-tool -n 100 --no-pager   # log gần nhất
sudo systemctl status s3-upload-tool                   # trạng thái service
tail -f /opt/s3-upload-tool/storage/logs/app-$(date +%F).log
```

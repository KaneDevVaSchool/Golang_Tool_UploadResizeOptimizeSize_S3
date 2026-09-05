# Deploy — pictures.vaschools.edu.vn

Deploy bằng **binary Go + systemd + Nginx**. Không dùng Docker.

## TL;DR

```bash
# Lần đầu
sudo bash deploy/preflight.sh      # kiểm tra VPS đủ điều kiện chưa
sudo bash deploy/deploy.sh         # build + cài service + nginx
sudo bash deploy/setup-https.sh    # bật HTTPS

# Các lần cập nhật sau
cd /opt/s3-upload-tool && sudo bash deploy/deploy.sh

# Khi có sự cố
sudo bash deploy/status.sh
```

---

## Các file trong thư mục này

| File | Việc nó làm |
|---|---|
| `preflight.sh` | Kiểm tra VPS: Go/Node/Nginx, `.env`, DNS, cổng, đĩa. Chỉ đọc, không sửa gì. |
| `deploy.sh` | Build web + binary, chạy migration, cài service/nginx, health-check, **tự rollback** nếu hỏng. |
| `setup-https.sh` | Xin chứng chỉ Let's Encrypt, bật redirect HTTPS, bật `CSRF_SECURE_COOKIE`. |
| `status.sh` | Xem service, health, chứng chỉ, phiên bản đang chạy, log gần nhất. |
| `systemd/*.service` | Định nghĩa service (auto-restart, graceful shutdown, sandbox bảo mật). |
| `nginx/*.conf` | Vhost: reverse proxy, gzip, giới hạn upload 200MB, security header. |
| `config.env.example` | Đổi domain/đường dẫn/port mặc định nếu cần. |

---

## Cài đặt lần đầu (chi tiết)

### 1. Chuẩn bị VPS

Ubuntu/Debian, có quyền sudo:

```bash
sudo apt update
sudo apt install -y git nginx curl

# Go 1.24+
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc

# Node 20+ (để build giao diện)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
```

### 2. Lấy mã nguồn

```bash
sudo git clone <repo-url> /opt/s3-upload-tool
cd /opt/s3-upload-tool
```

### 3. Tạo `.env`

```bash
sudo cp .env.example .env
sudo nano .env
```

Bắt buộc cho production — **thiếu là server không khởi động**:

```env
APP_ENV=production
CORS_ORIGINS=https://pictures.vaschools.edu.vn   # KHÔNG để "*"
API_REQUIRE_KEY=true
API_KEY=<sinh bằng: openssl rand -hex 32>

AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=<tên bucket>
AWS_ACCESS_KEY_ID=<...>            # bỏ qua nếu VPS là EC2 có IAM role
AWS_SECRET_ACCESS_KEY=<...>

CSRF_SECURE_COOKIE=true            # setup-https.sh sẽ tự bật

# Database MySQL (nếu dùng gallery/admin)
DATABASE_ENABLED=true
DATABASE_URL=user:pass@tcp(localhost:3306)/va_stu_pic_db_prd
SESSION_SECRET=<openssl rand -hex 32>

# Đăng nhập admin bằng Google
GOOGLE_CLIENT_ID=<...>
GOOGLE_CLIENT_SECRET=<...>
GOOGLE_REDIRECT_URL=https://pictures.vaschools.edu.vn/auth/google/callback
ADMIN_ALLOWED_EMAIL_DOMAIN=vaschools.edu.vn
```

> `GOOGLE_REDIRECT_URL` phải trùng **chính xác** với Authorized redirect URI
> khai trong Google Cloud Console, kể cả `https://` và dấu `/` cuối.

### 4. Kiểm tra rồi deploy

```bash
sudo bash deploy/preflight.sh     # sửa hết mục ✗ trước khi đi tiếp
sudo bash deploy/deploy.sh
```

### 5. Bật HTTPS

DNS phải trỏ đúng IP VPS trước.

```bash
sudo bash deploy/setup-https.sh
```

---

## Cập nhật code

```bash
cd /opt/s3-upload-tool
sudo bash deploy/deploy.sh
```

Tuỳ chọn:

```bash
sudo bash deploy/deploy.sh --skip-web   # chỉ sửa Go, khỏi build lại UI (nhanh hơn nhiều)
sudo bash deploy/deploy.sh --no-pull    # deploy code đang có, không git pull
```

`deploy.sh` giữ lại binary cũ (`server.prev`). Nếu bản mới không qua
health-check, script tự khôi phục bản cũ và restart để site không chết.

---

## Vận hành

```bash
sudo bash deploy/status.sh                      # tổng quan
sudo systemctl restart s3-upload-tool           # khởi động lại
sudo journalctl -u s3-upload-tool -f            # log realtime
sudo journalctl -u s3-upload-tool -p err -n 50  # chỉ lỗi
```

Log ứng dụng dạng file: `/opt/s3-upload-tool/storage/logs/app-YYYY-MM-DD.log`

---

## Xử lý sự cố

**Service không lên**
```bash
sudo journalctl -u s3-upload-tool -n 50 --no-pager
```
Hay gặp: `.env` thiếu `API_KEY`, hoặc `CORS_ORIGINS=*` khi `APP_ENV=production`
— hai lỗi này server chủ động từ chối khởi động.

**Web hiện 502 Bad Gateway** — Nginx chạy nhưng app thì không:
```bash
sudo systemctl status s3-upload-tool
curl http://127.0.0.1:8080/api/v1/health
```

**Vẫn thấy giao diện cũ** — trình duyệt cache; thử Ctrl+Shift+R. Nếu vẫn vậy,
kiểm tra `web/dist` đã build lại chưa (`bash deploy/status.sh`).

**Upload ảnh lớn báo 413** — tăng `client_max_body_size` trong
`/etc/nginx/sites-available/pictures.vaschools.edu.vn.conf` cho khớp
`UPLOAD_ABSOLUTE_MAX_MB`, rồi `sudo nginx -t && sudo systemctl reload nginx`.

**Certbot thất bại** — DNS chưa trỏ đúng hoặc cổng 80 bị chặn:
```bash
getent hosts pictures.vaschools.edu.vn   # phải ra IP VPS
sudo ufw allow 80 && sudo ufw allow 443
```

---

## Rollback thủ công

```bash
cd /opt/s3-upload-tool
sudo git log --oneline -5                  # chọn commit tốt trước đó
sudo git checkout <commit-hash>
sudo bash deploy/deploy.sh --no-pull
```

---

## Đổi domain / đường dẫn

```bash
cp deploy/config.env.example deploy/config.env
nano deploy/config.env
```

Nhớ tạo file vhost tương ứng trong `deploy/nginx/<domain>.conf` (copy từ file
sẵn có rồi đổi `server_name`).

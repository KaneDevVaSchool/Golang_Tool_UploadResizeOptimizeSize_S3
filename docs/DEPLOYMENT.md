# Deployment Guide (VPS)

Hai cách deploy — chọn 1:

- **[Cách A: Docker](#cách-a-docker)** — dùng sẵn [Dockerfile](../Dockerfile) multi-stage, ít phụ thuộc môi trường VPS nhất.
- **[Cách B: Binary trực tiếp + systemd](#cách-b-binary-trực-tiếp--systemd)** — không cần Docker, build ra 1 binary Go tĩnh chạy thẳng.

Cả hai đều cần chung phần **cấu hình `.env`** và **reverse proxy/HTTPS** ở cuối bài.

---

## Chuẩn bị chung

- VPS Linux (Ubuntu/Debian khuyến nghị), có SSH access.
- Domain trỏ về IP VPS (nếu cần HTTPS/tên miền — không bắt buộc để chạy thử qua IP:port).
- S3 bucket + credentials (hoặc IAM role nếu VPS là EC2).
- (Tuỳ chọn) PostgreSQL nếu dùng `/api/v1/upload-transaction`.

---

## Cách A: Docker

### 1. Cài Docker
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # logout/login lại để áp dụng
```

### 2. Đưa code lên VPS
```bash
git clone <your-repo-url> s3-upload-tool
cd s3-upload-tool
```

### 3. Tạo `.env` production
Xem mục [Cấu hình .env production](#cấu-hình-env-production) bên dưới.

### 4. Build & chạy
```bash
docker build -t s3-upload-tool .

docker run -d \
  --name s3-upload-tool \
  --restart unless-stopped \
  -p 8080:8080 \
  --env-file .env \
  -v /opt/s3-upload-tool/uploads:/app/uploads \
  -v /opt/s3-upload-tool/wp-uploads:/app/wp-uploads \
  s3-upload-tool
```

Kiểm tra:
```bash
docker logs -f s3-upload-tool
curl http://localhost:8080/api/v1/health
```

`--restart unless-stopped` lo việc tự khởi động lại khi container crash hoặc VPS reboot.

### 5. Cập nhật code sau này
```bash
git pull
docker build -t s3-upload-tool .
docker stop s3-upload-tool && docker rm s3-upload-tool
# chạy lại lệnh docker run ở bước 4
```

---

## Cách B: Binary trực tiếp + systemd

### 1. Build binary

**B1 — build ngay trên VPS** (cần cài Go + Node trên VPS):
```bash
# Go 1.24+
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc

# Node.js 20+ (build web UI)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

cd /opt/s3-upload-tool   # sau khi git clone / rsync code lên
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server ./cmd/server
```

**B2 — cross-compile từ máy Windows local rồi upload** (không cần cài Go trên VPS):
```powershell
cd web
npm install
npm run build
cd ..

$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -trimpath -ldflags="-s -w" -o server_linux ./cmd/server
```
```bash
scp server_linux user@your-vps-ip:/opt/s3-upload-tool/server
scp -r web/dist user@your-vps-ip:/opt/s3-upload-tool/web/dist
scp .env user@your-vps-ip:/opt/s3-upload-tool/.env
```

### 2. Cấu hình `.env` production
Xem mục [Cấu hình .env production](#cấu-hình-env-production) bên dưới.

### 3. Chạy thử
```bash
chmod +x /opt/s3-upload-tool/server
cd /opt/s3-upload-tool && ./server
curl http://localhost:8080/api/v1/health
```

### 4. Chạy nền bằng systemd

Tạo user riêng (không chạy bằng root):
```bash
sudo useradd -r -s /sbin/nologin appuser
sudo chown -R appuser:appuser /opt/s3-upload-tool
```

Tạo `/etc/systemd/system/s3-upload-tool.service`:
```ini
[Unit]
Description=S3 Upload Tool
After=network.target

[Service]
Type=simple
User=appuser
WorkingDirectory=/opt/s3-upload-tool
EnvironmentFile=/opt/s3-upload-tool/.env
ExecStart=/opt/s3-upload-tool/server
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

Kích hoạt:
```bash
sudo systemctl daemon-reload
sudo systemctl enable s3-upload-tool
sudo systemctl start s3-upload-tool
sudo systemctl status s3-upload-tool

# xem log
sudo journalctl -u s3-upload-tool -f
```

### 5. Cập nhật code sau này
```bash
cd /opt/s3-upload-tool
git pull
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server ./cmd/server
sudo systemctl restart s3-upload-tool
```

---

## Cấu hình `.env` production

Copy `.env.example` → `.env` rồi chỉnh tối thiểu:

```env
APP_ENV=production

AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=your-bucket-name
# Ưu tiên IAM role nếu VPS là EC2; nếu không, cần access key thật (không để placeholder)
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

API_REQUIRE_KEY=true
API_KEY=<random dài, vd: openssl rand -hex 32>

CORS_ORIGINS=https://your-domain.com   # KHÔNG để "*" ở production

CSRF_SECURE_COOKIE=true   # bắt buộc true khi chạy sau HTTPS

DATABASE_ENABLED=false    # true nếu cần /api/v1/upload-transaction
# DATABASE_URL=postgres://user:pass@db-host:5432/dbname?sslmode=disable
```

⚠️ **Bắt buộc**: `APP_ENV=production` sẽ khiến server **từ chối khởi động** nếu thiếu `API_KEY` hoặc `CORS_ORIGINS` không hợp lệ (để `*`) — đây là validate cứng trong `internal/config`, không phải gợi ý.

Nếu bật database, chạy migration trước khi start server:
```bash
go run cmd/migrate/main.go -up
# hoặc build binary migrate riêng nếu không có Go trên VPS production
```

---

## Reverse proxy + HTTPS (áp dụng cho cả 2 cách)

Cài Nginx làm proxy trước ứng dụng (chạy ở `127.0.0.1:8080`):

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 200m;   # khớp UPLOAD_ABSOLUTE_MAX_MB
    }
}
```

Cấp SSL miễn phí bằng Certbot:
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

Sau khi có HTTPS, đảm bảo `CSRF_SECURE_COOKIE=true` trong `.env` rồi restart service.

---

## Checklist trước khi go-live

- [ ] `APP_ENV=production`, `API_KEY` đã set (không phải placeholder)
- [ ] `CORS_ORIGINS` trỏ đúng domain thật, không phải `*`
- [ ] AWS credentials thật (không phải `your_access_key_id` mẫu) — hoặc dùng IAM role
- [ ] `CSRF_SECURE_COOKIE=true` nếu đã có HTTPS
- [ ] `client_max_body_size` ở Nginx ≥ `UPLOAD_ABSOLUTE_MAX_MB`
- [ ] Volume/thư mục `uploads/`, `wp-uploads/` có đủ dung lượng trống và được backup nếu cần (đặc biệt WP resize lưu local, không lên S3)
- [ ] `GET /api/v1/health` trả `200 ok` sau khi deploy
- [ ] Nếu dùng nhiều instance/load balancer: chunked upload session lưu in-memory, cần sticky session (xem [ARCHITECTURE.md](./ARCHITECTURE.md#giới-hạn-đã-biết-cần-lưu-ý-khi-scale))

## Liên quan

- [ARCHITECTURE.md](./ARCHITECTURE.md) — kiến trúc, luồng request, giới hạn hệ thống
- [API.md](./API.md) — tham chiếu endpoint đầy đủ

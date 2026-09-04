# Deploy pictures.vaschools.edu.vn (Binary + systemd + Nginx)

Mục tiêu: domain `pictures.vaschools.edu.vn` hiện đang trỏ tạm và hiển thị giao diện
cũ vì **chưa có app + vhost thật chạy phía sau**. Làm theo thứ tự dưới đây để nó chạy
đúng bản UI mới trong repo này.

## 0. Điều kiện

- DNS: `pictures.vaschools.edu.vn` → A record trỏ đúng IP VPS (kiểm tra bằng
  `nslookup pictures.vaschools.edu.vn` hoặc `dig +short pictures.vaschools.edu.vn`
  từ máy local — phải ra đúng IP VPS, nếu chưa đúng thì HTTPS ở bước 4 sẽ fail).
- Nginx đã cài trên VPS (`sudo apt install -y nginx`).
- Đã build binary + web UI (xem [docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md) mục
  "Cách B: Binary trực tiếp + systemd") và có sẵn ở `/opt/s3-upload-tool/`:
  - `/opt/s3-upload-tool/server` (binary)
  - `/opt/s3-upload-tool/web/dist` (build UI mới)
  - `/opt/s3-upload-tool/.env` (copy từ `.env.example`, chỉnh production)

## Cách nhanh: dùng deploy.sh

Nếu đã thoả điều kiện ở mục 0 (DNS trỏ đúng, Nginx/Go/Node đã cài, và
`/opt/s3-upload-tool/.env` đã có), có thể chạy 1 lệnh thay cho toàn bộ bước 1-3
bên dưới:

```bash
cd /opt/s3-upload-tool
sudo bash deploy/deploy.sh
```

Script sẽ tự: `git pull` → build web UI → build binary → tạo `appuser` nếu
chưa có → cài/enable systemd service → cài/enable Nginx vhost → reload Nginx →
health-check qua `127.0.0.1:8080`. Chạy lại được nhiều lần (idempotent), dùng
luôn cho các lần cập nhật code sau này.

Bước 1-3 dưới đây là chi tiết thủ công tương đương, đọc khi cần debug hoặc muốn
hiểu script đang làm gì. Sau khi chạy xong (script hoặc thủ công), tiếp tục
mục 4 (Certbot/HTTPS).

## 1. Tạo user chạy app + cấp quyền

```bash
sudo useradd -r -s /sbin/nologin appuser
sudo chown -R appuser:appuser /opt/s3-upload-tool
sudo chmod +x /opt/s3-upload-tool/server
```

## 2. Cài systemd service

```bash
sudo cp deploy/systemd/s3-upload-tool.service /etc/systemd/system/s3-upload-tool.service
sudo systemctl daemon-reload
sudo systemctl enable s3-upload-tool
sudo systemctl start s3-upload-tool
sudo systemctl status s3-upload-tool   # phải là active (running)

# App phải trả lời trên localhost trước khi đụng tới Nginx:
curl http://127.0.0.1:8080/api/v1/health
```

Nếu bước này chưa OK (service crash / health không trả 200 ok), dừng lại xử lý
trước — đừng cấu hình Nginx trên một service chưa chạy được, sẽ chỉ thấy 502.
Xem log: `sudo journalctl -u s3-upload-tool -f`.

## 3. Cài vhost Nginx (HTTP trước)

```bash
sudo cp deploy/nginx/pictures.vaschools.edu.vn.conf /etc/nginx/sites-available/pictures.vaschools.edu.vn.conf
sudo ln -sf /etc/nginx/sites-available/pictures.vaschools.edu.vn.conf /etc/nginx/sites-enabled/

# Nếu VPS còn vhost/default nào khác đang chiếm server_name này hoặc default_server
# trên port 80 (chính là nguyên nhân "giao diện cũ" thường gặp), gỡ hoặc tắt nó:
#   ls /etc/nginx/sites-enabled/
#   sudo rm /etc/nginx/sites-enabled/<tên-vhost-cũ>

sudo nginx -t
sudo systemctl reload nginx
```

Kiểm tra: mở `http://pictures.vaschools.edu.vn` — phải thấy đúng UI mới (chưa
có ổ khoá HTTPS, đó là bình thường ở bước này).

Nếu vẫn thấy giao diện cũ ở bước này (không phải lỗi kết nối), gần như chắc chắn
là do:

- Trình duyệt cache trang cũ → thử tab ẩn danh / hard refresh (Ctrl+Shift+R).
- Có CDN/proxy trung gian (Cloudflare...) đang cache — cần purge cache ở đó.
- DNS chưa trỏ đúng VPS này (xem lại bước 0).

## 4. Bật HTTPS bằng Certbot

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d pictures.vaschools.edu.vn
```

Certbot sẽ tự thêm block `listen 443 ssl` + redirect HTTP→HTTPS vào file vhost.
Sau khi có HTTPS, cập nhật `.env` trên VPS rồi restart app:

```bash
# trong /opt/s3-upload-tool/.env
CSRF_SECURE_COOKIE=true

sudo systemctl restart s3-upload-tool
```

Certbot tự cài cron/timer gia hạn cert — kiểm tra bằng:

```bash
sudo certbot renew --dry-run
```

## 5. Checklist cuối cùng

- [ ] `curl https://pictures.vaschools.edu.vn/api/v1/health` → `200 ok`
- [ ] `.env` trên VPS: `APP_ENV=production`, `API_KEY` đã set thật,
      `CORS_ORIGINS=https://pictures.vaschools.edu.vn` (không phải `*`),
      `CSRF_SECURE_COOKIE=true`
- [ ] Không còn vhost/site nào khác trả lời cho `pictures.vaschools.edu.vn`
- [ ] Test upload thử 1 ảnh qua UI thật trên domain

## Cập nhật code sau này

```bash
cd /opt/s3-upload-tool
git pull
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server ./cmd/server
sudo systemctl restart s3-upload-tool
```

Không cần đụng lại Nginx khi chỉ update code app — vhost chỉ cần sửa nếu đổi port,
domain, hoặc giới hạn upload.

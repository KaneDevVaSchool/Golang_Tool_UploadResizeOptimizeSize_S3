#!/usr/bin/env bash
# Deploy script — chạy TRÊN VPS (không chạy trên máy Windows local).
#
# Việc script làm:
#   1. git pull code mới nhất (repo tại APP_DIR)
#   2. build web UI (npm ci && npm run build)
#   3. build binary Go tĩnh
#   4. lần đầu chạy: tạo user appuser, cài systemd service + Nginx vhost, enable
#      các lần sau: chỉ copy binary/web mới rồi restart service
#   5. reload Nginx, health-check qua localhost
#
# Cách dùng:
#   cd /opt/s3-upload-tool
#   sudo bash deploy/deploy.sh
#
# Yêu cầu trước khi chạy lần đầu:
#   - Đã có /opt/s3-upload-tool/.env (copy từ .env.example, chỉnh production)
#   - Đã cài Nginx, Go 1.24+, Node 20+ trên VPS (xem docs/DEPLOYMENT.md)
#   - DNS pictures.vaschools.edu.vn đã trỏ về IP VPS này

set -euo pipefail

APP_DIR="/opt/s3-upload-tool"
APP_USER="appuser"
DOMAIN="pictures.vaschools.edu.vn"
SERVICE_NAME="s3-upload-tool"

if [[ $EUID -ne 0 ]]; then
  echo "Cần chạy bằng sudo/root (để quản lý systemd + Nginx + quyền file)." >&2
  exit 1
fi

cd "$APP_DIR"

echo "==> [1/6] git pull"
git pull

echo "==> [2/6] build web UI"
(cd web && npm ci && npm run build)

echo "==> [3/6] build binary Go"
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server.new ./cmd/server
mv server.new server
chmod +x server

if [[ ! -f .env ]]; then
  echo "❌ Thiếu $APP_DIR/.env — copy từ .env.example và chỉnh production trước khi deploy." >&2
  exit 1
fi

echo "==> [4/6] user + quyền"
if ! id "$APP_USER" &>/dev/null; then
  useradd -r -s /sbin/nologin "$APP_USER"
fi
chown -R "$APP_USER:$APP_USER" "$APP_DIR"

echo "==> [5/6] systemd service"
cp "$APP_DIR/deploy/systemd/${SERVICE_NAME}.service" "/etc/systemd/system/${SERVICE_NAME}.service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null
systemctl restart "$SERVICE_NAME"

echo "==> [6/6] Nginx vhost"
NGINX_CONF="/etc/nginx/sites-available/${DOMAIN}.conf"
cp "$APP_DIR/deploy/nginx/${DOMAIN}.conf" "$NGINX_CONF"
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/${DOMAIN}.conf"
nginx -t
systemctl reload nginx

echo "==> Health check (chờ app khởi động)"
sleep 2
if curl -fsS "http://127.0.0.1:8080/api/v1/health"; then
  echo -e "\n✅ Deploy xong. Kiểm tra: http://${DOMAIN} (và bật HTTPS bằng certbot nếu chưa có — xem deploy/README.md)"
else
  echo -e "\n❌ Health check thất bại. Xem log: journalctl -u ${SERVICE_NAME} -n 100 --no-pager" >&2
  exit 1
fi

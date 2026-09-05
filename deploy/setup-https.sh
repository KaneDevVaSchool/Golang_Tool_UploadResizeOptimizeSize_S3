#!/usr/bin/env bash
# Bật HTTPS bằng Let's Encrypt (Certbot) — chạy MỘT LẦN sau khi deploy.sh
# đã chạy được và site mở qua http://<domain>.
#
#   sudo bash deploy/setup-https.sh
#
# Certbot sẽ tự sửa file vhost để thêm block 443 + redirect 80→443, và tự
# cài cron/timer gia hạn chứng chỉ. Sau đó nhớ bật CSRF_SECURE_COOKIE=true.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="/opt/s3-upload-tool"
DOMAIN="pictures.vaschools.edu.vn"
SERVICE_NAME="s3-upload-tool"
# config.env là tuỳ chọn — dùng if để set -e không giết script khi vắng mặt.
# shellcheck source=/dev/null
if [[ -f "$SCRIPT_DIR/config.env" ]]; then source "$SCRIPT_DIR/config.env"; fi

log()  { echo -e "\033[1;36m==>\033[0m $*"; }
fail() { echo -e "\033[1;31m✗\033[0m $*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || fail "Cần chạy bằng sudo."

command -v certbot >/dev/null || {
  log "Cài certbot"
  apt update && apt install -y certbot python3-certbot-nginx
}

# Certbot xác thực qua HTTP nên DNS phải trỏ đúng trước, nếu không sẽ bị
# rate-limit của Let's Encrypt khi thử lại nhiều lần.
DNS_IP="$(getent hosts "$DOMAIN" | awk '{print $1}' | head -1 || true)"
SERVER_IP="$(curl -fsS --max-time 5 https://api.ipify.org || echo '')"
[[ -n "$DNS_IP" ]] || fail "$DOMAIN chưa phân giải. Tạo A record trỏ về ${SERVER_IP:-IP VPS} rồi chờ TTL."
if [[ -n "$SERVER_IP" && "$DNS_IP" != "$SERVER_IP" ]]; then
  echo -e "\033[1;33m! DNS trỏ $DNS_IP nhưng máy này là $SERVER_IP.\033[0m"
  read -rp "  Vẫn tiếp tục? [y/N] " a; [[ "$a" == "y" || "$a" == "Y" ]] || exit 1
fi

# Thư mục webroot cho ACME challenge (khớp vhost trong deploy/nginx/).
mkdir -p /var/www/certbot
chown -R www-data:www-data /var/www/certbot

log "Xin chứng chỉ cho $DOMAIN"
certbot --nginx -d "$DOMAIN" --agree-tos --redirect --non-interactive \
  --email "admin@${DOMAIN#*.}" || fail "Certbot thất bại. Kiểm tra DNS + cổng 80 mở."

nginx -t && systemctl reload nginx

# Cookie CSRF phải có cờ Secure khi site đã chạy HTTPS.
if [[ -f "$APP_DIR/.env" ]] && ! grep -qE '^CSRF_SECURE_COOKIE=true' "$APP_DIR/.env"; then
  log "Bật CSRF_SECURE_COOKIE=true trong .env"
  if grep -qE '^CSRF_SECURE_COOKIE=' "$APP_DIR/.env"; then
    sed -i 's/^CSRF_SECURE_COOKIE=.*/CSRF_SECURE_COOKIE=true/' "$APP_DIR/.env"
  else
    echo 'CSRF_SECURE_COOKIE=true' >> "$APP_DIR/.env"
  fi
  systemctl restart "$SERVICE_NAME"
fi

systemctl list-timers 2>/dev/null | grep -q certbot \
  && log "Tự động gia hạn: đã bật (systemd timer)" \
  || log "Kiểm tra gia hạn tự động: systemctl status certbot.timer"

echo -e "\033[1;32m✓ HTTPS đã bật → https://${DOMAIN}\033[0m"
echo "  Kiểm tra gia hạn thử: sudo certbot renew --dry-run"

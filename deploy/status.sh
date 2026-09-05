#!/usr/bin/env bash
# Xem nhanh tình trạng hệ thống khi có sự cố.
#
#   sudo bash deploy/status.sh
#
# Chỉ đọc, không thay đổi gì.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="/opt/s3-upload-tool"
DOMAIN="pictures.vaschools.edu.vn"
SERVICE_NAME="s3-upload-tool"
APP_PORT="8080"
# shellcheck source=/dev/null
[[ -f "$SCRIPT_DIR/config.env" ]] && source "$SCRIPT_DIR/config.env"

hdr() { echo -e "\n\033[1;36m── $* ─────────────────────────────\033[0m"; }

hdr "Service $SERVICE_NAME"
systemctl is-active "$SERVICE_NAME" &>/dev/null \
  && echo -e "\033[1;32m● đang chạy\033[0m" || echo -e "\033[1;31m● KHÔNG chạy\033[0m"
systemctl status "$SERVICE_NAME" --no-pager -n 0 2>/dev/null | sed -n '2,5p'

hdr "Health check nội bộ"
if curl -fsS --max-time 5 "http://127.0.0.1:${APP_PORT}/api/v1/health" 2>/dev/null; then
  echo -e "\n\033[1;32m✓ App phản hồi\033[0m"
else
  echo -e "\033[1;31m✗ App không phản hồi ở cổng $APP_PORT\033[0m"
fi

hdr "Truy cập từ ngoài"
code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 8 "https://${DOMAIN}" 2>/dev/null || echo '---')"
echo "https://${DOMAIN} → HTTP $code"

hdr "Chứng chỉ HTTPS"
if command -v certbot >/dev/null; then
  certbot certificates 2>/dev/null | grep -E 'Certificate Name|Expiry Date' | sed 's/^ *//'
else
  echo "certbot chưa cài — site chưa có HTTPS?"
fi

hdr "Nginx"
nginx -t 2>&1 | sed 's/^/  /'

hdr "Phiên bản đang chạy"
if [[ -d "$APP_DIR/.git" ]]; then
  git -C "$APP_DIR" log -1 --format='  %h  %s  (%cr)' 2>/dev/null
fi
[[ -f "$APP_DIR/server" ]] && echo "  binary: $(date -r "$APP_DIR/server" '+%Y-%m-%d %H:%M')"
[[ -f "$APP_DIR/web/dist/index.html" ]] \
  && echo "  web/dist: $(date -r "$APP_DIR/web/dist/index.html" '+%Y-%m-%d %H:%M')" \
  || echo -e "  \033[1;31mweb/dist chưa build\033[0m"

hdr "Tài nguyên"
df -Ph "$APP_DIR" 2>/dev/null | awk 'NR==2{print "  Đĩa: dùng "$5" ("$4" trống)"}'
free -h 2>/dev/null | awk 'NR==2{print "  RAM: dùng "$3"/"$2}'

hdr "10 dòng log gần nhất"
journalctl -u "$SERVICE_NAME" -n 10 --no-pager 2>/dev/null | sed 's/^/  /'

echo -e "\n\033[2mLog realtime: journalctl -u ${SERVICE_NAME} -f\033[0m"

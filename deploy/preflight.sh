#!/usr/bin/env bash
# Kiểm tra VPS đã sẵn sàng deploy chưa — chạy TRƯỚC lần deploy đầu tiên.
# Chỉ đọc và báo cáo, KHÔNG thay đổi gì trên máy.
#
#   sudo bash deploy/preflight.sh
#
# Mỗi mục FAIL đều kèm lệnh khắc phục ngay bên dưới.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="/opt/s3-upload-tool"
DOMAIN="pictures.vaschools.edu.vn"
APP_PORT="8080"
SERVICE_NAME="s3-upload-tool"
# shellcheck source=/dev/null
[[ -f "$SCRIPT_DIR/config.env" ]] && source "$SCRIPT_DIR/config.env"

PASS=0; FAIL=0; WARN=0
ok()   { echo -e "  \033[1;32m✓\033[0m $1"; PASS=$((PASS+1)); }
bad()  { echo -e "  \033[1;31m✗\033[0m $1"; [[ -n "${2:-}" ]] && echo -e "      → fix: $2"; FAIL=$((FAIL+1)); }
warn() { echo -e "  \033[1;33m!\033[0m $1"; [[ -n "${2:-}" ]] && echo -e "      → $2"; WARN=$((WARN+1)); }

echo
echo "Preflight check — $DOMAIN"
echo "================================================"

echo
echo "[1] Công cụ build & chạy"
check_ver() { # tên, lệnh-lấy-version, version tối thiểu, lệnh cài
  local name="$1" cur="$2" want="$3" fix="$4"
  if [[ -z "$cur" ]]; then bad "$name chưa cài" "$fix"; return; fi
  if [[ "$(printf '%s\n%s\n' "$want" "$cur" | sort -V | head -1)" == "$want" ]]; then
    ok "$name $cur (cần >= $want)"
  else
    bad "$name $cur quá cũ (cần >= $want)" "$fix"
  fi
}
check_ver "Go" "$(go version 2>/dev/null | grep -oP 'go\K[0-9.]+' || echo '')" "1.24" \
  "wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz && sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz && echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc"
check_ver "Node" "$(node -v 2>/dev/null | tr -d 'v' || echo '')" "20.0.0" \
  "curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && sudo apt install -y nodejs"
command -v npm   >/dev/null && ok "npm $(npm -v)"       || bad "npm chưa cài" "cài kèm Node ở trên"
command -v git   >/dev/null && ok "git đã cài"          || bad "git chưa cài" "sudo apt install -y git"
command -v nginx >/dev/null && ok "nginx đã cài"        || bad "nginx chưa cài" "sudo apt install -y nginx"
command -v curl  >/dev/null && ok "curl đã cài"         || bad "curl chưa cài" "sudo apt install -y curl"
command -v certbot >/dev/null && ok "certbot đã cài"    || warn "certbot chưa cài (cần cho HTTPS)" "sudo apt install -y certbot python3-certbot-nginx"

echo
echo "[2] Mã nguồn & cấu hình"
if [[ -d "$APP_DIR" ]]; then
  ok "Thư mục app: $APP_DIR"
  [[ -d "$APP_DIR/.git" ]] && ok "Là git repo (deploy.sh pull được)" \
    || warn "Không phải git repo" "deploy.sh cần chạy với --no-pull"
  if [[ -f "$APP_DIR/.env" ]]; then
    ok ".env tồn tại"
    perm="$(stat -c '%a' "$APP_DIR/.env")"
    [[ "$perm" == "600" ]] && ok ".env quyền 600 (chỉ owner đọc)" \
      || warn ".env đang quyền $perm — chứa secret" "deploy.sh sẽ tự chmod 600"
    # Các giá trị bắt buộc ở production: server sẽ từ chối khởi động nếu sai.
    grep -qE '^APP_ENV=production'      "$APP_DIR/.env" && ok "APP_ENV=production" \
      || bad "APP_ENV chưa phải production" "sửa APP_ENV=production trong $APP_DIR/.env"
    grep -qE '^CORS_ORIGINS=\*'         "$APP_DIR/.env" && bad "CORS_ORIGINS đang là '*' — server sẽ không khởi động" \
      "đổi thành CORS_ORIGINS=https://$DOMAIN" || ok "CORS_ORIGINS không phải '*'"
    grep -qE '^CSRF_SECURE_COOKIE=true' "$APP_DIR/.env" && ok "CSRF_SECURE_COOKIE=true" \
      || warn "CSRF_SECURE_COOKIE chưa true" "bật true khi site chạy HTTPS"
    grep -qE '^S3_BUCKET_NAME=.+'       "$APP_DIR/.env" && ok "S3_BUCKET_NAME đã điền" \
      || bad "Thiếu S3_BUCKET_NAME" "điền tên bucket vào $APP_DIR/.env"
  else
    bad "Thiếu $APP_DIR/.env" "cp $APP_DIR/.env.example $APP_DIR/.env && nano $APP_DIR/.env"
  fi
else
  bad "Chưa có $APP_DIR" "sudo git clone <repo-url> $APP_DIR"
fi

echo
echo "[3] Mạng & DNS"
SERVER_IP="$(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || echo '')"
DNS_IP="$(getent hosts "$DOMAIN" 2>/dev/null | awk '{print $1}' | head -1)"
if [[ -z "$DNS_IP" ]]; then
  bad "$DOMAIN chưa phân giải được" "tạo A record trỏ $DOMAIN → ${SERVER_IP:-<IP VPS>}"
elif [[ -n "$SERVER_IP" && "$DNS_IP" == "$SERVER_IP" ]]; then
  ok "DNS $DOMAIN → $DNS_IP (khớp IP máy này)"
else
  warn "DNS $DOMAIN → $DNS_IP, IP máy này là ${SERVER_IP:-không xác định}" \
    "nếu khác nhau thì Certbot sẽ fail; sửa A record rồi chờ TTL"
fi

echo
echo "[4] Cổng & service"
if ss -ltn 2>/dev/null | grep -q ":${APP_PORT}\b"; then
  if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    ok "Cổng $APP_PORT đang do $SERVICE_NAME dùng (deploy lại sẽ restart)"
  else
    bad "Cổng $APP_PORT bị tiến trình khác chiếm" "sudo ss -ltnp | grep :$APP_PORT  # xem là ai rồi tắt"
  fi
else
  ok "Cổng $APP_PORT còn trống"
fi
ss -ltn 2>/dev/null | grep -qE ':(80|443)\b' && ok "Nginx đang nghe 80/443" \
  || warn "Chưa có gì nghe cổng 80/443" "sudo systemctl start nginx"

echo
echo "[5] Dung lượng đĩa"
AVAIL_KB="$(df -Pk "${APP_DIR%/*}" 2>/dev/null | awk 'NR==2{print $4}')"
if [[ -n "$AVAIL_KB" ]]; then
  AVAIL_MB=$((AVAIL_KB / 1024))
  # Go build cache + node_modules + binary ≈ cần vài trăm MB.
  [[ $AVAIL_MB -gt 2048 ]] && ok "Còn trống ${AVAIL_MB}MB" \
    || warn "Chỉ còn ${AVAIL_MB}MB (build cần ~2GB)" "dọn bớt: sudo apt clean; go clean -cache"
fi

echo
echo "================================================"
echo -e "Kết quả: \033[1;32m$PASS đạt\033[0m, \033[1;33m$WARN cảnh báo\033[0m, \033[1;31m$FAIL lỗi\033[0m"
if [[ $FAIL -gt 0 ]]; then
  echo -e "\033[1;31mCòn $FAIL mục phải sửa trước khi deploy.\033[0m"
  exit 1
fi
echo -e "\033[1;32mSẵn sàng deploy:\033[0m sudo bash deploy/deploy.sh"

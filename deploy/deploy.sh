#!/usr/bin/env bash
# Deploy / cập nhật ứng dụng — chạy TRÊN VPS Linux (không chạy trên Windows local).
#
#   sudo bash deploy/deploy.sh              # deploy đầy đủ
#   sudo bash deploy/deploy.sh --no-pull    # deploy code đang có sẵn, bỏ qua git pull
#   sudo bash deploy/deploy.sh --skip-web   # chỉ build lại binary Go (sửa backend)
#
# Script idempotent: chạy lần đầu để cài đặt, các lần sau để cập nhật.
# Nếu app không khoẻ sau khi restart, script tự rollback về binary trước đó.
#
# Lần đầu chạy cần: preflight.sh đã pass (xem deploy/README.md).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# --- Cấu hình (ghi đè trong deploy/config.env) ---------------------------------
APP_DIR="/opt/s3-upload-tool"
APP_USER="appuser"
DOMAIN="pictures.vaschools.edu.vn"
SERVICE_NAME="s3-upload-tool"
APP_PORT="8080"
GIT_BRANCH="main"
# config.env là tuỳ chọn — dùng if để set -e không giết script khi vắng mặt.
# shellcheck source=/dev/null
if [[ -f "$SCRIPT_DIR/config.env" ]]; then source "$SCRIPT_DIR/config.env"; fi

DO_PULL=1
DO_WEB=1
for arg in "$@"; do
  case "$arg" in
    --no-pull)  DO_PULL=0 ;;
    --skip-web) DO_WEB=0 ;;
    -h|--help)  sed -n '2,10p' "$0"; exit 0 ;;
    *) echo "Tham số lạ: $arg (xem --help)" >&2; exit 2 ;;
  esac
done

# Dọn file tạm kể cả khi script thoát giữa chừng (build lỗi, migration fail).
trap 'rm -f "$APP_DIR/server.new" "$APP_DIR/migrate.tmp" 2>/dev/null || true' EXIT

log()  { echo -e "\033[1;36m==>\033[0m $*"; }
ok()   { echo -e "\033[1;32m✓\033[0m $*"; }
fail() { echo -e "\033[1;31m✗\033[0m $*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || fail "Cần chạy bằng sudo (quản lý systemd + Nginx + quyền file)."
[[ -d "$APP_DIR" ]] || fail "Không thấy $APP_DIR. Clone code vào đó trước (xem deploy/README.md)."
cd "$APP_DIR"

# .env kiểm tra TRƯỚC khi build để không tốn công build rồi mới báo thiếu.
[[ -f .env ]] || fail "Thiếu $APP_DIR/.env — copy từ .env.example và chỉnh production trước."

for bin in go git nginx curl; do
  command -v "$bin" >/dev/null || fail "Thiếu '$bin'. Chạy 'sudo bash deploy/preflight.sh' để xem còn thiếu gì."
done
if [[ $DO_WEB -eq 1 ]]; then
  command -v npm >/dev/null || fail "Thiếu 'npm' (cần build web UI). Dùng --skip-web nếu chỉ sửa backend."
fi

if [[ $DO_PULL -eq 1 ]]; then
  log "[1/7] Cập nhật source (git pull origin $GIT_BRANCH)"
  git pull --ff-only origin "$GIT_BRANCH"
else
  log "[1/7] Bỏ qua git pull (--no-pull)"
fi

if [[ $DO_WEB -eq 1 ]]; then
  log "[2/7] Build giao diện web"
  ( cd web && npm ci && npm run build )
  [[ -f web/dist/index.html ]] || fail "Build web xong nhưng không thấy web/dist/index.html."
else
  log "[2/7] Bỏ qua build web (--skip-web)"
fi

log "[3/7] Build binary Go"
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server.new ./cmd/server
# Giữ bản đang chạy để rollback nếu bản mới không khoẻ. Dùng if thay vì
# `[[ ... ]] && cp` vì với set -e, biểu thức false ở cuối sẽ giết script
# ngay lần deploy đầu (khi chưa có file server nào).
if [[ -f server ]]; then
  cp -p server server.prev
fi
mv server.new server
chmod +x server

log "[4/7] Chạy migration database (nếu bật)"
if grep -qE '^DATABASE_ENABLED=true' .env; then
  # cmd/migrate tự đọc .env theo thư mục hiện hành, mà ta đang ở $APP_DIR.
  CGO_ENABLED=0 GOOS=linux go build -trimpath -o migrate.tmp ./cmd/migrate
  ./migrate.tmp -up
  rm -f migrate.tmp
  ok "Migration xong"
else
  echo "    DATABASE_ENABLED khác true — bỏ qua."
fi

log "[5/7] User hệ thống + quyền thư mục"
id "$APP_USER" &>/dev/null || useradd -r -s /usr/sbin/nologin "$APP_USER"
# Các thư mục runtime không nằm trong git (bị .gitignore) nhưng service cần
# có sẵn: systemd ReadWritePaths sẽ fail nếu đường dẫn chưa tồn tại.
mkdir -p "$APP_DIR"/{uploads,wp-uploads,storage/logs}
chown -R "$APP_USER:$APP_USER" "$APP_DIR"
# .env chứa secret (AWS key, DB password, session secret) — chỉ owner đọc được.
chmod 600 "$APP_DIR/.env"

log "[6/7] systemd service"
install -m 644 "$SCRIPT_DIR/systemd/${SERVICE_NAME}.service" "/etc/systemd/system/${SERVICE_NAME}.service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null
systemctl restart "$SERVICE_NAME"

log "[7/7] Nginx vhost"
NGINX_CONF="/etc/nginx/sites-available/${DOMAIN}.conf"
# Certbot sửa trực tiếp file vhost để thêm block SSL. Ghi đè sẽ mất cấu hình
# HTTPS, nên chỉ cài lần đầu; sau đó muốn cập nhật thì sửa tay rồi reload.
if [[ -f "$NGINX_CONF" ]]; then
  echo "    Đã có $NGINX_CONF — giữ nguyên (tránh ghi đè cấu hình SSL của Certbot)."
else
  install -m 644 "$SCRIPT_DIR/nginx/${DOMAIN}.conf" "$NGINX_CONF"
  ln -sfn "$NGINX_CONF" "/etc/nginx/sites-enabled/${DOMAIN}.conf"
fi
nginx -t
systemctl reload nginx

# --- Health check + rollback --------------------------------------------------
log "Health check http://127.0.0.1:${APP_PORT}/api/v1/health"
HEALTHY=0
for i in {1..15}; do
  if curl -fsS --max-time 3 "http://127.0.0.1:${APP_PORT}/api/v1/health" >/dev/null 2>&1; then
    HEALTHY=1; break
  fi
  sleep 1
done

if [[ $HEALTHY -eq 1 ]]; then
  ok "Deploy thành công → https://${DOMAIN}"
  rm -f server.prev
  echo "   Log: journalctl -u ${SERVICE_NAME} -f"
  exit 0
fi

echo -e "\033[1;31m✗ Health check thất bại sau 15s.\033[0m" >&2
if [[ -f server.prev ]]; then
  log "Rollback về binary trước đó"
  mv server.prev server
  chown "$APP_USER:$APP_USER" server
  systemctl restart "$SERVICE_NAME"
  sleep 3
  if curl -fsS --max-time 3 "http://127.0.0.1:${APP_PORT}/api/v1/health" >/dev/null 2>&1; then
    echo -e "\033[1;33m⚠ Đã rollback — site chạy lại bản CŨ. Sửa lỗi rồi deploy lại.\033[0m" >&2
  else
    echo -e "\033[1;31m✗ Rollback cũng không khoẻ — cần xử lý thủ công.\033[0m" >&2
  fi
fi
echo "Xem log lỗi: journalctl -u ${SERVICE_NAME} -n 100 --no-pager" >&2
exit 1

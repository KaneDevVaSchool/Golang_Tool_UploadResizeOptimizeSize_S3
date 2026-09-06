#!/usr/bin/env bash
# PostToolUse hook (Write|Edit) — nhắc cập nhật tài liệu khi code vừa đổi
# theo cách chắc chắn làm tài liệu lệch (quy tắc mục 0 trong CLAUDE.md).
#
# Chỉ CẢNH BÁO, không chặn (luôn exit 0). In ra stderr để Claude thấy trong
# tool result và tự xử lý ngay trong cùng phiên làm việc, thay vì để nợ lại.
#
# Cố ý im lặng với phần lớn thay đổi: hook kêu quá nhiều thì người ta học
# cách phớt lờ nó. Chỉ nói khi có căn cứ cụ thể.
set -u

payload="$(cat)"

# Lấy file_path từ JSON stdin, không phụ thuộc jq (VPS/máy dev có thể chưa cài).
file_path="$(printf '%s' "$payload" \
  | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' \
  | head -1 | sed -E 's/.*:[[:space:]]*"//; s/"$//')"

[ -z "$file_path" ] && exit 0

# Chuẩn hoá để so khớp trên cả Windows lẫn Linux:
#  - đổi \ thành / (đường dẫn Windows),
#  - cắt bỏ tiền tố thư mục dự án, giữ lại đường dẫn tương đối từ gốc repo.
# Cắt tiền tố là cần thiết vì tool có thể đưa vào đường dẫn tuyệt đối HOẶC
# tương đối; các pattern bên dưới nhờ vậy chỉ cần viết một dạng.
#
# Phải đổi \ thành / TRƯỚC khi kiểm tra file tồn tại: bash trên Git Bash
# không stat được đường dẫn kiểu "C:\Users\..." nên [ -f ] luôn sai, khiến
# hook im lặng với đúng những lần Claude ghi bằng đường dẫn tuyệt đối.
norm="$(printf '%s' "$file_path" | tr '\\' '/')"

[ -f "$norm" ] || exit 0

case "$norm" in
  */internal/*|*/web/*|*/deploy/*|*/cmd/*|*/docs/*)
    norm="${norm#*/}"
    while :; do
      case "$norm" in
        internal/*|web/*|deploy/*|cmd/*|docs/*) break ;;
        */*) norm="${norm#*/}" ;;
        *) break ;;
      esac
    done
    ;;
esac

remind() {
  echo "📋 [docs-audit] $1" >&2
  shift
  for doc in "$@"; do
    echo "    → cập nhật: $doc" >&2
  done
}

case "$norm" in

  internal/database/migrations/*.sql)
    remind "Migration mới/đổi — schema đã lệch khỏi tài liệu." \
      "docs/detail_design/01-database.md (bảng, cột, index, VÀ lý do thiết kế)"
    ;;

  internal/config/builder.go|internal/config/config.go)
    remind "Cấu hình đổi — biến môi trường mới phải có đủ ở BA chỗ." \
      ".env.example (kèm chú thích)" \
      "docs/deploys/02-configuration.md"
    ;;

  internal/handlers/*_test.go)
    # Sửa test không làm tài liệu lệch — im lặng.
    exit 0
    ;;

  internal/handlers/*.go)
    remind "Handler đổi — endpoint có thể đã thêm/đổi/xoá." \
      "docs/API.md (request, response, mã lỗi)"
    ;;

  internal/container/container.go)
    remind "Container đổi — route hoặc chuỗi middleware có thể đã khác." \
      "docs/ARCHITECTURE.md (chuỗi middleware, luồng request)" \
      "docs/API.md (nếu có route mới)"
    ;;

  deploy/*.sh|deploy/systemd/*|deploy/nginx/*)
    remind "Script/cấu hình deploy đổi — hướng dẫn có thể sai bước." \
      "docs/deploys/00-tu-dau-den-cuoi.md" \
      "docs/deploys/03-operations.md" \
      "deploy/README.md"
    ;;

  web/src/pages/*|web/src/App.tsx)
    remind "Trang/route frontend đổi." \
      "docs/detail_design/06-frontend.md (danh sách trang, sơ đồ router)"
    ;;

  *) exit 0 ;;
esac

echo "    (Chạy skill 'docs-audit' để rà đầy đủ, kể cả docs/plan/. Cập nhật tài liệu TRONG CÙNG commit với code.)" >&2

exit 0

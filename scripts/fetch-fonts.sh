#!/usr/bin/env bash
# Tải font Google về self-host + sinh @font-face trỏ vào /fonts/.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../web" && pwd)"
OUT_DIR="$ROOT/public/fonts"
CSS_OUT="$ROOT/src/styles/fonts.css"
UA="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
URL="https://fonts.googleapis.com/css2?family=Be+Vietnam+Pro:ital,wght@0,300;0,400;0,500;0,600;0,700;0,800;1,500;1,600&family=Fraunces:opsz,wght@9..144,500;9..144,700;9..144,800&display=swap"

mkdir -p "$OUT_DIR"
RAW="$(mktemp)"
curl -s -A "$UA" "$URL" -o "$RAW"

python - "$RAW" "$OUT_DIR" "$CSS_OUT" <<'PY'
import re, sys, urllib.request, pathlib, hashlib
sys.stdout.reconfigure(encoding="utf-8", errors="replace")

raw_path, out_dir, css_out = sys.argv[1], pathlib.Path(sys.argv[2]), pathlib.Path(sys.argv[3])
css = pathlib.Path(raw_path).read_text(encoding="utf-8")

# Mỗi khối gồm: comment tên subset (nếu có) + @font-face {...}
blocks = re.findall(r"(?:/\*\s*([\w-]+)\s*\*/\s*)?(@font-face\s*\{[^}]*\})", css)

seen = {}
out_blocks = []
for subset, block in blocks:
    fam = re.search(r"font-family:\s*'([^']+)'", block).group(1)
    style = re.search(r"font-style:\s*(\w+)", block).group(1)
    weight_m = re.search(r"font-weight:\s*([\d\s]+);", block)
    weight = weight_m.group(1).strip()
    url = re.search(r"url\(([^)]+)\)", block).group(1)
    urange = re.search(r"unicode-range:\s*([^;]+);", block)

    slug_fam = fam.lower().replace(" ", "-")
    slug_w = weight.replace(" ", "-")
    name = f"{slug_fam}-{style}-{slug_w}-{subset or 'all'}.woff2"
    dest = out_dir / name

    if not dest.exists():
        data = urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})).read()
        dest.write_bytes(data)
        print(f"tải {name} ({len(data)//1024} KB)")
    seen[name] = dest.stat().st_size

    lines = [
        "@font-face {",
        f"  font-family: '{fam}';",
        f"  font-style: {style};",
        f"  font-weight: {weight};",
        "  font-display: swap;",
        f"  src: url('/fonts/{name}') format('woff2');",
    ]
    if urange:
        lines.append(f"  unicode-range: {urange.group(1).strip()};")
    lines.append("}")
    out_blocks.append((subset or "all", "\n".join(lines)))

header = """/*
  Font self-host. KHÔNG nạp từ fonts.googleapis.com.

  Vì sao: CSP của app đặt style-src 'self' 'unsafe-inline' và font-src 'self'
  data: (xem internal/middleware/security_headers.go). Nạp stylesheet từ
  Google bị chặn thẳng, và cách sửa kia - thêm fonts.googleapis.com vào CSP -
  đổi lấy việc mỗi lượt xem trang đều gọi sang máy chủ Google, tức là địa chỉ
  IP của từng học sinh và phụ huynh xem tranh đều đi ra ngoài.

  File sinh bằng script từ CSS gốc của Google, giữ nguyên unicode-range nên
  trình duyệt vẫn chỉ tải subset nó cần (phần lớn người dùng chỉ chạm vào
  subset vietnamese + latin).

  Muốn đổi bộ weight: sửa danh sách trong scripts/fetch-fonts.sh rồi chạy lại,
  đừng sửa tay file này.
*/

"""

body = "\n\n".join(f"/* {s} */\n{b}" for s, b in out_blocks)
# newline="\n": chạy script trên Windows mà để Python tự chọn thì file ra CRLF,
# nên mỗi lần sinh lại trên máy khác hệ điều hành là cả file hiện thành đã đổi
# trong git diff dù nội dung y hệt.
css_out.write_text(header + body + "\n", encoding="utf-8", newline="\n")
total = sum(seen.values())
print(f"\n{len(seen)} file, tổng {total//1024} KB -> {css_out}")
PY

---
name: docs-audit
description: Đối chiếu thay đổi code vừa thực hiện với bộ tài liệu trong docs/ và kế hoạch trong docs/plan/, phát hiện tài liệu đã lệch khỏi code — endpoint mới chưa ghi vào API.md, biến môi trường thiếu trong .env.example hoặc 02-configuration.md, migration mới chưa cập nhật 01-database.md, mục roadmap đã xong nhưng chưa đánh dấu, con số trong tài liệu không còn đúng. Dùng khi người dùng yêu cầu "audit docs", "kiểm tra tài liệu", "đối chiếu docs với code", "review plan", hoặc TRƯỚC KHI hoàn thành bất kỳ task nào có sửa code Go, frontend, migration, hay biến môi trường.
---

# Docs Audit

Rà soát xem bộ tài liệu còn khớp với code không, theo quy tắc mục 0 trong `CLAUDE.md`.

Chạy skill này **sau khi code xong, trước khi commit**. Mục tiêu là cập nhật tài liệu
trong **cùng commit** với code, không để nợ lại.

## Bước 1 — Xác định phạm vi thay đổi

```bash
git status --short
git diff --stat HEAD
git diff --name-only HEAD
```

Nếu đã commit rồi thì so với điểm phân nhánh:

```bash
git diff --name-only main...HEAD
```

Phân loại file đã đổi theo bảng ở bước 2. File nào không rơi vào nhóm nào thì bỏ qua.

## Bước 2 — Đối chiếu theo loại thay đổi

### 2.1 Endpoint

Nếu có sửa `internal/handlers/` hoặc phần đăng ký route trong `internal/container/container.go`:

```bash
# Route thực tế đăng ký trong code
grep -oE '(Handle|HandleFunc)\("[^"]*"' internal/container/container.go | sort -u

# Endpoint đã ghi trong tài liệu
grep -oE '^#{2,4} `?(GET|POST|PUT|PATCH|DELETE) [^`]*' docs/API.md | sort
```

Đối chiếu hai danh sách. Endpoint có trong code mà không có trong `docs/API.md` là thiếu;
ngược lại là tài liệu mô tả thứ không còn tồn tại.

Kiểm tra thêm với endpoint đã đổi: request body, response, mã lỗi trong tài liệu còn đúng
không.

### 2.2 Cơ sở dữ liệu

Nếu có file mới trong `internal/database/migrations/`:

```bash
ls internal/database/migrations/
grep -rn "ALTER TABLE\|CREATE TABLE\|CREATE INDEX" internal/database/migrations/<file-mới>
```

Kiểm tra `docs/detail_design/01-database.md`:

- Bảng/cột mới đã có trong tài liệu chưa?
- Có ghi **lý do** của quyết định thiết kế không (vì sao JSON thay vì cột riêng, vì sao
  denormalize)? Đây là phần tài liệu có giá trị nhất và cũng hay thiếu nhất.
- Số lượng bảng nêu trong `docs/README.md` và `docs/detail_design/README.md` còn đúng không?

### 2.3 Biến môi trường

Nếu có sửa `internal/config/`:

```bash
# Biến đọc trong code
grep -oE 'getEnv[A-Za-z]*\("[A-Z_]+"' internal/config/builder.go | grep -oE '"[A-Z_]+"' | tr -d '"' | sort -u

# Biến khai trong .env.example
grep -oE '^[A-Z_]+=' .env.example | tr -d '=' | sort -u

# Biến có ghi trong tài liệu cấu hình
grep -oE '`[A-Z_]{3,}`' docs/deploys/02-configuration.md | tr -d '`' | sort -u
```

Ba danh sách phải khớp nhau. Biến mới **bắt buộc** có đủ ở cả ba chỗ — thiếu ở
`.env.example` thì người deploy không biết mà đặt.

### 2.4 Package / kiến trúc

Nếu thêm package mới trong `internal/`, hoặc sửa chuỗi middleware:

- `docs/MODULES.md` — package mới đã có mục mô tả trách nhiệm và phụ thuộc chưa?
- `docs/ARCHITECTURE.md` — sơ đồ chuỗi middleware và luồng request còn đúng không?

### 2.5 Frontend

Nếu thêm trang, route, hoặc component lớn trong `web/src/`:

```bash
ls web/src/components/public/ | wc -l
ls web/src/pages/public/ web/src/pages/admin/
```

- `docs/detail_design/06-frontend.md` — số component, danh sách trang, sơ đồ router còn
  đúng không?
- Route mới có dùng `React.lazy` + `RouteFallback` không (quy tắc mục 6 `CLAUDE.md`)?
- Chỗ hiển thị ảnh tác phẩm có đi qua `lib/artworkImage.ts` không?

### 2.6 Deploy / vận hành

Nếu sửa `deploy/`, `internal/container/container.go` phần khởi động, hoặc systemd/nginx:

- `docs/deploys/00-tu-dau-den-cuoi.md` — bước nào trong hướng dẫn không còn đúng?
- `docs/deploys/03-operations.md` — lệnh vận hành, cách chẩn đoán sự cố còn khớp không?
- `deploy/README.md` — cờ dòng lệnh của script còn đúng không?

## Bước 3 — Đối chiếu với kế hoạch

Luôn chạy bước này, dù thay đổi thuộc loại nào.

```bash
grep -n "^### P[0-9]" docs/plan/02-roadmap.md
```

Với mỗi mục roadmap liên quan đến thay đổi vừa làm:

- **Việc vừa làm có nằm trong roadmap không?** Nếu không, đó là phạm vi phát sinh — hỏi
  người dùng có muốn thêm vào roadmap không, đừng tự thêm.
- **Đã hoàn thành mục nào chưa?** Nếu rồi, đánh dấu theo mẫu đã dùng:

  ```markdown
  ### ~~P1.5 — Tên mục~~ ✅ Xong YYYY-MM-DD
  ```

  Đánh dấu `- [x]` cho từng tiêu chí hoàn thành, và **ghi kết quả đo được thật** thay vì
  chỉ nói "đã xong".

- Cập nhật `docs/plan/01-current-state.md`: chuyển mục từ "Đang làm dở 🚧" sang "Đã hoàn
  thành ✅", xoá khỏi "Nợ kỹ thuật ⚠️" nếu đã trả xong.
- Cập nhật khối "Thứ tự thực hiện đề xuất" ở cuối `02-roadmap.md`.
- Phát hiện rủi ro mới thì thêm vào `docs/plan/03-risks.md`.

## Bước 4 — Kiểm chứng những gì tài liệu khẳng định

Tài liệu dự án này có nhiều khẳng định kiểm chứng được bằng lệnh. Chạy lại và sửa nếu lệch:

```bash
go build ./...    # docs/plan nói "biên dịch sạch"
go test ./...     # docs/plan nói "toàn bộ gói đạt"

ls internal/database/migrations/ | wc -l        # số migration
ls web/src/components/public/ | wc -l           # số component public
grep -c '^[A-Z_]*=' .env.example                # số biến môi trường
```

Nếu `go test` không đạt, **đó là kết quả phải ghi vào tài liệu**, không phải thứ để giấu.
Ghi rõ test nào hỏng, vì sao, và thuộc phần nào.

⚠️ **Con số đã ghi trong tài liệu**: tìm và kiểm tra lại.

```bash
grep -rnE '[0-9]+ (bảng|endpoint|file migration|component|biến)' docs/
```

Con số nào không còn đúng thì sửa — hoặc tốt hơn, viết lại câu để không cần số.

## Bước 5 — Kiểm tra tham chiếu chéo

Tài liệu dẫn chiếu code theo dạng `file.go:dòng`. Số dòng lệch sau mỗi lần sửa.

```bash
grep -rnoE '`?[a-z_/]+\.(go|ts|tsx):[0-9]+`?' docs/ | head -40
```

Với các tham chiếu trỏ vào file vừa sửa, mở ra kiểm tra dòng đó còn đúng nội dung không.

Kiểm tra link nội bộ không gãy:

```bash
grep -rhoE '\]\(\./[^)]+\)' docs/ | tr -d '](.' | sort -u
```

## Bước 6 — Báo cáo

Trình bày kết quả dạng bảng, đừng chỉ nói "đã kiểm tra xong":

| Tài liệu | Tình trạng | Việc cần làm |
|---|---|---|
| `docs/API.md` | ⚠️ Lệch | Thiếu `PATCH /api/v1/admin/...` |
| `docs/plan/02-roadmap.md` | ⚠️ Lệch | P1.2 đã xong, chưa đánh dấu |
| `docs/detail_design/01-database.md` | ✅ Khớp | — |

Sau đó **sửa luôn** những chỗ lệch, đừng chỉ báo cáo. Rồi commit tài liệu **cùng** với code.

## Nguyên tắc

1. **Code là đúng.** Tài liệu lệch thì sửa tài liệu, trừ khi tài liệu mô tả một yêu cầu
   nghiệp vụ thật mà code làm sai.
2. **Không viết con số không đếm được.** Đếm lại, hoặc diễn đạt không cần số.
3. **Ghi lý do, không chỉ ghi sự việc.** "Dùng JSON thay 6 cột riêng vì số biến thể mỗi ảnh
   không cố định" có giá trị hơn "cột variants kiểu JSON".
4. **Không giấu kết quả xấu.** Test hỏng, tính năng làm dở, nợ kỹ thuật — ghi thẳng vào
   `01-current-state.md`, kèm bối cảnh để người đọc biết mức độ nghiêm trọng.

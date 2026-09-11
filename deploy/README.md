# `deploy/` — File cấu hình mẫu

Thư mục này **chỉ chứa file mẫu để copy lên VPS**. Không còn script nào.

| File | Copy đi đâu |
|---|---|
| `systemd/s3-upload-tool.service` | `/etc/systemd/system/` |
| `nginx/pictures.vaschools.edu.vn.conf` | `/etc/nginx/sites-available/` rồi tạo symlink sang `sites-enabled/` |

**Quy trình deploy đầy đủ nằm ở [docs/deploys/00-tu-dau-den-cuoi.md](../docs/deploys/00-tu-dau-den-cuoi.md)** —
từ VPS trắng đến HTTPS chạy, giai đoạn H là phần build và cài đặt.

## Vì sao không còn script

Trước đây có `deploy.sh`, `preflight.sh`, `setup-https.sh` và `status.sh`. Chúng đã bị xoá
ngày 2026-09-07.

Lý do: toàn bộ quy trình build, migration, cài systemd và Nginx bị dồn vào một dòng
`sudo bash deploy/deploy.sh`. Khi script hỏng — hoặc khi cần làm khác đi một chút — người vận
hành không có tài liệu nào để bám vào, vì tài liệu chỉ nói "chạy script". Giờ từng bước nằm
tường minh trong tài liệu, kèm giải thích bước đó làm gì và hỏng thì nhận ra bằng dấu hiệu nào.

## Hai điều cần biết khi sửa file mẫu

**`WorkingDirectory` trong systemd unit là bắt buộc.** Ứng dụng đọc ba thứ theo đường dẫn
tương đối: migration (`internal/database/migrations`), giao diện (`web/dist`), và thư mục
file tạm (`uploads`). Đặt sai thư mục làm việc thì cả ba hỏng — giao diện trả JSON thay vì
trang web là dấu hiệu dễ thấy nhất.

**`ReadWritePaths` phải liệt kê đủ thư mục cần ghi.** `ProtectSystem=strict` khiến toàn hệ
thống chỉ đọc. Nếu sau này ứng dụng cần ghi vào thư mục mới, phải thêm vào dòng này — `chown`
đúng vẫn sẽ bị từ chối quyền nếu thiếu.

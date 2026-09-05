# Hướng dẫn triển khai → đã chuyển sang `deploys/`

> ⚠️ **Tài liệu này đã được thay thế.** Nội dung cũ hướng dẫn deploy bằng Docker và nhắc tới
> PostgreSQL — **cả hai đều không còn đúng**: `Dockerfile` đã bị xoá khỏi repo, và cơ sở dữ
> liệu là MySQL 8. Giữ lại file này chỉ để các liên kết cũ không gãy.

Xem tài liệu triển khai hiện hành tại **[deploys/](./deploys/README.md)**:

| Bạn cần | Đọc |
|---|---|
| Triển khai từ đầu lên VPS | [deploys/01-vps-systemd.md](./deploys/01-vps-systemd.md) |
| Tra cứu biến môi trường | [deploys/02-configuration.md](./deploys/02-configuration.md) |
| Vận hành, xử lý sự cố, backup | [deploys/03-operations.md](./deploys/03-operations.md) |
| Mở quyền đọc ảnh trên S3 | [S3-PUBLIC-READ.md](./S3-PUBLIC-READ.md) |
| Danh sách kiểm tra trước production | [deploys/README.md](./deploys/README.md) |

Phương án triển khai chính thức: **binary Go + systemd + Nginx + Certbot** trên một VPS.

# Triển khai — Tổng quan

Tài liệu triển khai cho `pictures.vaschools.edu.vn`.

## Phương án hiện hành

**Binary Go + systemd + Nginx + Certbot** trên một VPS Linux.

> ⚠️ **Docker không còn được hỗ trợ.** `Dockerfile` đã bị xoá khỏi repo (xem `git status`).
> Tài liệu cũ từng mô tả hai lựa chọn Docker/binary — hiện chỉ còn một. Đừng làm theo hướng
> dẫn Docker trong bất kỳ bản sao tài liệu cũ nào.

Lý do chọn binary trực tiếp: Go biên dịch ra một file tĩnh không phụ thuộc runtime, nên
Docker chỉ thêm một tầng trừu tượng mà không giải quyết vấn đề nào ở quy mô một VPS. systemd
lo khởi động lại khi lỗi, Nginx lo TLS và giới hạn kích thước upload.

## Danh mục tài liệu

| Tài liệu | Nội dung |
|---|---|
| [00-tu-dau-den-cuoi.md](./00-tu-dau-den-cuoi.md) | **Deploy lần đầu — đọc file này.** Cực chi tiết, từ lúc chưa mua VPS: làm cứng server, cài Go/Node/MySQL, tạo IAM user AWS, OAuth, nghiệm thu, backup |
| [01-vps-systemd.md](./01-vps-systemd.md) | Quy trình rút gọn cho người đã quen, khi VPS đã có sẵn Go/Node/MySQL/Nginx |
| [02-configuration.md](./02-configuration.md) | Toàn bộ biến môi trường: ý nghĩa, mặc định, ràng buộc |
| [03-operations.md](./03-operations.md) | Vận hành hằng ngày: log, backup, sự cố, rollback |
| [../S3-PUBLIC-READ.md](../S3-PUBLIC-READ.md) | Bucket policy cho ảnh đọc công khai |

## Kiến trúc khi chạy thật

```text
      Internet
         │  443 (HTTPS, cert Let's Encrypt)
         ▼
   ┌──────────────────────────────────────┐
   │  Nginx                               │
   │   • kết thúc TLS                     │
   │   • client_max_body_size 200m        │
   │   • truyền X-Forwarded-For/Proto     │  ← thiết yếu cho rate limit + OG
   │   • timeout 300s cho upload dài      │
   └────────────────┬─────────────────────┘
                    │ 127.0.0.1:8080
                    ▼
   ┌──────────────────────────────────────┐
   │  systemd: s3-upload-tool.service     │
   │   User=appuser (không có shell)      │
   │   Restart=on-failure, RestartSec=5   │
   │   EnvironmentFile=/opt/.../.env      │
   └────────┬──────────────────┬──────────┘
            ▼                  ▼
      ┌──────────┐      ┌─────────────┐
      │ MySQL 8  │      │  Amazon S3  │
      │ localhost│      │  ap-southeast-1
      └──────────┘      └─────────────┘
```

## Bố cục thư mục trên VPS

```text
/opt/s3-upload-tool/
├── server              binary Go đã build
├── .env                cấu hình production  ⚠️ không nằm trong git
├── web/dist/           frontend đã build
├── storage/
│   ├── logs/           app-YYYY-MM-DD.log (xoay theo ngày)
│   └── backups/        dump SQL trước các thao tác rủi ro
├── uploads/            file tạm + chunk đang dở (tự dọn)
├── wp-uploads/         ảnh resize kiểu WordPress (nếu bật)
└── deploy/             script và file cấu hình mẫu
```

## Triển khai nhanh (khi đã cài đặt lần đầu xong)

```bash
cd /opt/s3-upload-tool
sudo bash deploy/deploy.sh
```

Script chạy lại nhiều lần được: `git pull` → build web → build binary → cài/khởi động lại
service → cài lại vhost Nginx → kiểm tra sức khoẻ. Chi tiết từng bước trong
[01-vps-systemd.md](./01-vps-systemd.md).

## Danh sách kiểm tra trước khi lên production

Toàn bộ mục dưới đây phải xong. Bốn mục đầu server sẽ **tự chặn khởi động** nếu sai.

- [ ] `APP_ENV=production`
- [ ] `API_REQUIRE_KEY=true` và `API_KEY` là chuỗi ngẫu nhiên thật — ⚠️ đọc cảnh báo về
      ảnh hưởng tới trang public trong [02-configuration.md](./02-configuration.md)
- [ ] `CORS_ORIGINS=https://pictures.vaschools.edu.vn` (**không** dùng `*`)
- [ ] `CSRF_SECURE_COOKIE=true` (sau khi đã có HTTPS)
- [ ] `DATABASE_URL` trỏ đúng MySQL production, mật khẩu mạnh
- [ ] `GOOGLE_REDIRECT_URL` khớp **chính xác** URI đã khai trong Google Cloud Console
- [ ] `ADMIN_ALLOWED_EMAILS` đúng danh sách ban tổ chức
- [ ] `S3_USE_PRESIGNED_URL=false` (ảnh public phải sống lâu dài)
- [ ] Bucket policy cho phép đọc công khai — xem [S3-PUBLIC-READ.md](../S3-PUBLIC-READ.md)
- [ ] `curl https://pictures.vaschools.edu.vn/api/v1/health` trả `200`
- [ ] Không còn vhost Nginx nào khác trả lời cho tên miền này
- [ ] Thử upload một ảnh thật qua giao diện trên tên miền

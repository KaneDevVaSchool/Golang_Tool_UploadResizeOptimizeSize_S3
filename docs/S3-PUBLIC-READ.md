# Cấu hình S3 cho ảnh triển lãm (public read)

## Vấn đề đã sửa

Trước đây `.env` đặt `S3_USE_PRESIGNED_URL=true` với `S3_PRESIGNED_URL_EXPIRY=300`.

Đơn vị của biến này là **phút** (xem `internal/config/config.go`, trường
`PresignedURLExpiry int // minutes`), nên URL ký được có hạn dùng 5 giờ. Vấn đề nằm ở
chỗ URL đã ký này được **lưu vĩnh viễn** vào cột `artworks.s3_url` ngay lúc upload
(`internal/service/artwork_service.go` → `CreateArtworkFromUpload`), rồi API công khai
trả đúng chuỗi đó cho trình duyệt.

Hệ quả: mọi tác phẩm sẽ hiển thị bình thường trong khoảng 5 giờ đầu, sau đó S3 trả
`403 Forbidden` và cả kho ảnh trắng xoá — mà không có lỗi nào trong log ứng dụng.

Presigned URL sinh ra để cấp quyền truy cập **tạm thời** vào object **private**. Ảnh dự
thi ở đây là nội dung công khai, ai cũng xem được, nên không có gì để bảo vệ; đổi lại
việc ký URL khiến ảnh vừa hết hạn vừa không cache được (mỗi lần ký ra một URL khác nhau
nên trình duyệt và CDN đều coi là tài nguyên mới).

## Cách làm hiện tại

`.env` đặt `S3_USE_PRESIGNED_URL=false`. Nhánh này đã có sẵn trong code
(`internal/service/upload_service.go` → `objectURL()` → `utils.BuildS3ObjectURL`), sinh
URL tĩnh dạng:

```
https://vaschools-s3-bucket.s3.ap-southeast-1.amazonaws.com/vaschools-uploads/<key>
```

URL này không hết hạn và cache được vô thời hạn.

## Việc cần làm trên AWS Console (ngoài code)

Để URL tĩnh truy cập được, bucket phải cho phép đọc công khai **giới hạn trong prefix
chứa ảnh**, chứ không mở toàn bucket.

### 1. Tắt chặn public access ở mức cần thiết

S3 → bucket `vaschools-s3-bucket` → tab **Permissions** → **Block public access
(bucket settings)** → **Edit**:

- Bỏ chọn `Block public access to buckets and objects granted through new public bucket
  or access point policies`
- Bỏ chọn `Block public and cross-account access to buckets and objects through any
  public bucket or access point policies`
- **Giữ nguyên** hai tuỳ chọn liên quan tới *ACL* (đang bật) — ta cấp quyền bằng bucket
  policy, không dùng ACL. Biến `S3_USE_ACL` cũng đang để `false`.

### 2. Gắn bucket policy

Tab **Permissions** → **Bucket policy** → **Edit**, dán:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadArtworkImages",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::vaschools-s3-bucket/vaschools-uploads/*"
    }
  ]
}
```

Chỉ `s3:GetObject`, và chỉ dưới prefix `vaschools-uploads/`. Không ai ghi/xoá/liệt kê
được; phần còn lại của bucket không bị ảnh hưởng.

### 3. Kiểm chứng

```bash
curl -I "https://vaschools-s3-bucket.s3.ap-southeast-1.amazonaws.com/vaschools-uploads/<key-bat-ky>"
```

Phải trả `HTTP/1.1 200 OK`. Nếu trả `403`, kiểm tra lại bước 1 (block public access
thường là nguyên nhân) trước khi ngờ tới policy.

## Ghi chú backfill

Tính tới thời điểm sửa, bảng `artworks` chỉ chứa dữ liệu demo (ảnh seed local
`/images/seed-*.jpg` và vài link placeholder), **không có bản ghi nào mang URL presigned
thật**, nên không cần chạy backfill. Nếu về sau phát hiện bản ghi cũ có URL chứa tham số
`X-Amz-`, dựng lại URL sạch từ `s3_key`:

```sql
-- Kiểm tra trước
SELECT COUNT(*) FROM artworks WHERE s3_url LIKE '%X-Amz-%';

-- Sửa (nhớ backup vào storage/backups/ trước)
UPDATE artworks
SET s3_url = CONCAT('https://vaschools-s3-bucket.s3.ap-southeast-1.amazonaws.com/', s3_key)
WHERE s3_url LIKE '%X-Amz-%';
```

## Bước tiếp theo: CDN

URL tĩnh đã sẵn sàng để đặt CloudFront lên trước. Khi làm, trỏ origin về bucket rồi đổi
`utils.BuildS3ObjectURL` (hoặc thêm biến `S3_PUBLIC_BASE_URL`) sang domain CloudFront —
không phải sửa gì trong dữ liệu đã lưu nếu dùng biến cấu hình ngay từ đầu.

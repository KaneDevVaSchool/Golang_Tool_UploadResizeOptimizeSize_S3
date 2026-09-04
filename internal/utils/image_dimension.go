package utils

import (
	"image"
	// Đăng ký decoder cho image.DecodeConfig - chỉ cần đọc header, không
	// decode toàn bộ ảnh nên không cần golang.org/x/image/webp ở đây (webp
	// header không được stdlib hỗ trợ, chấp nhận width/height rỗng cho webp).
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

// DecodeImageDimensions đọc header ảnh (không decode toàn bộ) để lấy
// width/height, dùng khi tạo artwork từ ảnh đã upload. reader phải hỗ trợ
// Seek để caller đọc lại từ đầu sau khi gọi hàm này (dùng io.Seeker riêng,
// không tự seek trong hàm - caller kiểm soát vị trí đọc tiếp theo).
// Trả (0, 0, false) nếu không decode được (định dạng lạ, file hỏng) - không
// coi là lỗi cứng vì width/height chỉ là metadata hiển thị, không bắt buộc.
func DecodeImageDimensions(r io.Reader) (width, height int, ok bool) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

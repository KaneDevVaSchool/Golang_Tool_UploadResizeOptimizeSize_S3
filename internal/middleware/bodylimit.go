package middleware

import (
	"net/http"
	"strings"
)

// Giới hạn kích thước body theo loại đường dẫn.
//
// Nginx đặt client_max_body_size 200m ở mức server để đường upload ảnh đi lọt,
// nhưng như vậy nghĩa là MỌI endpoint đều nhận được body 200MB - kể cả
// POST /api/v1/public/artworks/{id}/comments. Một người có thể gửi vài chục
// request bình luận với body khổng lồ để ép server đọc và cấp phát bộ nhớ,
// hoàn toàn hợp lệ dưới mắt rate limit vì số request vẫn thấp.
//
// http.MaxBytesReader dừng việc đọc ngay khi vượt ngưỡng và trả lỗi cho
// handler, nên body quá khổ không bao giờ được nạp trọn vẹn vào bộ nhớ.

// DefaultJSONBodyLimit là trần cho endpoint JSON. Bình luận giới hạn vài trăm
// ký tự, payload admin lớn nhất là danh sách ID để xoá/gắn tiêu biểu hàng
// loạt - 1MB đã rộng rãi gấp nhiều lần nhu cầu thật.
const DefaultJSONBodyLimit int64 = 1 << 20 // 1MB

// BodyLimitMiddleware áp trần cho các đường không phải upload.
//
// uploadLimit là trần cho đường upload ảnh (truyền từ
// UPLOAD_ABSOLUTE_MAX_MB), jsonLimit cho phần còn lại.
func BodyLimitMiddleware(jsonLimit, uploadLimit int64) func(http.Handler) http.Handler {
	if jsonLimit <= 0 {
		jsonLimit = DefaultJSONBodyLimit
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// GET/HEAD/OPTIONS không mang body đáng kể - bọc thêm chỉ tốn công.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			limit := jsonLimit
			if uploadLimit > 0 && isUploadPath(r.URL.Path) {
				limit = uploadLimit
			}

			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isUploadPath nhận diện các đường thực sự nhận file ảnh.
func isUploadPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/upload") ||
		strings.HasPrefix(path, "/api/v1/admin/artworks/bulk-upload") ||
		path == "/api/v1/admin/artworks" ||
		strings.HasPrefix(path, "/api/v1/admin/artworks/")
}

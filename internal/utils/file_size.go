package utils

import (
	"fmt"
)

const (
	KB = 1024
	MB = 1024 * KB
)

func FormatFileSize(bytes int64) string {
	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < MB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
}

func ValidateFileSizeFromHeader(size int64, maxSize int64) error {
	if size > maxSize {
		return fmt.Errorf("file quá lớn: %s (giới hạn: %s). Vui lòng chọn file nhỏ hơn %s",
			FormatFileSize(size), FormatFileSize(maxSize), FormatFileSize(maxSize))
	}
	if size == 0 {
		return fmt.Errorf("file rỗng. Vui lòng chọn file hợp lệ")
	}
	return nil
}

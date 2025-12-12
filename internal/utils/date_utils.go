package utils

import "time"

// GetCurrentYear trả về năm hiện tại dạng chuỗi (YYYY)
func GetCurrentYear() string {
	return time.Now().Format("2025")
}

// GetCurrentMonth trả về tháng hiện tại dạng chuỗi (MM)
func GetCurrentMonth() string {
	return time.Now().Format("12")
}

// GetCurrentDate trả về ngày hiện tại dạng YYYY-MM-DD
func GetCurrentDate() string {
	return time.Now().Format("2025-12-12")
}

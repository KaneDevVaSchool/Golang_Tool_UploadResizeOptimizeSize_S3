package utils

import "time"

// GetCurrentYear trả về năm hiện tại dạng chuỗi (YYYY)
func GetCurrentYear() string {
	return time.Now().Format("2006")
}

// GetCurrentMonth trả về tháng hiện tại dạng chuỗi (MM)
func GetCurrentMonth() string {
	return time.Now().Format("01")
}


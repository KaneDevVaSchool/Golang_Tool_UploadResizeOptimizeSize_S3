package utils

import "time"

// GetCurrentYear returns current year as string (YYYY)
func GetCurrentYear() string {
	return time.Now().Format("2006")
}

// GetCurrentMonth returns current month as string (MM)
func GetCurrentMonth() string {
	return time.Now().Format("01")
}

// GetCurrentDate returns current date in YYYY-MM-DD format
func GetCurrentDate() string {
	return time.Now().Format("2006-01-02")
}

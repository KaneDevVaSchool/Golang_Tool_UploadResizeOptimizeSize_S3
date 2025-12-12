package handlers

import (
	"strings"

	"s3-upload-tool/internal/service"
)

// errorMapper map internal errors sang user-friendly messages
type errorMapper struct {
	mappings map[string]string
}

var defaultErrorMapper = &errorMapper{
	mappings: map[string]string{
		"method not allowed":    "Invalid request method",
		"parse multipart form":  "Unable to parse form data",
		"get file from request": "Unable to read file from request",
		"validate file":         "File validation failed",
		"failed to create temp": "Unable to process file",
		"failed to copy file":   "Unable to process file",
		"failed to upload":      "Unable to upload file",
		"file size":             "File size validation failed",
		"invalid file type":     "File type not supported",
		"database":              "Database operation failed",
		"transaction":           "Transaction failed",
	},
}

// mapError chuyển đổi internal error sang user-friendly message
func (m *errorMapper) mapError(err error) string {
	if err == nil {
		return "An unexpected error occurred"
	}

	errStr := strings.ToLower(err.Error())

	if fileSizeErr, ok := err.(*service.FileSizeError); ok {
		return fileSizeErr.Error()
	}

	if err == service.ErrInvalidFileFormat {
		return "File type not supported"
	}

	for pattern, message := range m.mappings {
		if strings.Contains(errStr, pattern) {
			return message
		}
	}

	// ! Trả về message generic để tránh leak thông tin nội bộ
	return "An error occurred processing your request"
}

// sanitizeError đảm bảo error messages không leak thông tin nội bộ
func sanitizeError(err error) string {
	return defaultErrorMapper.mapError(err)
}

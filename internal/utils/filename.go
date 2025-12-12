package utils

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	MaxFilenameLength = 255
)

// SanitizeFilename sanitize filename để chống path traversal và các lỗ hổng bảo mật
func SanitizeFilename(filename string) (string, error) {
	if len(filename) == 0 {
		return "", errors.New("filename cannot be empty")
	}

	filename = filepath.Base(filename)

	// ! Xóa path traversal attempts
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")

	// ! Xóa control characters và các ký tự nguy hiểm
	var builder strings.Builder
	for _, r := range filename {
		if unicode.IsControl(r) || r == '<' || r == '>' || r == ':' || r == '"' || r == '|' || r == '?' || r == '*' {
			continue
		}
		builder.WriteRune(r)
	}
	filename = builder.String()
	filename = strings.TrimSpace(filename)

	if len(filename) == 0 {
		return "", errors.New("filename becomes empty after sanitization")
	}

	if len(filename) > MaxFilenameLength {
		ext := filepath.Ext(filename)
		maxNameLen := MaxFilenameLength - len(ext)
		if maxNameLen <= 0 {
			return "", errors.New("filename extension too long")
		}
		filename = filename[:maxNameLen] + ext
	}

	return filename, nil
}

// ValidateFilename validate filename trước khi xử lý
func ValidateFilename(filename string) error {
	if len(filename) == 0 {
		return errors.New("filename cannot be empty")
	}

	if len(filename) > MaxFilenameLength {
		return errors.New("filename too long (max 255 characters)")
	}

	// ! Kiểm tra path traversal attempts
	if strings.Contains(filename, "..") {
		return errors.New("filename contains invalid path components")
	}

	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return errors.New("filename contains path separators")
	}

	// ! Kiểm tra dangerous characters
	for _, char := range []rune{'<', '>', ':', '"', '|', '?', '*'} {
		if strings.ContainsRune(filename, char) {
			return errors.New("filename contains invalid characters")
		}
	}

	return nil
}

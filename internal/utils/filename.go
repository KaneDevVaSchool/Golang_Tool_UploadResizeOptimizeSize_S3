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

// SanitizeFilename sanitizes a filename to prevent path traversal and other security issues
func SanitizeFilename(filename string) (string, error) {
	if len(filename) == 0 {
		return "", errors.New("filename cannot be empty")
	}

	// Remove any path components (only keep base name)
	filename = filepath.Base(filename)

	// Remove path traversal attempts
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")

	// Remove control characters and other dangerous characters
	var builder strings.Builder
	for _, r := range filename {
		if unicode.IsControl(r) || r == '<' || r == '>' || r == ':' || r == '"' || r == '|' || r == '?' || r == '*' {
			continue
		}
		builder.WriteRune(r)
	}
	filename = builder.String()

	// Trim whitespace
	filename = strings.TrimSpace(filename)

	if len(filename) == 0 {
		return "", errors.New("filename becomes empty after sanitization")
	}

	// Limit length
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

// ValidateFilename validates filename before processing
func ValidateFilename(filename string) error {
	if len(filename) == 0 {
		return errors.New("filename cannot be empty")
	}

	if len(filename) > MaxFilenameLength {
		return errors.New("filename too long (max 255 characters)")
	}

	// Check for path traversal attempts
	if strings.Contains(filename, "..") {
		return errors.New("filename contains invalid path components")
	}

	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return errors.New("filename contains path separators")
	}

	// Check for dangerous characters
	for _, char := range []rune{'<', '>', ':', '"', '|', '?', '*'} {
		if strings.ContainsRune(filename, char) {
			return errors.New("filename contains invalid characters")
		}
	}

	return nil
}

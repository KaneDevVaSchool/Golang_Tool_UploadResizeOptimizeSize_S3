package utils

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// ValidateFileContent validates that file content matches the file extension
func ValidateFileContent(file multipart.File, filename string) error {
	// Read first 512 bytes for MIME detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return errors.New("unable to read file content")
	}

	// Reset file pointer for later use
	if _, seekErr := file.Seek(0, 0); seekErr != nil {
		return errors.New("unable to reset file pointer")
	}

	if n == 0 {
		return errors.New("file is empty")
	}

	// Detect MIME type from content
	detectedType := http.DetectContentType(buffer[:n])

	// Get expected MIME type from extension
	expectedType := GetContentType(filename)

	// Extract base type (e.g., "image" from "image/jpeg")
	detectedBase := strings.Split(detectedType, "/")[0]
	expectedBase := strings.Split(expectedType, "/")[0]

	// For images, be more strict
	ext := strings.ToLower(filepath.Ext(filename))
	if IsImage(filename) {
		// Validate image MIME types strictly
		imageMIMEs := map[string][]string{
			".jpg":  {"image/jpeg"},
			".jpeg": {"image/jpeg"},
			".png":  {"image/png"},
			".gif":  {"image/gif"},
			".webp": {"image/webp"},
			".bmp":  {"image/bmp", "image/x-ms-bmp"},
			".svg":  {"image/svg+xml", "text/xml", "application/xml"},
			".ico":  {"image/x-icon", "image/vnd.microsoft.icon"},
		}

		validMIMEs, exists := imageMIMEs[ext]
		if !exists {
			return errors.New("unsupported image format")
		}

		// Check if detected type matches any valid MIME for this extension
		matched := false
		for _, validMIME := range validMIMEs {
			if strings.HasPrefix(detectedType, validMIME) || detectedType == validMIME {
				matched = true
				break
			}
		}

		if !matched {
			return errors.New("file content does not match image type")
		}
	} else {
		// For other file types, check base type matches
		if detectedBase != expectedBase && detectedBase != "application" {
			return errors.New("file content does not match file type")
		}
	}

	return nil
}

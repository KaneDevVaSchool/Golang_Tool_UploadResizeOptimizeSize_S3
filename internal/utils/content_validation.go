package utils

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// ValidateFileContent validate file content có khớp với extension không
func ValidateFileContent(file multipart.File, filename string) error {
	// * Đọc 512 bytes đầu để detect MIME type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return errors.New("unable to read file content")
	}

	if _, seekErr := file.Seek(0, 0); seekErr != nil {
		return errors.New("unable to reset file pointer")
	}

	if n == 0 {
		return errors.New("file is empty")
	}

	detectedType := http.DetectContentType(buffer[:n])
	expectedType := GetContentType(filename)

	detectedBase := strings.Split(detectedType, "/")[0]
	expectedBase := strings.Split(expectedType, "/")[0]

	ext := strings.ToLower(filepath.Ext(filename))
	if IsImage(filename) {
		// ! Validate image MIME types chặt chẽ hơn
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
		// * Với các file type khác, chỉ kiểm tra base type
		if detectedBase != expectedBase && detectedBase != "application" {
			return errors.New("file content does not match file type")
		}
	}

	return nil
}

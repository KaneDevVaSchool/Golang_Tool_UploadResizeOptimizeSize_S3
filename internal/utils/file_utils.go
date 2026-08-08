package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// IsImage kiểm tra file có phải là ảnh không
func IsImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".ico"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsDocument kiểm tra file có phải là document không
func IsDocument(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".rtf", ".odt", ".ods", ".odp"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsVideo kiểm tra file có phải là video không
func IsVideo(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv", ".m4v", ".3gp"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsAudio kiểm tra file có phải là audio không
func IsAudio(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma", ".opus"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsArchive kiểm tra file có phải là archive không
func IsArchive(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsAllowedFileType kiểm tra file type có được phép không
func IsAllowedFileType(filename string) bool {
	return IsImage(filename) || IsDocument(filename) || IsVideo(filename) || IsAudio(filename) || IsArchive(filename)
}

// GetFileCategory trả về category của file
func GetFileCategory(filename string) string {
	if IsImage(filename) {
		return "images"
	}
	if IsDocument(filename) {
		return "documents"
	}
	if IsVideo(filename) {
		return "videos"
	}
	if IsAudio(filename) {
		return "audio"
	}
	if IsArchive(filename) {
		return "archives"
	}
	return "files"
}

// GetContentType trả về MIME type dựa trên extension của file
func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	}

	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt":
		return "text/plain"
	case ".rtf":
		return "application/rtf"
	case ".odt":
		return "application/vnd.oasis.opendocument.text"
	case ".ods":
		return "application/vnd.oasis.opendocument.spreadsheet"
	case ".odp":
		return "application/vnd.oasis.opendocument.presentation"
	}

	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".m4v":
		return "video/x-m4v"
	case ".3gp":
		return "video/3gpp"
	}

	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	case ".wma":
		return "audio/x-ms-wma"
	case ".opus":
		return "audio/opus"
	}

	switch ext {
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	case ".7z":
		return "application/x-7z-compressed"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	case ".bz2":
		return "application/x-bzip2"
	case ".xz":
		return "application/x-xz"
	}

	return "application/octet-stream"
}

// GenerateS3Key tạo S3 key duy nhất với timestamp và random component.
// basePath (nếu có) được gắn phía trước, ví dụ: vaschools-uploads/images/...
// Files được tổ chức theo category: images/, documents/, videos/, audio/, archives/, files/
func GenerateS3Key(filename string, basePath string) string {
	timestamp := time.Now().Format("20060102-150405")
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	name = strings.ReplaceAll(name, " ", "-")

	category := GetFileCategory(filename)

	var key string
	// * Random component để tránh collision trong concurrent scenarios
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err == nil {
		randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:12]
		key = fmt.Sprintf("%s/%s-%s-%s%s", category, name, timestamp, randomStr, ext)
	} else {
		// ! Fallback nếu random generation fail
		key = fmt.Sprintf("%s/%s-%s-%d%s", category, name, timestamp, time.Now().UnixNano(), ext)
	}

	basePath = strings.Trim(strings.TrimSpace(basePath), "/")
	if basePath == "" {
		return key
	}
	return basePath + "/" + key
}

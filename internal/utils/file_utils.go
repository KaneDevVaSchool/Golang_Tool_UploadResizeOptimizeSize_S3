package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// IsImage checks if file is an image
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

// IsDocument checks if file is a document
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

// IsVideo checks if file is a video
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

// IsAudio checks if file is an audio file
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

// IsArchive checks if file is an archive
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

// IsAllowedFileType checks if file type is allowed
func IsAllowedFileType(filename string) bool {
	return IsImage(filename) || IsDocument(filename) || IsVideo(filename) || IsAudio(filename) || IsArchive(filename)
}

// GetFileCategory returns the category of the file
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
	return "files" // Default category
}

// GetContentType returns MIME type based on file extension
func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Images
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

	// Documents
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

	// Videos
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

	// Audio
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

	// Archives
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

	// Default: application/octet-stream
	return "application/octet-stream"
}

// GenerateS3Key generates a unique S3 key with timestamp and random component
// Files are organized by category: images/, documents/, videos/, audio/, archives/, files/
func GenerateS3Key(filename string) string {
	timestamp := time.Now().Format("20060102-150405")
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	name = strings.ReplaceAll(name, " ", "-")

	// Get file category for organization
	category := GetFileCategory(filename)

	// Add random component to prevent collisions in concurrent scenarios
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err == nil {
		randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:12] // Use first 12 chars
		return fmt.Sprintf("%s/%s-%s-%s%s", category, name, timestamp, randomStr, ext)
	}

	// Fallback if random generation fails (shouldn't happen, but be safe)
	return fmt.Sprintf("%s/%s-%s-%d%s", category, name, timestamp, time.Now().UnixNano(), ext)
}

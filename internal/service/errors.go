package service

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidImageFormat     = errors.New("file phải là ảnh (jpg, jpeg, png, gif, webp)")
	ErrInvalidFileFormat      = errors.New("file type không được hỗ trợ. Hỗ trợ: images (jpg, png, gif, webp), documents (pdf, doc, docx, xls, xlsx), videos (mp4, avi, mov), audio (mp3, wav), archives (zip, rar)")
	ErrCreateTempFile         = errors.New("không thể tạo file tạm")
	ErrSaveFile               = errors.New("không thể lưu file")
	ErrUploadToS3             = errors.New("không thể upload lên S3")
	ErrUnsupportedServiceType = errors.New("unsupported service type")
)

type FileSizeError struct {
	ActualSize int64
	MaxSize    int64
	Message    string
}

func (e *FileSizeError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("file quá lớn: %d bytes (giới hạn: %d bytes)", e.ActualSize, e.MaxSize)
}

func NewFileSizeError(actualSize, maxSize int64, message string) *FileSizeError {
	return &FileSizeError{
		ActualSize: actualSize,
		MaxSize:    maxSize,
		Message:    message,
	}
}

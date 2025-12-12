package utils

import (
	"errors"
	"mime/multipart"
	"net/http"
)

// GetFileFromRequest trích xuất file từ multipart form request
// * Thử field "file" trước, fallback về "image" để backward compatibility
func GetFileFromRequest(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := r.FormFile("file")
	if err != nil {
		file, header, err = r.FormFile("image")
		if err != nil {
			return nil, nil, errors.New("unable to read file from form. Please provide 'file' or 'image' field")
		}
	}
	return file, header, nil
}

// ValidateUploadFile validate file từ multipart request
// * Thực hiện: sanitize filename, validate content type, validate size
func ValidateUploadFile(file multipart.File, header *multipart.FileHeader, maxSize int64) error {
	sanitizedFilename, err := SanitizeFilename(header.Filename)
	if err != nil || sanitizedFilename == "" {
		return errors.New("invalid filename: " + err.Error())
	}
	header.Filename = sanitizedFilename

	if err := ValidateFileContent(file, header.Filename); err != nil {
		return errors.New("file content validation failed: " + err.Error())
	}

	if err := ValidateFileSizeFromHeader(header.Size, maxSize); err != nil {
		return err
	}

	return nil
}

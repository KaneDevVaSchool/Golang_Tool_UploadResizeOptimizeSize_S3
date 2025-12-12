package utils

import (
	"errors"
	"mime/multipart"
	"net/http"
)

// GetFileFromRequest extracts file from multipart form request
// Tries "file" field first, falls back to "image" for backward compatibility
func GetFileFromRequest(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := r.FormFile("file")
	if err != nil {
		// Fallback to "image" for backward compatibility
		file, header, err = r.FormFile("image")
		if err != nil {
			return nil, nil, errors.New("unable to read file from form. Please provide 'file' or 'image' field")
		}
	}
	return file, header, nil
}

// ValidateUploadFile validates file from multipart request
// Performs filename sanitization, content validation, and size validation
func ValidateUploadFile(file multipart.File, header *multipart.FileHeader, maxSize int64) error {
	// Sanitize filename
	sanitizedFilename, err := SanitizeFilename(header.Filename)
	if err != nil || sanitizedFilename == "" {
		return errors.New("invalid filename: " + err.Error())
	}
	header.Filename = sanitizedFilename

	// Validate file content type matches extension
	if err := ValidateFileContent(file, header.Filename); err != nil {
		return errors.New("file content validation failed: " + err.Error())
	}

	// Validate file size
	if err := ValidateFileSizeFromHeader(header.Size, maxSize); err != nil {
		return err
	}

	return nil
}

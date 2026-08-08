package handlers

import (
	"net/http"
	"strings"

	"s3-upload-tool/internal/service"
)

// mapS3UploadError maps S3/config failures to actionable client errors.
func mapS3UploadError(err error) (status int, code, message string) {
	if err == nil {
		return http.StatusInternalServerError, "UPLOAD_FAILED", "Unable to upload file. Please try again later."
	}

	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "permanentredirect"),
		strings.Contains(errStr, "must be addressed using the specified endpoint"),
		strings.Contains(errStr, "authorizationheadermalformed"):
		return http.StatusBadGateway, "S3_REGION_MISMATCH",
			"S3 bucket region does not match AWS_REGION. Update AWS_REGION in .env to the bucket's real region."
	case strings.Contains(errStr, "invalidaccesskeyid"),
		strings.Contains(errStr, "signaturedoesnotmatch"),
		strings.Contains(errStr, "invalidclienttokenid"),
		strings.Contains(errStr, "unrecognizedclientexception"),
		strings.Contains(errStr, "your_access_key"),
		strings.Contains(errStr, "your_secret"):
		return http.StatusBadGateway, "S3_CREDENTIALS",
			"Invalid AWS credentials. Set real AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY in .env."
	case strings.Contains(errStr, "nosuchbucket"):
		return http.StatusBadGateway, "S3_BUCKET_NOT_FOUND",
			"S3 bucket not found. Check S3_BUCKET_NAME in .env."
	case strings.Contains(errStr, "accessdenied"), strings.Contains(errStr, "allaccessdisabled"):
		return http.StatusBadGateway, "S3_ACCESS_DENIED",
			"Access denied to S3 bucket. Check IAM permissions for PutObject / GetObject."
	case strings.Contains(errStr, "failed to upload to s3"), strings.Contains(errStr, "không thể upload lên s3"):
		return http.StatusBadGateway, "S3_UPLOAD_FAILED",
			"Unable to upload to S3. Verify credentials, bucket name, and region."
	default:
		return http.StatusInternalServerError, "UPLOAD_FAILED", "Unable to upload file. Please try again later."
	}
}

// errorMapper map internal errors thành user-friendly messages
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

// mapError chuyển đổi internal error thành user-friendly message
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

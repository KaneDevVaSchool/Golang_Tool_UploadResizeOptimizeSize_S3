package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"runtime"
	"strings"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/metrics"
	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"
	"s3-upload-tool/internal/utils"
)

type APIHandler struct {
	BaseHandler
	uploadService   service.UploadService
	s3Repository    repository.S3Repository
	maxUploadSize   int64
	absoluteMaxSize int64
	db              *database.DB
}

// NewAPIHandler tạo API handler mới
// db có thể nil nếu database không được bật
func NewAPIHandler(
	uploadService service.UploadService,
	s3Repository repository.S3Repository,
	maxUploadSize, absoluteMaxSize int64,
	db *database.DB,
) *APIHandler {
	return &APIHandler{
		uploadService:   uploadService,
		s3Repository:    s3Repository,
		maxUploadSize:   maxUploadSize,
		absoluteMaxSize: absoluteMaxSize,
		db:              db,
	}
}

// UploadResponseData đại diện cho successful upload response
type UploadResponseData struct {
	URL  string `json:"url"`
	Key  string `json:"key"`
	Size int64  `json:"size"`
	Name string `json:"name"`
}

// uploadRequestData chứa validated upload request data
type uploadRequestData struct {
	File   io.ReadCloser
	Header *multipart.FileHeader
}

// processUploadRequest trích xuất và validate file từ multipart request
func (h *APIHandler) processUploadRequest(r *http.Request) (*uploadRequestData, error) {
	if r.Method != http.MethodPost {
		return nil, fmt.Errorf("method not allowed: %s", r.Method)
	}

	log.Printf("[API] Đang parse multipart form với max size: %s", utils.FormatFileSize(h.maxUploadSize))
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		log.Printf("[API] Lỗi parse multipart form: %v", err)
		return nil, fmt.Errorf("parse multipart form: %w", err)
	}

	file, header, err := utils.GetFileFromRequest(r)
	if err != nil {
		log.Printf("[API] Error reading file from form: %v", err)
		return nil, fmt.Errorf("get file from request: %w", err)
	}

	log.Printf("[API] File received - Name: %s, Size: %d bytes (%s)",
		header.Filename, header.Size, utils.FormatFileSize(header.Size))

	if err := utils.ValidateUploadFile(file, header, h.maxUploadSize); err != nil {
		file.Close()
		log.Printf("[API] File validation failed: %v", err)
		return nil, fmt.Errorf("validate file: %w", err)
	}

	return &uploadRequestData{
		File:   file,
		Header: header,
	}, nil
}

// handleUploadError map service errors sang HTTP responses
func (h *APIHandler) handleUploadError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	errorCode := "UPLOAD_FAILED"
	errorMessage := "Unable to upload file. Please try again later."

	if fileSizeErr, ok := err.(*service.FileSizeError); ok {
		statusCode = http.StatusBadRequest
		errorCode = "FILE_TOO_LARGE"
		errorMessage = fileSizeErr.Error()
	} else if err == service.ErrInvalidFileFormat {
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_FILE_TYPE"
		errorMessage = "File type not supported."
	} else {
		statusCode, errorCode, errorMessage = mapS3UploadError(err)
	}

	log.Printf("[API] Upload error: %v (code: %s, status: %d)", err, errorCode, statusCode)
	h.SendError(w, statusCode, errorCode, errorMessage)
}

func (h *APIHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] Nhận request upload từ IP: %s, Method: %s", r.RemoteAddr, r.Method)

	// ! Timeout 5 phút để tránh request treo
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	reqData, err := h.processUploadRequest(r)
	if err != nil {
		// ! Sanitize error để không leak thông tin nội bộ
		errMsg := sanitizeError(err)
		errStr := strings.ToLower(err.Error())

		if strings.Contains(errStr, "method not allowed") {
			h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		} else if strings.Contains(errStr, "parse multipart form") {
			h.SendError(w, http.StatusBadRequest, "INVALID_FORM", "Unable to parse form data. Please try again.")
		} else if strings.Contains(errStr, "get file from request") {
			h.SendError(w, http.StatusBadRequest, "FILE_NOT_FOUND", errMsg)
		} else {
			h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", errMsg)
		}
		return
	}
	defer reqData.File.Close()

	log.Printf("[API] Đang gọi upload service...")
	uploadStart := time.Now()
	result, err := h.uploadService.UploadImage(ctx, reqData.Header.Filename, reqData.File, reqData.Header.Size, h.maxUploadSize)
	uploadDuration := time.Since(uploadStart)

	metrics.GetMetrics().RecordUpload(err == nil, uploadDuration)

	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	log.Printf("[API] Upload thành công! URL: %s, Key: %s", result.URL, result.Key)
	h.SendSuccess(w, UploadResponseData{
		URL:  result.URL,
		Key:  result.Key,
		Size: reqData.Header.Size,
		Name: reqData.Header.Filename,
	})
}

func (h *APIHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ready := true
	health := map[string]interface{}{
		"status":                      "ok",
		"service":                     "s3-upload-api",
		"max_size":                    h.maxUploadSize,
		"max_size_formatted":          utils.FormatFileSize(h.maxUploadSize),
		"absolute_max_size":           h.absoluteMaxSize,
		"absolute_max_size_formatted": utils.FormatFileSize(h.absoluteMaxSize),
	}

	if requestID := middleware.GetRequestID(r.Context()); requestID != "" {
		health["request_id"] = requestID
	}

	checks := map[string]interface{}{
		"disk": checkDiskSpace(),
	}

	if h.db != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := h.db.PingContext(dbCtx); err != nil {
			ready = false
			checks["database"] = map[string]interface{}{
				"status": "error",
				"error":  "Database connectivity check failed",
			}
		} else {
			stats := h.db.GetStats()
			checks["database"] = map[string]interface{}{
				"status":           "ok",
				"open_connections": stats.OpenConnections,
				"in_use":           stats.InUse,
				"idle":             stats.Idle,
				"wait_count":       stats.WaitCount,
			}
		}
	}

	if h.s3Repository != nil {
		connectivityCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := h.s3Repository.CheckConnectivity(connectivityCtx); err != nil {
			ready = false
			checks["s3_service"] = map[string]interface{}{
				"status": "error",
				"error":  "S3 connectivity check failed",
			}
		} else {
			checks["s3_service"] = map[string]interface{}{
				"status": "ok",
			}
		}
	} else if h.uploadService != nil {
		checks["s3_service"] = "initialized"
	} else {
		ready = false
		checks["s3_service"] = map[string]interface{}{
			"status": "error",
			"error":  "S3 upload service is not initialized",
		}
	}

	health["checks"] = checks
	if !ready {
		health["status"] = "unavailable"
		h.SendJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Data: health, Error: &APIError{
			Code:    "NOT_READY",
			Message: "One or more dependencies are unavailable",
		}})
		return
	}
	h.SendSuccess(w, health)
}

// checkDiskSpace kiểm tra dung lượng disk cơ bản
func checkDiskSpace() map[string]interface{} {
	// ! Chưa implement đầy đủ: syscall.Statfs chỉ dùng được trên Unix
	// ? Nên dùng thư viện cross-platform như github.com/ricochet2200/go-disk-usage/du cho production

	if runtime.GOOS == "windows" {
		return map[string]interface{}{
			"status": "ok",
			"note":   "Disk space check not fully implemented on Windows. Consider using cross-platform library.",
		}
	}

	return map[string]interface{}{
		"status": "ok",
		"note":   "Disk space check requires platform-specific implementation. Use cross-platform library for production.",
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"s3-upload-tool/internal/metrics"
	"s3-upload-tool/internal/service"
	"s3-upload-tool/internal/utils"
)

const maxChunkJSONBody = 1 << 20 // 1MB

type initChunkRequest struct {
	Filename  string `json:"filename"`
	TotalSize int64  `json:"total_size"`
}

type initChunkResponse struct {
	UploadID    string `json:"upload_id"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
	TotalSize   int64  `json:"total_size"`
	Filename    string `json:"filename"`
}

func decodeLimitedJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxChunkJSONBody)
	return json.NewDecoder(limited).Decode(dst)
}

func (h *APIHandler) HandleChunkInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}
	if h.chunkService == nil {
		h.SendError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Chunked upload is not available")
		return
	}

	var req initChunkRequest
	if err := decodeLimitedJSON(r, &req); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_JSON", "Unable to parse request body")
		return
	}
	req.Filename = strings.TrimSpace(req.Filename)
	if req.Filename == "" || req.TotalSize <= 0 {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "filename and total_size are required")
		return
	}
	if len(req.Filename) > 255 {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "filename is too long")
		return
	}

	session, err := h.chunkService.Init(req.Filename, req.TotalSize)
	if err != nil {
		h.handleChunkError(w, err)
		return
	}

	h.SendSuccess(w, initChunkResponse{
		UploadID:    session.ID,
		ChunkSize:   session.ChunkSize,
		TotalChunks: session.TotalChunks,
		TotalSize:   session.TotalSize,
		Filename:    session.Filename,
	})
}

func (h *APIHandler) HandleChunkUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}
	if h.chunkService == nil {
		h.SendError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Chunked upload is not available")
		return
	}

	// Cap multipart memory/parse slightly above chunk limit
	if err := r.ParseMultipartForm(h.maxUploadSize + (2 << 20)); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_FORM", "Unable to parse multipart form")
		return
	}

	uploadID := strings.TrimSpace(r.FormValue("upload_id"))
	indexStr := strings.TrimSpace(r.FormValue("index"))
	if uploadID == "" || indexStr == "" {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "upload_id and index are required")
		return
	}
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid chunk index")
		return
	}

	file, header, err := r.FormFile("chunk")
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		h.SendError(w, http.StatusBadRequest, "FILE_NOT_FOUND", "chunk file is required")
		return
	}
	defer file.Close()

	if err := h.chunkService.SaveChunk(uploadID, index, file, header.Size); err != nil {
		h.handleChunkError(w, err)
		return
	}

	h.SendSuccess(w, map[string]interface{}{
		"upload_id": uploadID,
		"index":     index,
		"size":      header.Size,
		"status":    "received",
	})
}

func (h *APIHandler) HandleChunkComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}
	if h.chunkService == nil {
		h.SendError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Chunked upload is not available")
		return
	}

	var req struct {
		UploadID string `json:"upload_id"`
	}
	if err := decodeLimitedJSON(r, &req); err != nil || strings.TrimSpace(req.UploadID) == "" {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "upload_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	start := time.Now()
	result, size, filename, err := h.chunkService.Complete(ctx, strings.TrimSpace(req.UploadID))
	metrics.GetMetrics().RecordUpload(err == nil, time.Since(start))
	if err != nil {
		h.handleChunkError(w, err)
		return
	}

	log.Printf("[API] Chunked upload complete: key=%s size=%s", result.Key, utils.FormatFileSize(size))
	h.SendSuccess(w, UploadResponseData{
		URL:  result.URL,
		Key:  result.Key,
		Size: size,
		Name: filename,
	})
}

func (h *APIHandler) HandleChunkAbort(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}
	if h.chunkService == nil {
		h.SendError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Chunked upload is not available")
		return
	}

	var req struct {
		UploadID string `json:"upload_id"`
	}
	if err := decodeLimitedJSON(r, &req); err != nil || strings.TrimSpace(req.UploadID) == "" {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "upload_id is required")
		return
	}

	h.chunkService.Abort(strings.TrimSpace(req.UploadID))
	h.SendSuccess(w, map[string]string{"status": "aborted"})
}

func (h *APIHandler) handleChunkError(w http.ResponseWriter, err error) {
	var fileSizeErr *service.FileSizeError
	if errors.As(err, &fileSizeErr) {
		h.SendError(w, http.StatusBadRequest, "FILE_TOO_LARGE", fileSizeErr.Error())
		return
	}
	if errors.Is(err, service.ErrInvalidFileFormat) {
		h.SendError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", "File type not supported.")
		return
	}
	if errors.Is(err, service.ErrChunkSessionNotFound) {
		h.SendError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Upload session not found or expired")
		return
	}
	if errors.Is(err, service.ErrChunkSessionBusy) {
		h.SendError(w, http.StatusConflict, "SESSION_BUSY", "Upload session is already completing")
		return
	}
	if errors.Is(err, service.ErrTooManySessions) {
		h.SendError(w, http.StatusTooManyRequests, "TOO_MANY_SESSIONS", "Too many concurrent uploads. Try again shortly.")
		return
	}
	if errors.Is(err, service.ErrUploadToS3) {
		log.Printf("[API] Chunk S3 error: %v", err)
		h.SendError(w, http.StatusInternalServerError, "UPLOAD_FAILED", "Unable to upload file. Please try again later.")
		return
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	status := http.StatusBadRequest
	code := "CHUNK_ERROR"
	clientMsg := msg
	// Avoid leaking filesystem / internal details
	if strings.Contains(lower, "failed to") || strings.Contains(lower, "open chunk") || strings.Contains(lower, "assemble") {
		clientMsg = "Unable to process upload chunks. Please retry."
		if strings.Contains(lower, "missing chunk") || strings.Contains(lower, "mismatch") || strings.Contains(lower, "incomplete") {
			clientMsg = msg
		} else {
			status = http.StatusInternalServerError
			code = "UPLOAD_FAILED"
		}
	}
	log.Printf("[API] Chunk error: %v", err)
	h.SendError(w, status, code, clientMsg)
}

package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"s3-upload-tool/internal/service"
	"s3-upload-tool/internal/utils"
)

type WPHandler struct {
	BaseHandler
	imageResizeService *service.ImageResizeService
	maxUploadSize      int64
	tempUploadDir      string
}

func NewWPHandler(imageResizeService *service.ImageResizeService, maxUploadSize int64, tempUploadDir string) *WPHandler {
	return &WPHandler{
		imageResizeService: imageResizeService,
		maxUploadSize:      maxUploadSize,
		tempUploadDir:      tempUploadDir,
	}
}

// WPUploadResponseData response upload WordPress
type WPUploadResponseData struct {
	File  WPFileInfo   `json:"file"`
	Sizes []WPSizeInfo `json:"sizes"`
}

// WPFileInfo thông tin file
type WPFileInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

// WPSizeInfo thông tin kích thước ảnh đã resize
type WPSizeInfo struct {
	Name   string `json:"name"`
	File   string `json:"file"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

func (h *WPHandler) HandleWPUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("[WP] Nhận request upload từ IP: %s, Method: %s", r.RemoteAddr, r.Method)

	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	log.Printf("[WP] Đang parse multipart form với max size: %s", utils.FormatFileSize(h.maxUploadSize))
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		log.Printf("[WP] Lỗi parse multipart form: %v", err)
		h.SendError(w, http.StatusBadRequest, "INVALID_FORM", "Unable to parse multipart form. Please try again.")
		return
	}

	file, header, err := utils.GetFileFromRequest(r)
	if err != nil {
		log.Printf("[WP] Error reading file from form: %v", err)
		h.SendError(w, http.StatusBadRequest, "FILE_NOT_FOUND", err.Error())
		return
	}
	defer file.Close()

	log.Printf("[WP] File received - Name: %s, Size: %d bytes (%s)",
		header.Filename, header.Size, utils.FormatFileSize(header.Size))

	if err := utils.ValidateUploadFile(file, header, h.maxUploadSize); err != nil {
		log.Printf("[WP] File validation failed: %v", err)
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// * WordPress chỉ hỗ trợ upload ảnh
	if !utils.IsImage(header.Filename) {
		log.Printf("[WP] File is not an image: %s", header.Filename)
		h.SendError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", "Only image files are supported for WordPress upload")
		return
	}

	tempFile, err := os.CreateTemp(h.tempUploadDir, "wp-upload-*"+filepath.Ext(header.Filename))
	if err != nil {
		log.Printf("[WP] Lỗi tạo file tạm: %v", err)
		h.SendError(w, http.StatusInternalServerError, "TEMP_FILE_ERROR", "Unable to create temporary file")
		return
	}
	tempFileName := tempFile.Name()

	if _, err := file.Seek(0, 0); err != nil {
		tempFile.Close()
		os.Remove(tempFileName)
		log.Printf("[WP] Failed to seek file: %v", err)
		h.SendError(w, http.StatusInternalServerError, "FILE_READ_ERROR", fmt.Sprintf("Unable to read file: %v", err))
		return
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		os.Remove(tempFileName)
		log.Printf("[WP] Failed to copy file: %v", err)
		h.SendError(w, http.StatusInternalServerError, "FILE_SAVE_ERROR", fmt.Sprintf("Unable to save file: %v", err))
		return
	}

	if err := tempFile.Close(); err != nil {
		log.Printf("[WP] Warning: Failed to close temp file: %v", err)
	}

	defer func() {
		if removeErr := os.Remove(tempFileName); removeErr != nil {
			log.Printf("[WP] Cảnh báo: Không thể xóa file tạm %s: %v", tempFileName, removeErr)
		}
	}()

	originalPath, originalURL, err := h.imageResizeService.SaveOriginal(ctx, tempFileName, header.Filename)
	if err != nil {
		log.Printf("[WP] Failed to save original: %v", err)
		h.SendError(w, http.StatusInternalServerError, "SAVE_ERROR", "Unable to save original file")
		return
	}

	sizes, err := h.imageResizeService.ResizeImage(ctx, originalPath, header.Filename)
	if err != nil {
		log.Printf("[WP] Failed to resize images: %v", err)
		// ! Tiếp tục dù resize fail, trả về ảnh gốc
	}

	responseData := WPUploadResponseData{
		File: WPFileInfo{
			Name: header.Filename,
			Type: utils.GetContentType(header.Filename),
			URL:  originalURL,
			Size: header.Size,
		},
		Sizes: make([]WPSizeInfo, 0),
	}

	for _, size := range sizes {
		responseData.Sizes = append(responseData.Sizes, WPSizeInfo{
			Name:   size.Name,
			File:   filepath.Base(size.Path),
			Width:  size.Width,
			Height: size.Height,
			URL:    size.URL,
		})
	}

	log.Printf("[WP] Upload thành công! Original: %s, Sizes: %d", originalURL, len(sizes))
	h.SendSuccess(w, responseData)
}

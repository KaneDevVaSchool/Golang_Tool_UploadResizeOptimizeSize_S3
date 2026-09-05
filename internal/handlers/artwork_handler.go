package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/service"
	"s3-upload-tool/internal/utils"
)

// ArtworkHandler quản lý tác phẩm - bulk-upload (chỉ đẩy S3) + CRUD record
// (2 bước riêng biệt, xem ArtworkService.CreateArtworkFromUpload).
type ArtworkHandler struct {
	BaseHandler
	service       service.ArtworkService
	maxUploadSize int64
}

func NewArtworkHandler(svc service.ArtworkService, maxUploadSize int64) *ArtworkHandler {
	return &ArtworkHandler{service: svc, maxUploadSize: maxUploadSize}
}

// HandleBulkUpload POST /api/v1/admin/artworks/bulk-upload - nhận field
// multipart "files" (nhiều file), upload song song lên S3 (giới hạn
// concurrency trong ArtworkService), KHÔNG ghi bảng artworks. Mỗi file dùng
// tên gốc (không kèm index) làm temp_key để FE tự khớp lại - nếu 2 file
// trùng tên, FE nên tự thêm hậu tố khi hiển thị vì response giữ nguyên thứ
// tự mảng input.
func (h *ArtworkHandler) HandleBulkUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}

	// Cho phép tổng dung lượng form lớn hơn maxUploadSize/file vì đây là
	// nhiều file cùng lúc - nhân 20 (đủ cho ~20 ảnh/lượt, khớp bulkConcurrency
	// mặc định trong ArtworkService) làm giới hạn tổng, mỗi file vẫn bị chặn
	// riêng ở maxUploadSize khi UploadService.UploadImage validate.
	if err := r.ParseMultipartForm(h.maxUploadSize * 20); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_FORM", "Không đọc được form dữ liệu")
		return
	}

	fileHeaders := r.MultipartForm.File["files"]
	if len(fileHeaders) == 0 {
		h.SendError(w, http.StatusBadRequest, "NO_FILES", "Chưa chọn ảnh nào để tải lên")
		return
	}

	files := make([]service.BulkUploadFile, 0, len(fileHeaders))
	var openErrors []string
	for _, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			openErrors = append(openErrors, fh.Filename)
			continue
		}
		defer f.Close()

		sanitized, err := utils.SanitizeFilename(fh.Filename)
		if err != nil || sanitized == "" {
			openErrors = append(openErrors, fh.Filename)
			continue
		}

		if err := utils.ValidateUploadFile(f, fh, h.maxUploadSize); err != nil {
			openErrors = append(openErrors, fh.Filename)
			continue
		}

		// utils.ValidateFileContent đã seek(0) file gốc, nhưng ta cần
		// io.ReadSeeker để decode dimension rồi upload - multipart.File đã
		// implement io.ReadSeeker sẵn nên dùng trực tiếp, không cần buffer riêng.
		files = append(files, service.BulkUploadFile{
			TempKey:  sanitized,
			FileName: sanitized,
			Reader:   f,
			FileSize: fh.Size,
		})
	}

	if len(files) == 0 {
		h.SendError(w, http.StatusBadRequest, "ALL_FILES_INVALID", "Không có ảnh hợp lệ để tải lên")
		return
	}

	results := h.service.BulkUploadToS3(r.Context(), files, h.maxUploadSize)

	successCount := 0
	for _, item := range results {
		if item.Error == "" {
			successCount++
		}
	}
	log.Printf("[ArtworkHandler] Bulk upload: %d/%d thành công", successCount, len(results))

	h.SendSuccess(w, map[string]any{
		"items":       results,
		"open_errors": openErrors,
	})
}

type createArtworkBody struct {
	Title        string `json:"title"`
	StudentName  string `json:"student_name"`
	SchoolID     int64  `json:"school_id"`
	GradeLevelID int64  `json:"grade_level_id"`
	ClassName    string `json:"class_name"`
	S3Key        string `json:"s3_key"`
	S3URL        string `json:"s3_url"`
	FileSize     int64  `json:"file_size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	// Variants do bulk-upload trả về ở bước 1, FE gửi lại nguyên vẹn. Không
	// bắt buộc: ảnh gốc nhỏ hoặc khâu sinh biến thể lỗi thì trường này rỗng
	// và trang vẫn chạy bằng ảnh gốc.
	Variants models.ArtworkVariants `json:"variants"`
	AwardID  *int64                 `json:"award_id"`
}

// HandleCreate POST /api/v1/admin/artworks - bước 2 sau bulk-upload: nhận
// metadata + s3_key/s3_url đã có từ bulk-upload, tạo student + artwork record.
func (h *ArtworkHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}

	var body createArtworkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	if err := validateCreateArtworkBody(body); err != nil {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	var createdBy *int64
	if admin := middleware.GetAdminUser(r.Context()); admin != nil {
		createdBy = &admin.ID
	}

	artwork, err := h.service.CreateArtworkFromUpload(r.Context(), service.CreateArtworkRequest{
		Title:        body.Title,
		StudentName:  body.StudentName,
		SchoolID:     body.SchoolID,
		GradeLevelID: body.GradeLevelID,
		ClassName:    body.ClassName,
		S3Key:        body.S3Key,
		S3URL:        body.S3URL,
		FileSize:     body.FileSize,
		Width:        body.Width,
		Height:       body.Height,
		Variants:     body.Variants,
		AwardID:      body.AwardID,
		CreatedBy:    createdBy,
	})
	if err != nil {
		log.Printf("[ArtworkHandler] Tạo tác phẩm thất bại: %v", err)
		h.SendError(w, http.StatusInternalServerError, "CREATE_FAILED", "Không tạo được tác phẩm")
		return
	}

	h.SendSuccess(w, artwork)
}

func validateCreateArtworkBody(b createArtworkBody) error {
	if b.Title == "" {
		return fmt.Errorf("tên tác phẩm không được để trống")
	}
	if b.StudentName == "" {
		return fmt.Errorf("tên học sinh sáng tác không được để trống")
	}
	if b.SchoolID == 0 {
		return fmt.Errorf("chưa chọn cơ sở trực thuộc")
	}
	if b.GradeLevelID == 0 {
		return fmt.Errorf("chưa chọn khối lớp")
	}
	if b.S3Key == "" || b.S3URL == "" {
		return fmt.Errorf("thiếu thông tin ảnh đã tải lên - vui lòng upload lại")
	}
	return nil
}

type updateArtworkBody struct {
	Title        string `json:"title"`
	StudentID    int64  `json:"student_id"`
	SchoolID     int64  `json:"school_id"`
	GradeLevelID int64  `json:"grade_level_id"`
	IsFeatured   bool   `json:"is_featured"`
	IsPublished  bool   `json:"is_published"`
	AwardID      *int64 `json:"award_id"`
}

// HandleUpdate PUT /api/v1/admin/artworks/{id}
func (h *ArtworkHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	var body updateArtworkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if body.Title == "" {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Tên tác phẩm không được để trống")
		return
	}

	artwork, err := h.service.UpdateArtwork(r.Context(), id, service.UpdateArtworkRequest{
		Title:        body.Title,
		StudentID:    body.StudentID,
		SchoolID:     body.SchoolID,
		GradeLevelID: body.GradeLevelID,
		IsFeatured:   body.IsFeatured,
		IsPublished:  body.IsPublished,
		AwardID:      body.AwardID,
	})
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Không cập nhật được tác phẩm")
		return
	}

	h.SendSuccess(w, artwork)
}

// HandleDelete DELETE /api/v1/admin/artworks/{id}
func (h *ArtworkHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	if err := h.service.DeleteArtwork(r.Context(), id); err != nil {
		h.SendError(w, http.StatusInternalServerError, "DELETE_FAILED", "Không xoá được tác phẩm")
		return
	}
	h.SendSuccess(w, map[string]bool{"deleted": true})
}

// HandleGet GET /api/v1/admin/artworks/{id}
func (h *ArtworkHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	artwork, err := h.service.GetArtwork(r.Context(), id)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được thông tin tác phẩm")
		return
	}
	if artwork == nil {
		h.SendError(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy tác phẩm")
		return
	}
	h.SendSuccess(w, artwork)
}

// HandleList GET /api/v1/admin/artworks?search=&school_id=&grade_level_id=&award_id=&featured=&page=&page_size=
func (h *ArtworkHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	filter := parseArtworkFilter(r)

	result, err := h.service.ListArtworks(r.Context(), filter)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách tác phẩm")
		return
	}

	h.SendSuccess(w, map[string]any{
		"items":       result.Items,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"page_size":   result.PageSize,
	})
}

// HandleSetFeatured PATCH /api/v1/admin/artworks/{id}/featured
func (h *ArtworkHandler) HandleSetFeatured(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ PATCH")
		return
	}
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	var body struct {
		Featured bool `json:"featured"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	if err := h.service.SetFeatured(r.Context(), id, body.Featured); err != nil {
		h.SendError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Không cập nhật được trạng thái tiêu biểu")
		return
	}
	h.SendSuccess(w, map[string]bool{"is_featured": body.Featured})
}

func parseArtworkFilter(r *http.Request) models.ArtworkFilter {
	q := r.URL.Query()
	filter := models.ArtworkFilter{
		Search:         q.Get("search"),
		EducationLevel: q.Get("education_level"),
		Page:           parseIntOrDefault(q.Get("page"), 1),
		PageSize:       parseIntOrDefault(q.Get("page_size"), 20),
	}

	if v, err := strconv.ParseInt(q.Get("school_id"), 10, 64); err == nil && v > 0 {
		filter.SchoolID = &v
	}
	if v, err := strconv.ParseInt(q.Get("grade_level_id"), 10, 64); err == nil && v > 0 {
		filter.GradeLevelID = &v
	}
	if v, err := strconv.ParseInt(q.Get("award_id"), 10, 64); err == nil && v > 0 {
		filter.AwardID = &v
	}
	if v := q.Get("featured"); v != "" {
		b := v == "true" || v == "1"
		filter.IsFeatured = &b
	}
	if v := q.Get("published"); v != "" {
		b := v == "true" || v == "1"
		filter.IsPublished = &b
	}

	return filter
}

func parseIntOrDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

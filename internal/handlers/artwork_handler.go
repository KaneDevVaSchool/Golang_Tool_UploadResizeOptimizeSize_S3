package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"
	"s3-upload-tool/internal/utils"
)

// ArtworkHandler quản lý tác phẩm - bulk-upload (chỉ đẩy S3) + CRUD record
// (2 bước riêng biệt, xem ArtworkService.CreateArtworkFromUpload).
type ArtworkHandler struct {
	BaseHandler
	service       service.ArtworkService
	maxUploadSize int64
	// s3Repo/s3Bucket chỉ phục vụ HandleDownload (tải ảnh gốc từ khu quản
	// trị) - cùng cơ chế PublicHandler.HandleDownloadArtwork nhưng KHÔNG
	// watermark (admin cần file gốc sạch để lưu trữ/in ấn) và KHÔNG ép
	// is_published (admin phải tải được cả ảnh đang ẩn).
	s3Repo   repository.S3Repository
	s3Bucket string
}

func NewArtworkHandler(svc service.ArtworkService, maxUploadSize int64, s3Repo repository.S3Repository, s3Bucket string) *ArtworkHandler {
	return &ArtworkHandler{service: svc, maxUploadSize: maxUploadSize, s3Repo: s3Repo, s3Bucket: s3Bucket}
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
	Title           string `json:"title"`
	StudentName     string `json:"student_name"`
	SchoolID        int64  `json:"school_id"`
	GradeLevelID    int64  `json:"grade_level_id"`
	TopicCategoryID *int64 `json:"topic_category_id"`
	ClassName       string `json:"class_name"`
	S3Key           string `json:"s3_key"`
	S3URL           string `json:"s3_url"`
	FileSize        int64  `json:"file_size"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	// Variants do bulk-upload trả về ở bước 1, FE gửi lại nguyên vẹn. Không
	// bắt buộc: ảnh gốc nhỏ hoặc khâu sinh biến thể lỗi thì trường này rỗng
	// và trang vẫn chạy bằng ảnh gốc.
	Variants models.ArtworkVariants `json:"variants"`
	// AwardIDs cho phép gán nhiều giải ngay lúc tạo (vd giải chính + giải Đặc
	// biệt phụ). Rỗng hoặc thiếu trường = chưa gán giải nào.
	AwardIDs []int64 `json:"award_ids"`
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
		Title:           body.Title,
		StudentName:     body.StudentName,
		SchoolID:        body.SchoolID,
		GradeLevelID:    body.GradeLevelID,
		TopicCategoryID: body.TopicCategoryID,
		ClassName:       body.ClassName,
		S3Key:           body.S3Key,
		S3URL:           body.S3URL,
		FileSize:        body.FileSize,
		Width:           body.Width,
		Height:          body.Height,
		Variants:        body.Variants,
		AwardIDs:        body.AwardIDs,
		CreatedBy:       createdBy,
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
	Title           string `json:"title"`
	StudentID       int64  `json:"student_id"`
	SchoolID        int64  `json:"school_id"`
	GradeLevelID    int64  `json:"grade_level_id"`
	TopicCategoryID *int64 `json:"topic_category_id"`
	IsFeatured      bool   `json:"is_featured"`
	IsPublished     bool   `json:"is_published"`
	// AwardIDs: trường không có trong JSON (client không gửi key) = nil =
	// giữ nguyên giải hiện tại; gửi [] = gỡ hết giải; gửi danh sách = thay
	// toàn bộ giải hiện tại bằng danh sách này. Dùng con trỏ để phân biệt
	// "không gửi" với "gửi mảng rỗng" - json.Unmarshal để AwardIDs là nil
	// nếu key vắng mặt, khác []int64{} nếu client gửi mảng rỗng tường minh.
	AwardIDs []int64 `json:"award_ids"`
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
		Title:           body.Title,
		StudentID:       body.StudentID,
		SchoolID:        body.SchoolID,
		GradeLevelID:    body.GradeLevelID,
		TopicCategoryID: body.TopicCategoryID,
		IsFeatured:      body.IsFeatured,
		IsPublished:     body.IsPublished,
		AwardIDs:        body.AwardIDs,
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

// HandleDeleteBatch DELETE /api/v1/admin/artworks/bulk-delete - xoá nhiều tác
// phẩm cùng lúc (thao tác bulk ở trang danh sách). Dùng method DELETE với
// thân JSON {ids} thay vì query string vì số lượng id có thể lớn.
func (h *ArtworkHandler) HandleDeleteBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ DELETE")
		return
	}

	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if len(body.IDs) == 0 {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Chưa chọn tác phẩm nào")
		return
	}

	deleted, err := h.service.DeleteArtworkBatch(r.Context(), body.IDs)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "DELETE_FAILED", "Không xoá được tác phẩm")
		return
	}
	h.SendSuccess(w, map[string]any{"deleted": deleted, "requested": len(body.IDs)})
}

// HandleDownload GET /api/v1/admin/artworks/{id}/download - stream ảnh gốc
// (không watermark, không ép is_published) để admin tải về lưu trữ/in ấn.
// Proxy qua backend thay vì link S3 trực tiếp, để trình duyệt tải
// same-origin và không cần CORS trên bucket. Đây là công cụ quản trị nội bộ -
// không có phiên bản công khai cho khách xem ẩn danh (đã gỡ có chủ đích, xem
// docs/plan/03-risks.md).
func (h *ArtworkHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	artwork, err := h.service.GetArtwork(r.Context(), id)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được tác phẩm")
		return
	}
	if artwork == nil {
		h.SendError(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy tác phẩm")
		return
	}
	if strings.TrimSpace(artwork.S3Key) == "" {
		h.SendError(w, http.StatusNotFound, "NOT_FOUND", "Không có file ảnh để tải")
		return
	}
	if h.s3Repo == nil || strings.TrimSpace(h.s3Bucket) == "" {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không cấu hình được tải ảnh")
		return
	}

	body, contentType, _, err := h.s3Repo.GetObject(r.Context(), h.s3Bucket, artwork.S3Key)
	if err != nil {
		h.SendError(w, http.StatusBadGateway, "STORAGE_ERROR", "Không lấy được ảnh từ kho lưu trữ")
		return
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		h.SendError(w, http.StatusBadGateway, "STORAGE_ERROR", "Không đọc được ảnh từ kho lưu trữ")
		return
	}

	ext := strings.ToLower(filepath.Ext(artwork.S3Key))
	if ext == "" {
		ext = ".jpg"
	}
	downloadName := artworkDownloadFileName(artwork.Title, ext)

	if contentType == "" || contentType == "application/octet-stream" {
		if typed := utils.GetContentType(downloadName); typed != "application/octet-stream" {
			contentType = typed
		}
	}

	logArtworkDownload(r, h.service, id, models.ArtworkDownloadSourceAdmin)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": downloadName,
	}))
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Write(raw)
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

// HandleList GET /api/v1/admin/artworks?search=&region=&school_id=&grade_level_id=&award_id=&featured=&page=&page_size=
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

// HandleSetFeaturedBatch PATCH /api/v1/admin/artworks/bulk-featured - bật/tắt
// tiêu biểu cho nhiều tác phẩm cùng lúc (thao tác bulk trên trang danh sách).
func (h *ArtworkHandler) HandleSetFeaturedBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ PATCH")
		return
	}

	var body struct {
		IDs      []int64 `json:"ids"`
		Featured bool    `json:"featured"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if len(body.IDs) == 0 {
		h.SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Chưa chọn tác phẩm nào")
		return
	}

	if err := h.service.SetFeaturedBatch(r.Context(), body.IDs, body.Featured); err != nil {
		h.SendError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Không cập nhật được trạng thái tiêu biểu")
		return
	}
	h.SendSuccess(w, map[string]any{"updated": len(body.IDs), "is_featured": body.Featured})
}

// HandleListComments GET /api/v1/admin/artworks/{id}/comments - trả TOÀN BỘ
// bình luận (kể cả đã ẩn) để màn hình kiểm duyệt tự hiển thị trạng thái, khác
// endpoint public luôn lọc bỏ comment is_hidden=1.
func (h *ArtworkHandler) HandleListComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	comments, err := h.service.ListComments(r.Context(), artworkID)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được bình luận")
		return
	}
	h.SendSuccess(w, comments)
}

// HandleSetCommentHidden PATCH /api/v1/admin/artworks/{id}/comments/{commentID}
// - body {is_hidden: bool}. Ẩn khỏi trang public ngay vì PublicHandler luôn
// lọc includeHidden=false ở mọi lượt gọi tiếp theo.
func (h *ArtworkHandler) HandleSetCommentHidden(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ PATCH")
		return
	}
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}
	commentID, ok := parsePathID(r, "commentID")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID bình luận không hợp lệ")
		return
	}

	var body struct {
		IsHidden bool `json:"is_hidden"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	if err := h.service.SetCommentHidden(r.Context(), artworkID, commentID, body.IsHidden); err != nil {
		h.SendError(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy bình luận")
		return
	}
	h.SendSuccess(w, map[string]bool{"is_hidden": body.IsHidden})
}

func parseArtworkFilter(r *http.Request) models.ArtworkFilter {
	q := r.URL.Query()
	filter := models.ArtworkFilter{
		Search:         q.Get("search"),
		EducationLevel: q.Get("education_level"),
		Page:           parseIntOrDefault(q.Get("page"), 1),
		PageSize:       parseIntOrDefault(q.Get("page_size"), 20),
	}

	if v := strings.TrimSpace(q.Get("region")); models.IsKnownRegion(v) {
		filter.Region = &v
	}
	if v, err := strconv.ParseInt(q.Get("school_id"), 10, 64); err == nil && v > 0 {
		filter.SchoolID = &v
	}
	if v, err := strconv.ParseInt(q.Get("grade_level_id"), 10, 64); err == nil && v > 0 {
		filter.GradeLevelID = &v
	}
	if v, err := strconv.ParseInt(q.Get("topic_category_id"), 10, 64); err == nil && v > 0 {
		filter.TopicCategoryID = &v
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

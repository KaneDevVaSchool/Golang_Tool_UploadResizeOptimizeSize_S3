package handlers

import (
	"net/http"

	"s3-upload-tool/internal/repository"
)

// MetaHandler expose danh sách "dữ liệu nền" gần như tĩnh (trường, khối
// lớp) - dùng chung cho cả form admin (chọn trường/khối khi tạo tác phẩm)
// và bộ lọc trang public, nên mount public (không cần session/API key).
type MetaHandler struct {
	BaseHandler
	schoolRepo repository.SchoolRepository
	gradeRepo  repository.GradeLevelRepository
}

func NewMetaHandler(schoolRepo repository.SchoolRepository, gradeRepo repository.GradeLevelRepository) *MetaHandler {
	return &MetaHandler{schoolRepo: schoolRepo, gradeRepo: gradeRepo}
}

// HandleListSchools GET /api/v1/schools - danh sách 5 cơ sở.
func (h *MetaHandler) HandleListSchools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}
	schools, err := h.schoolRepo.List(r.Context())
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách trường")
		return
	}
	h.SendSuccess(w, schools)
}

// HandleListGradeLevels GET /api/v1/grade-levels - danh sách 12 khối lớp,
// hỗ trợ query param ?education_level=primary|secondary để lọc theo cấp học.
func (h *MetaHandler) HandleListGradeLevels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	educationLevel := r.URL.Query().Get("education_level")

	if educationLevel != "" {
		list, listErr := h.gradeRepo.ListByEducationLevel(r.Context(), educationLevel)
		if listErr != nil {
			h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách khối lớp")
			return
		}
		h.SendSuccess(w, list)
		return
	}

	list, err := h.gradeRepo.List(r.Context())
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách khối lớp")
		return
	}
	h.SendSuccess(w, list)
}

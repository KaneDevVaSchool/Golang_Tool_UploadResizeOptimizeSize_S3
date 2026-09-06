package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/service"
)

// TopicCategoryHandler: CRUD nhóm chủ đề sáng tạo cho admin
// (/api/v1/admin/topic-categories) + 1 endpoint đọc công khai
// (/api/v1/topic-categories) cho trang public lọc/hiển thị theo nhóm chủ đề.
// Cùng mẫu AwardHandler.
type TopicCategoryHandler struct {
	BaseHandler
	service service.TopicCategoryService
}

func NewTopicCategoryHandler(svc service.TopicCategoryService) *TopicCategoryHandler {
	return &TopicCategoryHandler{service: svc}
}

type topicCategoryRequestBody struct {
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	ColorHex       string  `json:"color_hex"`
	EducationLevel *string `json:"education_level"`
	DisplayOrder   int     `json:"display_order"`
	IsActive       *bool   `json:"is_active"`
}

// HandleListTopicCategories GET /api/v1/admin/topic-categories (includeInactive=true)
// hoặc GET /api/v1/topic-categories (public, chỉ nhóm đang active).
func (h *TopicCategoryHandler) HandleListTopicCategories(includeInactive bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
			return
		}
		categories, err := h.service.ListTopicCategories(r.Context(), includeInactive)
		if err != nil {
			h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách nhóm chủ đề")
			return
		}
		h.SendSuccess(w, categories)
	}
}

// HandleCreateTopicCategory POST /api/v1/admin/topic-categories
func (h *TopicCategoryHandler) HandleCreateTopicCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}

	var body topicCategoryRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	category := &models.TopicCategory{
		Name:           strings.TrimSpace(body.Name),
		Slug:           strings.TrimSpace(body.Slug),
		ColorHex:       strings.TrimSpace(body.ColorHex),
		EducationLevel: body.EducationLevel,
		DisplayOrder:   body.DisplayOrder,
		IsActive:       isActive,
	}

	created, err := h.service.CreateTopicCategory(r.Context(), category)
	if err != nil {
		h.SendError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	h.SendSuccess(w, created)
}

// HandleUpdateTopicCategory PUT /api/v1/admin/topic-categories/{id}
func (h *TopicCategoryHandler) HandleUpdateTopicCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID nhóm chủ đề không hợp lệ")
		return
	}

	var body topicCategoryRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	category := &models.TopicCategory{
		ID:             id,
		Name:           strings.TrimSpace(body.Name),
		Slug:           strings.TrimSpace(body.Slug),
		ColorHex:       strings.TrimSpace(body.ColorHex),
		EducationLevel: body.EducationLevel,
		DisplayOrder:   body.DisplayOrder,
		IsActive:       isActive,
	}

	if err := h.service.UpdateTopicCategory(r.Context(), category); err != nil {
		h.SendError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	h.SendSuccess(w, category)
}

// HandleDeleteTopicCategory DELETE /api/v1/admin/topic-categories/{id}. Trả
// 409 nếu nhóm đang được gắn cho tác phẩm nào đó (FK constraint chặn xoá).
func (h *TopicCategoryHandler) HandleDeleteTopicCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID nhóm chủ đề không hợp lệ")
		return
	}

	if err := h.service.DeleteTopicCategory(r.Context(), id); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
			h.SendError(w, http.StatusConflict, "TOPIC_CATEGORY_IN_USE", "Không thể xoá nhóm chủ đề đang được gắn cho tác phẩm")
			return
		}
		h.SendError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	h.SendSuccess(w, map[string]bool{"deleted": true})
}

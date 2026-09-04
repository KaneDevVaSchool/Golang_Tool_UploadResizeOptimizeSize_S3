package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/service"
)

// AwardHandler: CRUD giải thưởng cho admin (/api/v1/admin/awards) + 1
// endpoint đọc công khai (/api/v1/awards) cho trang public dùng đúng
// màu/icon cấu hình khi hiển thị billboard vinh danh.
type AwardHandler struct {
	BaseHandler
	service service.AwardService
}

func NewAwardHandler(svc service.AwardService) *AwardHandler {
	return &AwardHandler{service: svc}
}

type awardRequestBody struct {
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	RankOrder int     `json:"rank_order"`
	ColorHex  string  `json:"color_hex"`
	IconKey   *string `json:"icon_key"`
	IsActive  *bool   `json:"is_active"`
}

// HandleListAwards GET /api/v1/admin/awards (includeInactive=true) hoặc
// GET /api/v1/awards (public, chỉ awards đang active).
func (h *AwardHandler) HandleListAwards(includeInactive bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
			return
		}
		awards, err := h.service.ListAwards(r.Context(), includeInactive)
		if err != nil {
			h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách giải thưởng")
			return
		}
		h.SendSuccess(w, awards)
	}
}

// HandleCreateAward POST /api/v1/admin/awards
func (h *AwardHandler) HandleCreateAward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}

	var body awardRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	award := &models.Award{
		Name:      strings.TrimSpace(body.Name),
		Slug:      strings.TrimSpace(body.Slug),
		RankOrder: body.RankOrder,
		ColorHex:  body.ColorHex,
		IconKey:   body.IconKey,
		IsActive:  isActive,
	}

	created, err := h.service.CreateAward(r.Context(), award)
	if err != nil {
		h.SendError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	h.SendSuccess(w, created)
}

// HandleUpdateAward PUT /api/v1/admin/awards/{id}
func (h *AwardHandler) HandleUpdateAward(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID giải thưởng không hợp lệ")
		return
	}

	var body awardRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	award := &models.Award{
		ID:        id,
		Name:      strings.TrimSpace(body.Name),
		Slug:      strings.TrimSpace(body.Slug),
		RankOrder: body.RankOrder,
		ColorHex:  body.ColorHex,
		IconKey:   body.IconKey,
		IsActive:  isActive,
	}

	if err := h.service.UpdateAward(r.Context(), award); err != nil {
		h.SendError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	h.SendSuccess(w, award)
}

// HandleDeleteAward DELETE /api/v1/admin/awards/{id}. Trả 409 nếu giải
// đang được gắn cho tác phẩm nào đó (FK constraint artwork_awards chặn xoá).
func (h *AwardHandler) HandleDeleteAward(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID giải thưởng không hợp lệ")
		return
	}

	if err := h.service.DeleteAward(r.Context(), id); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
			h.SendError(w, http.StatusConflict, "AWARD_IN_USE", "Không thể xoá giải thưởng đang được gắn cho tác phẩm")
			return
		}
		h.SendError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	h.SendSuccess(w, map[string]bool{"deleted": true})
}

// parsePathID đọc path param {name} qua r.PathValue (Go 1.22+ ServeMux hỗ
// trợ pattern "{id}" trực tiếp, vd "PUT /api/v1/admin/awards/{id}" - không
// cần router thư viện ngoài hay tự parse chuỗi path).
func parsePathID(r *http.Request, name string) (int64, bool) {
	raw := r.PathValue(name)
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

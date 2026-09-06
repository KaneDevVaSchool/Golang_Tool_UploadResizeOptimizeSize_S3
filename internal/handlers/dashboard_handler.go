package handlers

import (
	"log"
	"net/http"

	"s3-upload-tool/internal/service"
)

// DashboardHandler expose số liệu tổng hợp cho trang Dashboard admin.
type DashboardHandler struct {
	BaseHandler
	service service.DashboardService
}

func NewDashboardHandler(svc service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: svc}
}

// HandleStats GET /api/v1/admin/dashboard/stats
func (h *DashboardHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		// Ghi log nguyên nhân thật: thông báo trả về cho người dùng cố tình chung
		// chung, nên nếu không log ở đây thì lỗi 500 mất dấu hoàn toàn.
		log.Printf("[Dashboard] Không lấy được số liệu thống kê: %v", err)
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được số liệu thống kê")
		return
	}
	h.SendSuccess(w, stats)
}

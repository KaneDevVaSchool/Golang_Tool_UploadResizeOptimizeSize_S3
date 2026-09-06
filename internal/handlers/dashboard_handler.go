package handlers

import (
	"log"
	"net/http"
	"time"

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

// HandleStats GET /api/v1/admin/dashboard/stats?from=YYYY-MM-DD&to=YYYY-MM-DD
//
// from/to đều tuỳ chọn - thiếu một trong hai (hoặc cả hai, hoặc giá trị
// không đọc được) thì service tự áp mặc định 14 ngày gần nhất. Đây là tham
// số hiển thị cho biểu đồ nhịp hoạt động (chọn theo tháng hoặc theo khoảng
// ngày ở FE), không ảnh hưởng các khối số liệu còn lại trong response.
func (h *DashboardHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	from, _ := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	to, _ := time.Parse("2006-01-02", r.URL.Query().Get("to"))

	stats, err := h.service.GetStats(r.Context(), from, to)
	if err != nil {
		// Ghi log nguyên nhân thật: thông báo trả về cho người dùng cố tình chung
		// chung, nên nếu không log ở đây thì lỗi 500 mất dấu hoàn toàn.
		log.Printf("[Dashboard] Không lấy được số liệu thống kê: %v", err)
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được số liệu thống kê")
		return
	}
	h.SendSuccess(w, stats)
}

// HandleRegionSummary GET /api/v1/admin/dashboard/region-summary
func (h *DashboardHandler) HandleRegionSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	summary, err := h.service.GetRegionSummary(r.Context())
	if err != nil {
		log.Printf("[Dashboard] Không lấy được số liệu theo khu vực: %v", err)
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được số liệu theo khu vực")
		return
	}
	h.SendSuccess(w, summary)
}

package handlers

import (
	"net/http"

	"s3-upload-tool/internal/metrics"
)

// MetricsHandler handles metrics endpoint
type MetricsHandler struct {
	BaseHandler
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// HandleMetrics returns current metrics
func (h *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET method is allowed")
		return
	}

	stats := metrics.GetMetrics().GetStats()
	h.SendSuccess(w, stats)
}

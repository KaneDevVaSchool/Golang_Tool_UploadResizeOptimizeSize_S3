package metrics

import (
	"net/http"
	"sync"
	"time"
)

// Metrics collects basic application metrics
type Metrics struct {
	mu sync.RWMutex

	// Request metrics
	requestCount    map[string]int64           // endpoint -> count
	requestDuration map[string][]time.Duration // endpoint -> durations
	errorCount      map[string]int64           // endpoint -> error count
	maxEndpoints    int                        // Maximum number of endpoints to track (prevent unbounded growth)

	// Upload metrics
	uploadCount    int64
	uploadSuccess  int64
	uploadFailed   int64
	uploadDuration []time.Duration

	// Image processing metrics
	imageResizeCount   int64
	imageOptimizeCount int64
	imageOptimizeSaved int64 // bytes saved
}

var (
	globalMetrics *Metrics
	once          sync.Once
)

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	once.Do(func() {
		globalMetrics = &Metrics{
			requestCount:    make(map[string]int64),
			requestDuration: make(map[string][]time.Duration),
			errorCount:      make(map[string]int64),
			maxEndpoints:    1000, // Limit to 1000 unique endpoints to prevent unbounded growth
		}
	})
	return globalMetrics
}

// RecordRequest records a request
func (m *Metrics) RecordRequest(endpoint string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Limit number of tracked endpoints to prevent unbounded growth
	// If we exceed maxEndpoints, stop tracking new endpoints (but continue tracking existing ones)
	if len(m.requestCount) >= m.maxEndpoints {
		if _, exists := m.requestCount[endpoint]; !exists {
			// Don't track new endpoints if we've reached the limit
			return
		}
	}

	m.requestCount[endpoint]++
	if m.requestDuration[endpoint] == nil {
		m.requestDuration[endpoint] = make([]time.Duration, 0, 100)
	}

	// Create a copy of the slice to avoid race conditions
	// Slice append can cause data races if multiple goroutines modify concurrently
	durations := make([]time.Duration, len(m.requestDuration[endpoint]))
	copy(durations, m.requestDuration[endpoint])

	if len(durations) < 100 {
		durations = append(durations, duration)
	} else {
		// Keep only last 100 - remove oldest, add newest
		durations = append(durations[1:], duration)
	}

	// Assign the new slice back (atomic operation under lock)
	m.requestDuration[endpoint] = durations
}

// RecordError records an error for an endpoint
func (m *Metrics) RecordError(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount[endpoint]++
}

// RecordUpload records an upload operation
func (m *Metrics) RecordUpload(success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.uploadCount++
	if success {
		m.uploadSuccess++
	} else {
		m.uploadFailed++
	}

	// Create a copy to avoid race conditions
	durations := make([]time.Duration, len(m.uploadDuration))
	copy(durations, m.uploadDuration)

	if len(durations) < 100 {
		durations = append(durations, duration)
	} else {
		// Keep only last 100
		durations = append(durations[1:], duration)
	}

	m.uploadDuration = durations
}

// RecordImageResize records an image resize operation
func (m *Metrics) RecordImageResize() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.imageResizeCount++
}

// RecordImageOptimize records an image optimization operation
func (m *Metrics) RecordImageOptimize(savedBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.imageOptimizeCount++
	m.imageOptimizeSaved += savedBytes
}

// GetStats returns current metrics snapshot
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// Request stats
	requestStats := make(map[string]interface{})
	for endpoint, count := range m.requestCount {
		requestStats[endpoint] = map[string]interface{}{
			"count":       count,
			"error_count": m.errorCount[endpoint],
		}

		// Calculate average duration
		if durations, ok := m.requestDuration[endpoint]; ok && len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avgDuration := total / time.Duration(len(durations))
			// Safe type assertion to prevent panic
			if endpointStats, ok := requestStats[endpoint].(map[string]interface{}); ok {
				endpointStats["avg_duration_ms"] = avgDuration.Milliseconds()
			}
		}
	}
	stats["requests"] = requestStats

	// Upload stats
	uploadStats := map[string]interface{}{
		"total":   m.uploadCount,
		"success": m.uploadSuccess,
		"failed":  m.uploadFailed,
	}
	if len(m.uploadDuration) > 0 {
		var total time.Duration
		for _, d := range m.uploadDuration {
			total += d
		}
		avgDuration := total / time.Duration(len(m.uploadDuration))
		uploadStats["avg_duration_ms"] = avgDuration.Milliseconds()
	}
	stats["uploads"] = uploadStats

	// Image processing stats
	stats["image_processing"] = map[string]interface{}{
		"resize_count":         m.imageResizeCount,
		"optimize_count":       m.imageOptimizeCount,
		"optimize_saved_bytes": m.imageOptimizeSaved,
	}

	return stats
}

// MetricsMiddleware records request metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		endpoint := r.URL.Path

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		metrics := GetMetrics()
		metrics.RecordRequest(endpoint, duration)

		if wrapped.statusCode >= 400 {
			metrics.RecordError(endpoint)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

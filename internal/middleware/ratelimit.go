package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implement token bucket algorithm cho rate limiting
type RateLimiter struct {
	mu          sync.RWMutex
	visitors    map[string]*visitor
	rate        int           // số requests cho phép trong window
	window      time.Duration // thời gian window
	cleanupTick time.Duration
	lastCleanup time.Time
	ctx         context.Context
	cancel      context.CancelFunc
}

type visitor struct {
	lastSeen time.Time
	tokens   int
	mu       sync.Mutex
}

// NewRateLimiter tạo rate limiter mới
func NewRateLimiter(rate int, window, cleanupInterval time.Duration) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		visitors:    make(map[string]*visitor),
		rate:        rate,
		window:      window,
		cleanupTick: cleanupInterval,
		lastCleanup: time.Now(),
		ctx:         ctx,
		cancel:      cancel,
	}

	go rl.cleanup()

	return rl
}

// cleanup xóa old visitors để tránh memory leaks
// * Tối ưu cho high concurrency: batch deletion để giảm lock contention
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-rl.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			cutoff := rl.window * 2
			var toDelete []string

			// * First pass: collect IPs cần delete (minimal lock time)
			// * Check timestamps mà không hold visitor lock để tránh deadlock
			rl.mu.RLock()
			for ip, v := range rl.visitors {
				v.mu.Lock()
				lastSeen := v.lastSeen
				v.mu.Unlock()

				if now.Sub(lastSeen) > cutoff {
					toDelete = append(toDelete, ip)
				}
			}
			rl.mu.RUnlock()

			// * Second pass: delete collected IPs (write lock chỉ khi delete)
			if len(toDelete) > 0 {
				rl.mu.Lock()
				for _, ip := range toDelete {
					// * Double-check visitor vẫn tồn tại và vẫn old
					if v, exists := rl.visitors[ip]; exists {
						v.mu.Lock()
						if now.Sub(v.lastSeen) > cutoff {
							delete(rl.visitors, ip)
						}
						v.mu.Unlock()
					}
				}
				rl.mu.Unlock()
			}
		}
	}
}

// Stop dừng rate limiter và cancel cleanup goroutine
func (rl *RateLimiter) Stop() {
	rl.cancel()
}

// getVisitor trả về hoặc tạo visitor cho IP
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.RLock()
	v, exists := rl.visitors[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// * Double-check sau khi acquire write lock
		v, exists = rl.visitors[ip]
		if !exists {
			v = &visitor{
				tokens:   rl.rate,
				lastSeen: time.Now(),
			}
			rl.visitors[ip] = v
		}
		rl.mu.Unlock()
	}

	return v
}

// Allow kiểm tra request từ IP có được phép không
func (rl *RateLimiter) Allow(ip string) bool {
	v := rl.getVisitor(ip)

	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(v.lastSeen)

	// * Refill tokens dựa trên elapsed time
	if elapsed >= rl.window {
		v.tokens = rl.rate
		v.lastSeen = now
	} else {
		// * Refill tỷ lệ với elapsed time
		tokensToAdd := int(float64(rl.rate) * elapsed.Seconds() / rl.window.Seconds())
		if tokensToAdd > 0 {
			v.tokens += tokensToAdd
			if v.tokens > rl.rate {
				v.tokens = rl.rate
			}
			v.lastSeen = now
		}
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// RateLimitMiddleware tạo middleware rate limit requests
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)

			if !limiter.Allow(ip) {
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP trích xuất real client IP từ request - export để các package
// khác (vd handlers.PublicHandler) dùng chung logic thay vì viết lại.
func GetClientIP(r *http.Request) string {
	// * Kiểm tra X-Forwarded-For header (cho proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// * X-Forwarded-For có thể chứa nhiều IPs, format: "client, proxy1, proxy2"
		// * Lấy IP đầu tiên (original client IP)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// * Kiểm tra X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// * Fallback về RemoteAddr (remove port nếu có)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}

package middleware

import (
	"context"
	"net/http"
	"strconv"
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

// RateLimitMiddleware tạo middleware rate limit requests.
//
// Khoá đếm lấy từ ClientIPKey (đã gom IPv6 về /64) thay vì chuỗi IP thô, để
// người dùng IPv6 không thể xin bộ đếm mới bằng cách đổi địa chỉ trong cùng
// khối được cấp.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(ClientIPKey(r)) {
				WriteRateLimited(w, limiter.window)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WriteRateLimited trả 429 kèm Retry-After. Dùng chung cho mọi nơi từ chối vì
// vượt ngưỡng, để client (và cả bot lịch sự) nhận cùng một dạng phản hồi.
//
// Thân JSON thay cho http.Error dạng text: frontend đọc lỗi theo cùng khuôn
// {success,error:{code,message}} như phần còn lại của API.
func WriteRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(retryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"Bạn thao tác quá nhanh. Vui lòng chờ một lát rồi thử lại."}}`))
}

// GetClientIP đã chuyển sang clientip.go - nơi đó chỉ tin header chuyển tiếp
// khi request thật sự đến từ proxy tin cậy.

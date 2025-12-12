package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	mu          sync.RWMutex
	visitors    map[string]*visitor
	rate        int           // requests per window
	window      time.Duration // time window
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

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed
// window: time window for the rate limit
// cleanupInterval: how often to clean up old visitors
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

	// Start cleanup goroutine with context cancellation support
	go rl.cleanup()

	return rl
}

// cleanup removes old visitors to prevent memory leaks
// Optimized for high concurrency: batch deletion to reduce lock contention
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-rl.ctx.Done():
			return // Stop cleanup goroutine when context is cancelled
		case <-ticker.C:
			now := time.Now()
			cutoff := rl.window * 2
			var toDelete []string

			// First pass: collect IPs to delete (minimal lock time)
			// Check timestamps without holding visitor lock to avoid deadlock
			rl.mu.RLock()
			for ip, v := range rl.visitors {
				// Quick check without locking visitor - safe for read
				v.mu.Lock()
				lastSeen := v.lastSeen
				v.mu.Unlock()

				if now.Sub(lastSeen) > cutoff {
					toDelete = append(toDelete, ip)
				}
			}
			rl.mu.RUnlock()

			// Second pass: delete collected IPs (write lock only for deletion)
			if len(toDelete) > 0 {
				rl.mu.Lock()
				for _, ip := range toDelete {
					// Double-check visitor still exists and is still old
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

// Stop stops the rate limiter and cancels the cleanup goroutine
func (rl *RateLimiter) Stop() {
	rl.cancel()
}

// getVisitor returns or creates a visitor for the given IP
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.RLock()
	v, exists := rl.visitors[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
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

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	v := rl.getVisitor(ip)

	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(v.lastSeen)

	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		v.tokens = rl.rate
		v.lastSeen = now
	} else {
		// Refill proportional to elapsed time
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

// RateLimitMiddleware creates a middleware that rate limits requests
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs separated by comma
		// Format: "client, proxy1, proxy2"
		// We want the leftmost (original client) IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr (remove port if present)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}

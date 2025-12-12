package middleware

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

// ConcurrencyLimiter limits the number of concurrent requests using semaphore
type ConcurrencyLimiter struct {
	sem            *semaphore.Weighted
	maxConcurrent  int64
	current        int64
	waiting        int64
	acquireTimeout time.Duration
}

// NewConcurrencyLimiter creates a new concurrency limiter
func NewConcurrencyLimiter(maxConcurrent int64, acquireTimeout time.Duration) *ConcurrencyLimiter {
	if acquireTimeout == 0 {
		acquireTimeout = 30 * time.Second // Default: 30 seconds
	}
	return &ConcurrencyLimiter{
		sem:            semaphore.NewWeighted(maxConcurrent),
		maxConcurrent:  maxConcurrent,
		acquireTimeout: acquireTimeout,
	}
}

// ConcurrencyLimitMiddleware limits concurrent requests using semaphore
func ConcurrencyLimitMiddleware(limiter *ConcurrencyLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to acquire semaphore with context cancellation support
			ctx := r.Context()

			// Set timeout for acquiring semaphore (prevent indefinite wait)
			acquireCtx, cancel := context.WithTimeout(ctx, limiter.acquireTimeout)
			defer cancel()

			atomic.AddInt64(&limiter.waiting, 1)
			if err := limiter.sem.Acquire(acquireCtx, 1); err != nil {
				atomic.AddInt64(&limiter.waiting, -1)
				if err == context.DeadlineExceeded {
					http.Error(w, "Service busy, please try again later", http.StatusServiceUnavailable)
					return
				}
				if err == context.Canceled {
					http.Error(w, "Request cancelled", http.StatusRequestTimeout)
					return
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			atomic.AddInt64(&limiter.waiting, -1)
			atomic.AddInt64(&limiter.current, 1)
			defer func() {
				limiter.sem.Release(1)
				atomic.AddInt64(&limiter.current, -1)
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// GetStats returns current concurrency statistics
func (c *ConcurrencyLimiter) GetStats() (current, waiting int64) {
	return atomic.LoadInt64(&c.current), atomic.LoadInt64(&c.waiting)
}

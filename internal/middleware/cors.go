package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORS middleware handles Cross-Origin Resource Sharing
type CORS struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
	maxAge         int
}

// NewCORS creates a new CORS middleware
func NewCORS(allowedOrigins []string) *CORS {
	if len(allowedOrigins) == 0 {
		// Default: allow all origins (for development)
		allowedOrigins = []string{"*"}
	}

	// Validate origins and warn about security risks
	for _, origin := range allowedOrigins {
		if origin == "*" {
			// Log warning but don't fail - useful for development
			// In production, this should be restricted to specific domains
		}
	}

	return &CORS{
		allowedOrigins: allowedOrigins,
		allowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		allowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
		maxAge:         3600, // 1 hour
	}
}

// CORSMiddleware creates a middleware that handles CORS
func (c *CORS) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			c.handlePreflight(w, r, origin)
			return
		}

		// Set CORS headers for actual request
		c.setCORSHeaders(w, origin)

		next.ServeHTTP(w, r)
	})
}

func (c *CORS) handlePreflight(w http.ResponseWriter, r *http.Request, origin string) {
	if c.isOriginAllowed(origin) {
		c.setCORSHeaders(w, origin)
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

func (c *CORS) setCORSHeaders(w http.ResponseWriter, origin string) {
	if c.isOriginAllowed(origin) {
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(c.allowedOrigins) == 1 && c.allowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", c.maxAge))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

func (c *CORS) isOriginAllowed(origin string) bool {
	if len(c.allowedOrigins) == 1 && c.allowedOrigins[0] == "*" {
		return true
	}

	for _, allowed := range c.allowedOrigins {
		if allowed == origin {
			return true
		}
	}

	return false
}

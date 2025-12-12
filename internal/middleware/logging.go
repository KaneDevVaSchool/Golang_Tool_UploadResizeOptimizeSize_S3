package middleware

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		// * Include request ID trong logs để dễ trace
		requestID := GetRequestID(r.Context())
		if requestID != "" {
			log.Printf(
				"[%s] %s %s %s %v",
				requestID,
				r.Method,
				r.RequestURI,
				r.RemoteAddr,
				time.Since(start),
			)
		} else {
			log.Printf(
				"%s %s %s %v",
				r.Method,
				r.RequestURI,
				r.RemoteAddr,
				time.Since(start),
			)
		}
	})
}

package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/tkc/go-json-server/src/logger"
)

// Middleware represents an HTTP middleware
type Middleware func(http.Handler) http.Handler

// Logger is a middleware that logs HTTP requests
func Logger(log *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			// Call the next handler
			next.ServeHTTP(rw, r)
			
			// Calculate request duration and log the request
			duration := time.Since(start)
			log.AccessLog(r, rw.statusCode, duration)
		})
	}
}

// CORS is a middleware that adds CORS headers to responses
func CORS() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Timeout is a middleware that adds a timeout to the request context
func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Update request with new context
			r = r.WithContext(ctx)
			
			// Create done channel
			done := make(chan struct{})
			
			// Execute handler in goroutine
			go func() {
				next.ServeHTTP(w, r)
				close(done)
			}()

			// Wait for timeout or completion
			select {
			case <-done:
				return
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestTimeout)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "request timeout",
					})
				}
			}
		})
	}
}

// Recovery is a middleware that recovers from panics
func Recovery(log *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log panic details
					stack := debug.Stack()
					log.Error("Panic recovered", map[string]any{
						"error":      err,
						"stacktrace": string(stack),
						"path":       r.URL.Path,
						"method":     r.Method,
						"remoteAddr": r.RemoteAddr,
					})
					
					// Return error response to client
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "internal server error",
					})
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID is a middleware that assigns a unique ID to requests
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request ID is already set
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				// Generate UUID (in practice, use a UUID library)
				requestID = time.Now().Format("20060102150405") + "-" + randomString(8)
			}
			
			// Set request ID in response header
			w.Header().Set("X-Request-ID", requestID)
			
			// Add request ID to context
			ctx := context.WithValue(r.Context(), "requestID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Chain combines multiple middlewares into a single middleware
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write captures the default status code (200) if WriteHeader hasn't been called
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// randomString generates a random string
// Note: In production, use crypto/rand instead
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Small delay to get different values
	}
	return string(result)
}

package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tkc/go-json-server/src/logger"
)

func TestLogger_Middleware(t *testing.T) {
	// Create a buffer to capture logs
	var buf bytes.Buffer

	// Create a test logger
	log, err := logger.NewLogger(logger.LogConfig{
		Level:      logger.LevelDebug,
		Format:     logger.FormatText,
		TimeFormat: time.RFC3339,
	})
	assert.NoError(t, err)

	// Replace the writer with our buffer
	log.SetWriter(&buf)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Apply the middleware
	handler := Logger(log)(testHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute the handler
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())

	// Check log output
	logOutput := buf.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "200")
}

func TestCORS_Middleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Apply the middleware
	handler := CORS()(testHandler)

	// Test regular request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check CORS headers
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")

	// Test preflight request
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check response for OPTIONS
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Body.String()) // Empty body for OPTIONS
}

func TestTimeout_Middleware(t *testing.T) {
	// Create a handler that delays
	delayHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			// Context was canceled, don't write response
			return
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("delayed response"))
		}
	})

	// Test with timeout longer than delay
	t.Run("No timeout", func(t *testing.T) {
		handler := Timeout(200 * time.Millisecond)(delayHandler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "delayed response", w.Body.String())
	})

	// Test with timeout shorter than delay
	t.Run("With timeout", func(t *testing.T) {
		handler := Timeout(50 * time.Millisecond)(delayHandler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestTimeout, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "request timeout", response["error"])
	})
}

func TestRecovery_Middleware(t *testing.T) {
	// Create a buffer to capture logs
	var buf bytes.Buffer

	// Create a test logger
	log, err := logger.NewLogger(logger.LogConfig{
		Level:      logger.LevelDebug,
		Format:     logger.FormatText,
		TimeFormat: time.RFC3339,
	})
	assert.NoError(t, err)

	// Replace the writer with our buffer
	log.SetWriter(&buf)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Apply the middleware
	handler := Recovery(log)(panicHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute the handler (should recover from panic)
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "internal server error", response["error"])

	// Check log output
	logOutput := buf.String()
	assert.Contains(t, logOutput, "ERROR")
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "test panic")
}

func TestRequestID_Middleware(t *testing.T) {
	// Create a handler that checks request ID
	var capturedID string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Context().Value("requestID").(string)
		if ok {
			capturedID = id
		}
		w.WriteHeader(http.StatusOK)
	})

	// Apply the middleware
	handler := RequestID()(testHandler)

	// Test without existing request ID
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that a request ID was generated and added to both context and response
	assert.NotEmpty(t, capturedID)
	assert.Equal(t, capturedID, w.Header().Get("X-Request-ID"))

	// Test with existing request ID
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that the existing ID was preserved
	assert.Equal(t, "existing-id", capturedID)
	assert.Equal(t, "existing-id", w.Header().Get("X-Request-ID"))
}

func TestChain_Middleware(t *testing.T) {
	// Create middleware that add headers
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-1", "value1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-2", "value2")
			next.ServeHTTP(w, r)
		})
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Chain middleware
	handler := Chain(middleware1, middleware2)(testHandler)

	// Execute
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check headers were set in the correct order
	assert.Equal(t, "value1", w.Header().Get("X-Test-1"))
	assert.Equal(t, "value2", w.Header().Get("X-Test-2"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseWriter(t *testing.T) {
	origWriter := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: origWriter,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusCreated, origWriter.Code)

	// Test calling WriteHeader again (should not change status)
	rw.WriteHeader(http.StatusOK)
	assert.Equal(t, http.StatusCreated, rw.statusCode) // Status should not change

	// Test Write
	origWriter = httptest.NewRecorder()
	rw = &responseWriter{
		ResponseWriter: origWriter,
	}
	rw.Write([]byte("test"))
	assert.Equal(t, http.StatusOK, rw.statusCode) // Default status
	assert.True(t, rw.written)
	assert.Equal(t, "test", origWriter.Body.String())
}

func TestRandomString(t *testing.T) {
	// Test length
	for _, length := range []int{8, 16, 32} {
		result := randomString(length)
		assert.Len(t, result, length)
	}

	// Test uniqueness
	s1 := randomString(16)
	s2 := randomString(16)
	assert.NotEqual(t, s1, s2)
}

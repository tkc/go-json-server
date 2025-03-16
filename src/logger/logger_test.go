package logger

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"WARN", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"fatal", LevelFatal},
		{"FATAL", LevelFatal},
		{"unknown", LevelInfo},
		{"", LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseLogLevel(tt.input))
		})
	}
}

func TestNewLogger(t *testing.T) {
	// Test with stdout
	stdoutLogger, err := NewLogger(LogConfig{
		Level:  LevelInfo,
		Format: FormatText,
	})
	assert.NoError(t, err)
	assert.NotNil(t, stdoutLogger)

	// Test with stderr
	stderrLogger, err := NewLogger(LogConfig{
		Level:      LevelDebug,
		Format:     FormatJSON,
		OutputPath: "stderr",
	})
	assert.NoError(t, err)
	assert.NotNil(t, stderrLogger)

	// We don't test file output here as it would require temp file handling
	// That's covered implicitly in the log() test below
}

func TestLogger_log(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a logger that writes to the buffer
	log := &Logger{
		level:      LevelDebug,
		format:     FormatText,
		writer:     &buf,
		timeFormat: time.RFC3339,
	}

	// Test text format logging
	log.Debug("Debug message", map[string]any{"key": "value"})
	output := buf.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "Debug message")
	assert.Contains(t, output, `"key":"value"`)

	// Reset buffer
	buf.Reset()

	// Test JSON format logging
	log.format = FormatJSON
	log.Info("Info message", map[string]any{"number": 42})
	output = buf.String()

	var logEntry LogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "INFO", logEntry.Level)
	assert.Equal(t, "Info message", logEntry.Message)
	assert.Equal(t, float64(42), logEntry.Data["number"])
}

func TestLogger_LogMethods(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a logger that writes to the buffer
	log := &Logger{
		level:      LevelDebug,
		format:     FormatText,
		writer:     &buf,
		timeFormat: time.RFC3339,
	}

	// Test each log method
	tests := []struct {
		method      func(string, ...map[string]any)
		level       string
		message     string
		data        map[string]any
		shouldExist bool // If the level is above the logger's level, it won't appear
	}{
		{log.Debug, "DEBUG", "Debug test", map[string]any{"test": "debug"}, true},
		{log.Info, "INFO", "Info test", map[string]any{"test": "info"}, true},
		{log.Warn, "WARN", "Warn test", map[string]any{"test": "warn"}, true},
		{log.Error, "ERROR", "Error test", map[string]any{"test": "error"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			buf.Reset()
			tt.method(tt.message, tt.data)
			output := buf.String()

			if tt.shouldExist {
				assert.Contains(t, output, tt.level)
				assert.Contains(t, output, tt.message)
				for k, v := range tt.data {
					assert.Contains(t, output, k)
					assert.Contains(t, output, v)
				}
			} else {
				assert.Empty(t, output)
			}
		})
	}

	// We don't test Fatal as it would call os.Exit(1)
}

func TestLogger_AccessLog(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a logger that writes to the buffer
	log := &Logger{
		level:      LevelDebug,
		format:     FormatText,
		writer:     &buf,
		timeFormat: time.RFC3339,
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")

	// Log an access entry
	log.AccessLog(req, 200, 100*time.Millisecond)
	output := buf.String()

	// Check text format
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test")
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "100.00ms")
	assert.Contains(t, output, "test-agent")

	// Reset buffer and test JSON format
	buf.Reset()
	log.format = FormatJSON

	// Create a request with JSON body
	jsonBody := `{"test":"value"}`
	req = httptest.NewRequest("POST", "/api", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	// Log an access entry
	log.AccessLog(req, 201, 150*time.Millisecond)
	output = buf.String()

	var accessEntry AccessLogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &accessEntry)
	assert.NoError(t, err)
	assert.Equal(t, "POST", accessEntry.Method)
	assert.Equal(t, "/api", accessEntry.Path)
	assert.Equal(t, 201, accessEntry.Status)
	assert.Equal(t, 150.0, accessEntry.Latency)
	assert.Equal(t, "test-agent", accessEntry.UserAgent)
}

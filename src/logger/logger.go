package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogLevel represents logging levels
type LogLevel int

const (
	// Log level definitions
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a log level
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo // Default is Info
	}
}

// LogFormat represents the format type for logs
type LogFormat string

const (
	// Log format definitions
	FormatText LogFormat = "text"
	FormatJSON LogFormat = "json"
)

// Logger represents a logger instance
type Logger struct {
	level      LogLevel
	format     LogFormat
	writer     io.Writer
	timeFormat string
}

// LogConfig holds logger configuration
type LogConfig struct {
	Level      LogLevel
	Format     LogFormat
	OutputPath string
	TimeFormat string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// AccessLogEntry represents an HTTP access log entry
type AccessLogEntry struct {
	Time       string         `json:"time"`
	RemoteAddr string         `json:"remote_addr"`
	Method     string         `json:"method"`
	Path       string         `json:"path"`
	Protocol   string         `json:"protocol"`
	Status     int            `json:"status"`
	UserAgent  string         `json:"user_agent"`
	Latency    float64        `json:"latency_ms"`
	Body       map[string]any `json:"body,omitempty"`
}

// NewLogger creates a new logger instance
func NewLogger(config LogConfig) (*Logger, error) {
	var writer io.Writer

	// Configure output destination
	switch config.OutputPath {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// Ensure directory exists
		dir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		
		file, err := os.OpenFile(config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writer = file
	}

	// Default time format
	timeFormat := config.TimeFormat
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	// Default format
	format := config.Format
	if format == "" {
		format = FormatText
	}

	return &Logger{
		level:      config.Level,
		format:     format,
		writer:     writer,
		timeFormat: timeFormat,
	}, nil
}

// log records a message at the specified level
func (l *Logger) log(level LogLevel, message string, data map[string]any) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format(l.timeFormat)

	entry := LogEntry{
		Time:    timestamp,
		Level:   level.String(),
		Message: message,
		Data:    data,
	}

	var err error
	var output []byte

	switch l.format {
	case FormatJSON:
		output, err = json.Marshal(entry)
		if err == nil {
			output = append(output, '\n')
		}
	default: // FormatText
		var dataStr string
		if len(data) > 0 {
			dataJSON, jsonErr := json.Marshal(data)
			if jsonErr == nil {
				dataStr = string(dataJSON)
			} else {
				dataStr = fmt.Sprintf("%+v", data)
			}
		}

		if dataStr != "" {
			output = []byte(fmt.Sprintf("[%s] %s - %s: %s\n", entry.Time, entry.Level, entry.Message, dataStr))
		} else {
			output = []byte(fmt.Sprintf("[%s] %s - %s\n", entry.Time, entry.Level, entry.Message))
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling log entry: %v\n", err)
		return
	}

	if _, err := l.writer.Write(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing log: %v\n", err)
	}

	// Exit program on Fatal level logs
	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, data ...map[string]any) {
	var logData map[string]any
	if len(data) > 0 {
		logData = data[0]
	}
	l.log(LevelDebug, message, logData)
}

// Info logs an info message
func (l *Logger) Info(message string, data ...map[string]any) {
	var logData map[string]any
	if len(data) > 0 {
		logData = data[0]
	}
	l.log(LevelInfo, message, logData)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, data ...map[string]any) {
	var logData map[string]any
	if len(data) > 0 {
		logData = data[0]
	}
	l.log(LevelWarn, message, logData)
}

// Error logs an error message
func (l *Logger) Error(message string, data ...map[string]any) {
	var logData map[string]any
	if len(data) > 0 {
		logData = data[0]
	}
	l.log(LevelError, message, logData)
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(message string, data ...map[string]any) {
	var logData map[string]any
	if len(data) > 0 {
		logData = data[0]
	}
	l.log(LevelFatal, message, logData)
}

// AccessLog records an HTTP request in the log
func (l *Logger) AccessLog(r *http.Request, status int, latency time.Duration) {
	var reqBody map[string]any

	// Read body for non-GET requests
	if r.Method != http.MethodGet && r.Header.Get("Content-Type") == "application/json" {
		if r.Body != nil {
			var bodyBytes []byte
			var err error
			
			// Read request body
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				// Log error
				l.Error("Failed to read request body", map[string]any{"error": err.Error()})
			} else {
				// Restore body so it can be read again
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				
				// Try to parse as JSON
				if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
					l.Debug("Failed to parse request body as JSON", map[string]any{"error": err.Error()})
				}
			}
		}
	}

	// Calculate latency in milliseconds
	latencyMs := float64(latency.Microseconds()) / 1000.0

	entry := AccessLogEntry{
		Time:       time.Now().Format(l.timeFormat),
		RemoteAddr: r.RemoteAddr,
		Method:     r.Method,
		Path:       r.URL.Path,
		Protocol:   r.Proto,
		Status:     status,
		UserAgent:  r.UserAgent(),
		Latency:    latencyMs,
		Body:       reqBody,
	}

	var err error
	var output []byte

	switch l.format {
	case FormatJSON:
		output, err = json.Marshal(entry)
		if err == nil {
			output = append(output, '\n')
		}
	default: // FormatText
		output = []byte(fmt.Sprintf(
			"[%s] %s - %s %s %s %d %.2fms %s\n",
			entry.Time,
			entry.RemoteAddr,
			entry.Method,
			entry.Path,
			entry.Protocol,
			entry.Status,
			entry.Latency,
			entry.UserAgent,
		))
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling access log entry: %v\n", err)
		return
	}

	if _, err := l.writer.Write(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing access log: %v\n", err)
	}
}

// Close closes the logger's file handle
func (l *Logger) Close() error {
	if closer, ok := l.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

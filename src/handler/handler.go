package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tkc/go-json-server/src/config"
	"github.com/tkc/go-json-server/src/logger"
)

// Error definitions
var (
	ErrNotFound         = errors.New("endpoint not found")
	ErrInternalServer   = errors.New("internal server error")
	ErrInvalidJSON      = errors.New("invalid JSON format")
	ErrJSONFileNotFound = errors.New("JSON file not found")
	ErrMethodNotAllowed = errors.New("method not allowed")
)

// Content type constants
const (
	MIMEApplicationJSON     = "application/json"
	MIMEApplicationJSONUTF8 = MIMEApplicationJSON + "; charset=UTF-8"
	MIMETextPlainUTF8       = "text/plain; charset=UTF-8"
)

// contextKey is a custom type used for context value keys
type contextKey string

const (
	// PathParamsKey is the context key for path parameters
	PathParamsKey contextKey = "pathParams"
)

// ResponseCache caches JSON responses
type ResponseCache struct {
	mu    sync.RWMutex
	cache map[string]cachedResponse
}

// cachedResponse represents a cached response
type cachedResponse struct {
	content    []byte
	expiration time.Time
}

// NewResponseCache creates a new response cache
func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		cache: make(map[string]cachedResponse),
	}
}

// Get retrieves a cached response
func (c *ResponseCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	// Check if the cache has expired
	if time.Now().After(cached.expiration) {
		delete(c.cache, key)
		return nil, false
	}

	return cached.content, true
}

// Set stores a response in the cache
func (c *ResponseCache) Set(key string, content []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = cachedResponse{
		content:    content,
		expiration: time.Now().Add(ttl),
	}
}

// Clear clears the cache
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]cachedResponse)
}

// Server represents the JSON server
type Server struct {
	Config      *config.Config
	Logger      *logger.Logger
	Cache       *ResponseCache
	CacheTTL    time.Duration
	PathParams  map[string][]string
	paramRegexp *regexp.Regexp
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, log *logger.Logger, cacheTTL time.Duration) *Server {
	s := &Server{
		Config:      cfg,
		Logger:      log,
		Cache:       NewResponseCache(),
		CacheTTL:    cacheTTL,
		PathParams:  make(map[string][]string),
		paramRegexp: regexp.MustCompile(`:([\w]+)`),
	}

	// Pre-process endpoints to find path parameters
	for _, ep := range cfg.GetEndpoints() {
		if ep.Folder == "" { // Only for API endpoints, not static file servers
			params := s.extractPathParams(ep.Path)
			if len(params) > 0 {
				s.PathParams[ep.Path] = params
			}
		}
	}

	return s
}

// extractPathParams extracts parameter names from a path pattern
// e.g., "/users/:id" returns ["id"]
func (s *Server) extractPathParams(path string) []string {
	matches := s.paramRegexp.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// matchPath checks if a request path matches a route pattern
// and extracts path parameters if present
func (s *Server) matchPath(pattern, path string) (bool, map[string]string) {
	// Check for exact match (no parameters)
	if pattern == path {
		return true, nil
	}

	// Check if this pattern has registered parameters
	_, hasParams := s.PathParams[pattern]
	if !hasParams {
		return false, nil
	}

	// Convert pattern to regexp
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false, nil
	}

	paramValues := make(map[string]string)

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			// This is a parameter
			paramName := part[1:] // Remove the colon
			paramValues[paramName] = pathParts[i]
		} else if part != pathParts[i] {
			// Non-parameter parts must match exactly
			return false, nil
		}
	}

	return true, paramValues
}

// HandleRequest handles all HTTP requests
func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers (this is also done in middleware, but useful as a fallback)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

	// Handle OPTIONS requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check for file server endpoints first
	for _, ep := range s.Config.GetEndpoints() {
		if ep.Folder != "" && strings.HasPrefix(r.URL.Path, ep.Path) {
			// This is a static file server endpoint
			fileServer := http.StripPrefix(ep.Path, http.FileServer(http.Dir(ep.Folder)))
			fileServer.ServeHTTP(w, r)
			return
		}
	}

	// Handle API endpoints
	for _, ep := range s.Config.GetEndpoints() {
		if ep.Folder != "" {
			continue // Skip file server endpoints
		}

		// Check if path matches (with or without params)
		match, pathParams := s.matchPath(ep.Path, r.URL.Path)

		if match && ep.Method == r.Method {
			s.Logger.Debug("Matched endpoint", map[string]any{
				"path":    r.URL.Path,
				"method":  r.Method,
				"pattern": ep.Path,
				"params":  pathParams,
			})

			// Store path params in context
			ctx := context.WithValue(r.Context(), PathParamsKey, pathParams)
			r = r.WithContext(ctx)

			// Set headers
			w.Header().Set("Content-Type", MIMEApplicationJSONUTF8)

			// Try to get response from cache
			cacheKey := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
			if cachedResponse, found := s.Cache.Get(cacheKey); found {
				w.WriteHeader(ep.Status)
				w.Write(cachedResponse)
				return
			}

			// Get JSON response
			respBody, err := s.getJSONResponse(ep.JsonPath, pathParams)
			if err != nil {
				s.Logger.Error("Error getting JSON response", map[string]any{
					"error": err.Error(),
					"path":  ep.JsonPath,
				})

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Internal server error"}`))
				return
			}

			// Write response
			w.WriteHeader(ep.Status)
			w.Write(respBody)

			// Cache the response for future requests
			s.Cache.Set(cacheKey, respBody, s.CacheTTL)
			return
		}
	}

	// If we got here, no endpoint matched
	w.Header().Set("Content-Type", MIMEApplicationJSONUTF8)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "Not found"}`))
}

// getJSONResponse gets the JSON response for an endpoint
func (s *Server) getJSONResponse(jsonPath string, pathParams map[string]string) ([]byte, error) {
	file, err := os.Open(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJSONFileNotFound, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file: %w", err)
	}

	// If no path parameters, return the content as is
	if len(pathParams) == 0 {
		return content, nil
	}

	// Replace path parameters in the JSON content if needed
	contentStr := string(content)
	for param, value := range pathParams {
		placeholder := fmt.Sprintf(":%s", param)
		contentStr = strings.ReplaceAll(contentStr, placeholder, value)
	}

	// Validate that the result is still valid JSON
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(contentStr), &jsonObj); err != nil {
		// If parameter replacement made the JSON invalid, return the original
		s.Logger.Warn("Parameter replacement resulted in invalid JSON", map[string]any{
			"error": err.Error(),
			"path":  jsonPath,
		})
		return content, nil
	}

	return []byte(contentStr), nil
}

// ClearCache clears the response cache
func (s *Server) ClearCache() {
	s.Cache.Clear()
	s.Logger.Info("Response cache cleared")
}

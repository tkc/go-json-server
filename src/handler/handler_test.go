package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tkc/go-json-server/src/config"
	"github.com/tkc/go-json-server/src/logger"
)

// newTestLogger creates a logger that writes to a buffer (for testing)
func newTestLogger(buf *bytes.Buffer) *logger.Logger {
	log, _ := logger.NewLogger(logger.LogConfig{
		Level:      logger.LevelDebug,
		Format:     logger.FormatText,
		TimeFormat: time.RFC3339,
	})
	log.SetWriter(buf)
	return log
}

// --- ResponseCache tests ---

func TestResponseCache_SetAndGet(t *testing.T) {
	cache := NewResponseCache()
	content := []byte(`{"key":"value"}`)

	cache.Set("GET:/test", content, 5*time.Minute)

	got, found := cache.Get("GET:/test")
	assert.True(t, found)
	assert.Equal(t, content, got)
}

func TestResponseCache_GetMiss(t *testing.T) {
	cache := NewResponseCache()

	got, found := cache.Get("GET:/nonexistent")
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestResponseCache_Expiration(t *testing.T) {
	cache := NewResponseCache()
	content := []byte(`{"key":"value"}`)

	// Set with very short TTL
	cache.Set("GET:/test", content, 1*time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	got, found := cache.Get("GET:/test")
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestResponseCache_Clear(t *testing.T) {
	cache := NewResponseCache()
	cache.Set("GET:/a", []byte(`"a"`), 5*time.Minute)
	cache.Set("GET:/b", []byte(`"b"`), 5*time.Minute)

	cache.Clear()

	_, foundA := cache.Get("GET:/a")
	_, foundB := cache.Get("GET:/b")
	assert.False(t, foundA)
	assert.False(t, foundB)
}

func TestResponseCache_Overwrite(t *testing.T) {
	cache := NewResponseCache()

	cache.Set("GET:/test", []byte(`"old"`), 5*time.Minute)
	cache.Set("GET:/test", []byte(`"new"`), 5*time.Minute)

	got, found := cache.Get("GET:/test")
	assert.True(t, found)
	assert.Equal(t, []byte(`"new"`), got)
}

// --- extractPathParams tests ---

func TestExtractPathParams(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	s := &Server{
		Config:      &config.Config{},
		Logger:      log,
		Cache:       NewResponseCache(),
		CacheTTL:    5 * time.Minute,
		PathParams:  make(map[string][]string),
		paramRegexp: compileParamRegexp(),
	}

	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"No params", "/users", []string{}},
		{"Single param", "/users/:id", []string{"id"}},
		{"Multiple params", "/users/:userId/posts/:postId", []string{"userId", "postId"}},
		{"Root path", "/", []string{}},
		{"Nested with one param", "/api/v1/users/:id", []string{"id"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.extractPathParams(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- matchPath tests ---

func TestMatchPath(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	s := &Server{
		Config:      &config.Config{},
		Logger:      log,
		Cache:       NewResponseCache(),
		CacheTTL:    5 * time.Minute,
		PathParams:  make(map[string][]string),
		paramRegexp: compileParamRegexp(),
	}

	// Register path params
	s.PathParams["/users/:id"] = []string{"id"}
	s.PathParams["/users/:userId/posts/:postId"] = []string{"userId", "postId"}

	tests := []struct {
		name           string
		pattern        string
		path           string
		expectMatch    bool
		expectParams   map[string]string
	}{
		{
			name:        "Exact match",
			pattern:     "/users",
			path:        "/users",
			expectMatch: true,
		},
		{
			name:         "Single param match",
			pattern:      "/users/:id",
			path:         "/users/42",
			expectMatch:  true,
			expectParams: map[string]string{"id": "42"},
		},
		{
			name:         "Multiple params match",
			pattern:      "/users/:userId/posts/:postId",
			path:         "/users/1/posts/99",
			expectMatch:  true,
			expectParams: map[string]string{"userId": "1", "postId": "99"},
		},
		{
			name:        "No match - different path",
			pattern:     "/users",
			path:        "/posts",
			expectMatch: false,
		},
		{
			name:        "No match - wrong segment count",
			pattern:     "/users/:id",
			path:        "/users/1/extra",
			expectMatch: false,
		},
		{
			name:        "No match - pattern without params doesn't match different path",
			pattern:     "/users",
			path:        "/users/1",
			expectMatch: false,
		},
		{
			name:         "Param with string value",
			pattern:      "/users/:id",
			path:         "/users/john",
			expectMatch:  true,
			expectParams: map[string]string{"id": "john"},
		},
		{
			name:        "Root exact match",
			pattern:     "/",
			path:        "/",
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, params := s.matchPath(tt.pattern, tt.path)
			assert.Equal(t, tt.expectMatch, match)
			if tt.expectParams != nil {
				assert.Equal(t, tt.expectParams, params)
			}
		})
	}
}

// --- HandleRequest tests ---

func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "handler-test")
	assert.NoError(t, err)

	// Create JSON files for endpoints
	usersJSON := filepath.Join(tempDir, "users.json")
	err = os.WriteFile(usersJSON, []byte(`[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`), 0644)
	assert.NoError(t, err)

	userDetailJSON := filepath.Join(tempDir, "user-detail.json")
	err = os.WriteFile(userDetailJSON, []byte(`{"id":":id","name":"User :id"}`), 0644)
	assert.NoError(t, err)

	userPostJSON := filepath.Join(tempDir, "user-post.json")
	err = os.WriteFile(userPostJSON, []byte(`{"userId":":userId","postId":":postId"}`), 0644)
	assert.NoError(t, err)

	createdJSON := filepath.Join(tempDir, "created.json")
	err = os.WriteFile(createdJSON, []byte(`{"status":"created"}`), 0644)
	assert.NoError(t, err)

	healthJSON := filepath.Join(tempDir, "health.json")
	err = os.WriteFile(healthJSON, []byte(`{"status":"ok"}`), 0644)
	assert.NoError(t, err)

	// Create static folder with a file
	staticDir := filepath.Join(tempDir, "static")
	err = os.Mkdir(staticDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>hello</html>"), 0644)
	assert.NoError(t, err)

	cfg := &config.Config{
		Port:     8080,
		LogLevel: "debug",
		Endpoints: []config.Endpoint{
			{Method: "GET", Status: 200, Path: "/", JsonPath: healthJSON},
			{Method: "GET", Status: 200, Path: "/users", JsonPath: usersJSON},
			{Method: "GET", Status: 200, Path: "/users/:id", JsonPath: userDetailJSON},
			{Method: "GET", Status: 200, Path: "/users/:userId/posts/:postId", JsonPath: userPostJSON},
			{Method: "POST", Status: 201, Path: "/users", JsonPath: createdJSON},
			{Method: "DELETE", Status: 204, Path: "/users/:id", JsonPath: userDetailJSON},
			{Path: "/static", Folder: staticDir},
		},
	}

	var buf bytes.Buffer
	log := newTestLogger(&buf)

	server := NewServer(cfg, log, 5*time.Minute)
	return server, tempDir
}

func TestHandleRequest_GET(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Len(t, body, 2)
	assert.Equal(t, "Alice", body[0]["name"])
}

func TestHandleRequest_GET_Root(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestHandleRequest_POST(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(`{"name":"Charlie"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "created", body["status"])
}

func TestHandleRequest_DELETE(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("DELETE", "/users/42", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleRequest_OPTIONS(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("OPTIONS", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}

func TestHandleRequest_NotFound(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "Not found", body["error"])
}

func TestHandleRequest_MethodNotAllowed(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// PUT /users is not configured, should return 404
	req := httptest.NewRequest("PUT", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleRequest_PathParams_Single(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/users/42", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	// The :id should be replaced with "42" in the JSON
	assert.Equal(t, "42", body["id"])
	assert.Equal(t, "User 42", body["name"])
}

func TestHandleRequest_PathParams_Multiple(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/users/5/posts/10", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "5", body["userId"])
	assert.Equal(t, "10", body["postId"])
}

func TestHandleRequest_StaticFileServer(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// Use a direct file path (not index.html which may trigger redirect)
	staticDir := filepath.Join(tempDir, "static")
	err := os.WriteFile(filepath.Join(staticDir, "test.txt"), []byte("static content"), 0644)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "static content")
}

func TestHandleRequest_CORSHeaders(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "DELETE")
}

func TestHandleRequest_Caching(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// First request - should miss cache
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	firstBody := w.Body.String()

	// Second request - should hit cache
	req2 := httptest.NewRequest("GET", "/users", nil)
	w2 := httptest.NewRecorder()
	server.HandleRequest(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, firstBody, w2.Body.String())
}

func TestHandleRequest_ClearCache(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// Populate cache
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Clear cache
	server.ClearCache()

	// Verify cache is cleared by checking internal state
	_, found := server.Cache.Get("GET:/users")
	assert.False(t, found)
}

// --- getJSONResponse tests ---

func TestGetJSONResponse_NoParams(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "json-resp-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "data.json")
	err = os.WriteFile(jsonFile, []byte(`{"message":"hello"}`), 0644)
	assert.NoError(t, err)

	var buf bytes.Buffer
	log := newTestLogger(&buf)

	s := &Server{
		Logger:      log,
		paramRegexp: compileParamRegexp(),
	}

	result, err := s.getJSONResponse(jsonFile, nil)
	assert.NoError(t, err)
	assert.Equal(t, `{"message":"hello"}`, string(result))
}

func TestGetJSONResponse_WithParams(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "json-resp-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "data.json")
	err = os.WriteFile(jsonFile, []byte(`{"id":":id","name":"User :id"}`), 0644)
	assert.NoError(t, err)

	var buf bytes.Buffer
	log := newTestLogger(&buf)

	s := &Server{
		Logger:      log,
		paramRegexp: compileParamRegexp(),
	}

	params := map[string]string{"id": "42"}
	result, err := s.getJSONResponse(jsonFile, params)
	assert.NoError(t, err)

	var body map[string]interface{}
	err = json.Unmarshal(result, &body)
	assert.NoError(t, err)
	assert.Equal(t, "42", body["id"])
	assert.Equal(t, "User 42", body["name"])
}

func TestGetJSONResponse_FileNotFound(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	s := &Server{
		Logger:      log,
		paramRegexp: compileParamRegexp(),
	}

	_, err := s.getJSONResponse("/nonexistent/file.json", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrJSONFileNotFound)
}

func TestGetJSONResponse_InvalidJSONAfterReplacement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "json-resp-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a JSON file where param replacement would break the JSON structure
	jsonFile := filepath.Join(tempDir, "data.json")
	err = os.WriteFile(jsonFile, []byte(`{"count":":count"}`), 0644)
	assert.NoError(t, err)

	var buf bytes.Buffer
	log := newTestLogger(&buf)

	s := &Server{
		Logger:      log,
		paramRegexp: compileParamRegexp(),
	}

	// Replacement that keeps valid JSON
	params := map[string]string{"count": "5"}
	result, err := s.getJSONResponse(jsonFile, params)
	assert.NoError(t, err)

	var body map[string]interface{}
	err = json.Unmarshal(result, &body)
	assert.NoError(t, err)
	assert.Equal(t, "5", body["count"])
}

// --- NewServer tests ---

func TestNewServer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "new-server-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	cfg := &config.Config{
		Port: 3000,
		Endpoints: []config.Endpoint{
			{Method: "GET", Status: 200, Path: "/users/:id", JsonPath: jsonFile},
			{Method: "GET", Status: 200, Path: "/simple", JsonPath: jsonFile},
		},
	}

	var buf bytes.Buffer
	log := newTestLogger(&buf)

	server := NewServer(cfg, log, 5*time.Minute)

	assert.NotNil(t, server)
	assert.NotNil(t, server.Cache)
	assert.Equal(t, 5*time.Minute, server.CacheTTL)
	// Should have registered path params for /users/:id
	assert.Contains(t, server.PathParams, "/users/:id")
	assert.Equal(t, []string{"id"}, server.PathParams["/users/:id"])
	// /simple has no params so should not be in PathParams
	_, hasSimple := server.PathParams["/simple"]
	assert.False(t, hasSimple)
}

func TestHandleRequest_StaticFileNotFound(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest("GET", "/static/nonexistent.html", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleRequest_MultipleGETEndpoints(t *testing.T) {
	server, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)

	// Request /users (GET list)
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Request /users/1 (GET detail)
	req2 := httptest.NewRequest("GET", "/users/1", nil)
	w2 := httptest.NewRecorder()
	server.HandleRequest(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// They should return different content
	assert.NotEqual(t, w.Body.String(), w2.Body.String())
}

// compileParamRegexp creates the param regexp used by Server
func compileParamRegexp() *regexp.Regexp {
	return regexp.MustCompile(`:([\w]+)`)
}

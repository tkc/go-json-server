package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test JSON file
	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{"message":"test"}`), 0644)
	assert.NoError(t, err)

	// Create a test config file
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"port": 8080,
		"logLevel": "debug",
		"logFormat": "json",
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test loading the config
	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Len(t, cfg.Endpoints, 1)
	assert.Equal(t, "GET", cfg.Endpoints[0].Method)
	assert.Equal(t, 200, cfg.Endpoints[0].Status)
	assert.Equal(t, "/test", cfg.Endpoints[0].Path)
	assert.Equal(t, jsonFile, cfg.Endpoints[0].JsonPath)
}

func TestConfig_Validate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "validate-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test JSON file
	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{"message":"test"}`), 0644)
	assert.NoError(t, err)

	// Create a test folder
	testFolder := filepath.Join(tempDir, "static")
	err = os.Mkdir(testFolder, 0755)
	assert.NoError(t, err)

	type testCase struct {
		name      string
		setupFn   func() Config
		wantError bool
	}

	tests := []testCase{
		{
			name: "Valid config",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{
						{Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
					},
				}
			},
			wantError: false,
		},
		{
			name: "Valid config with file server",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{
						{Path: "/static", Folder: testFolder},
					},
				}
			},
			wantError: false,
		},
		{
			name: "No endpoints",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{},
				}
			},
			wantError: true,
		},
		{
			name: "Empty path",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{
						{Method: "GET", Path: "", JsonPath: jsonFile, Status: 200},
					},
				}
			},
			wantError: true,
		},
		{
			name: "Duplicate endpoint",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{
						{Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
						{Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
					},
				}
			},
			wantError: true,
		},
		{
			name: "JSON file not found",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{
						{Method: "GET", Path: "/test", JsonPath: filepath.Join(tempDir, "notfound.json"), Status: 200},
					},
				}
			},
			wantError: true,
		},
		{
			name: "Folder not found",
			setupFn: func() Config {
				return Config{
					Endpoints: []Endpoint{
						{Path: "/static", Folder: filepath.Join(tempDir, "notfound")},
					},
				}
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupFn()
			err := config.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Reload(t *testing.T) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "reload-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test JSON file
	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{"message":"test"}`), 0644)
	assert.NoError(t, err)

	// Initial config
	configPath := filepath.Join(tempDir, "config.json")
	initialConfig := `{
		"port": 8080,
		"logLevel": "debug",
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// Load the initial config
	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)

	// Updated config
	updatedConfig := `{
		"port": 9090,
		"logLevel": "info",
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	assert.NoError(t, err)

	// Reload the config
	err = cfg.Reload(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestWatchConfig(t *testing.T) {
	// This test is simplified as full testing would require more complex setup
	tempDir, err := os.MkdirTemp("", "watch-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test JSON file
	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{"message":"test"}`), 0644)
	assert.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"port": 8080,
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Load the config
	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)

	// Setup a channel to receive notifications
	reloadCh := make(chan bool, 1)

	// Start watching
	err = WatchConfig(configPath, cfg, reloadCh)
	assert.NoError(t, err)

	// We can't easily test the file watching functionality in a unit test
	// but we can at least verify the watcher is set up without errors
}

func TestLoadConfig_Defaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "defaults-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	// Config with no port, logLevel, or logFormat
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 3000, cfg.Port, "default port should be 3000")
	assert.Equal(t, "info", cfg.LogLevel, "default logLevel should be info")
	assert.Equal(t, "text", cfg.LogFormat, "default logFormat should be text")
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "invalid-json-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	err = os.WriteFile(configPath, []byte(`{invalid json}`), 0644)
	assert.NoError(t, err)

	_, err = LoadConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing config file")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestLoadConfig_WithHost(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "host-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"host": "localhost",
		"port": 9090,
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 9090, cfg.Port)
}

func TestConfig_GetEndpoints(t *testing.T) {
	cfg := &Config{
		Endpoints: []Endpoint{
			{Method: "GET", Path: "/a"},
			{Method: "POST", Path: "/b"},
		},
	}

	endpoints := cfg.GetEndpoints()
	assert.Len(t, endpoints, 2)
	assert.Equal(t, "/a", endpoints[0].Path)
	assert.Equal(t, "/b", endpoints[1].Path)

	// Modifying returned slice should not affect original
	endpoints[0].Path = "/modified"
	assert.Equal(t, "/a", cfg.Endpoints[0].Path)
}

func TestConfig_GetPort(t *testing.T) {
	cfg := &Config{Port: 4000}
	assert.Equal(t, 4000, cfg.GetPort())
}

func TestConfig_GetHost(t *testing.T) {
	cfg := &Config{Host: "example.com"}
	assert.Equal(t, "example.com", cfg.GetHost())
}

func TestConfig_GetLogConfig(t *testing.T) {
	cfg := &Config{
		LogLevel:  "debug",
		LogFormat: "json",
		LogPath:   "/var/log/server.log",
	}

	level, format, path := cfg.GetLogConfig()
	assert.Equal(t, "debug", level)
	assert.Equal(t, "json", format)
	assert.Equal(t, "/var/log/server.log", path)
}

func TestConfig_Validate_DifferentMethodsSamePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "methods-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	cfg := Config{
		Endpoints: []Endpoint{
			{Method: "GET", Path: "/users", JsonPath: jsonFile, Status: 200},
			{Method: "POST", Path: "/users", JsonPath: jsonFile, Status: 201},
			{Method: "PUT", Path: "/users", JsonPath: jsonFile, Status: 200},
			{Method: "DELETE", Path: "/users", JsonPath: jsonFile, Status: 204},
		},
	}

	err = cfg.Validate()
	assert.NoError(t, err, "different HTTP methods on the same path should be allowed")
}

func TestConfig_Validate_MultipleEndpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multi-ep-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	cfg := Config{
		Endpoints: []Endpoint{
			{Method: "GET", Path: "/users", JsonPath: jsonFile, Status: 200},
			{Method: "GET", Path: "/posts", JsonPath: jsonFile, Status: 200},
			{Method: "GET", Path: "/comments", JsonPath: jsonFile, Status: 200},
		},
	}

	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_EndpointWithType(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "type-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	cfg := Config{
		Endpoints: []Endpoint{
			{Type: "api", Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
		},
	}

	err = cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Reload_InvalidConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "reload-invalid-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	// Create initial valid config
	configPath := filepath.Join(tempDir, "config.json")
	initialConfig := `{
		"port": 8080,
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)

	// Overwrite with invalid JSON
	err = os.WriteFile(configPath, []byte(`{invalid}`), 0644)
	assert.NoError(t, err)

	// Reload should fail
	err = cfg.Reload(configPath)
	assert.Error(t, err)
	// Original config should remain unchanged
	assert.Equal(t, 8080, cfg.Port)
}

func TestLoadConfig_WithLogPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logpath-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{}`), 0644)
	assert.NoError(t, err)

	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"port": 3000,
		"logLevel": "warn",
		"logFormat": "json",
		"logPath": "/tmp/test-server.log",
		"endpoints": [
			{
				"method": "GET",
				"status": 200,
				"path": "/test",
				"jsonPath": "` + jsonFile + `"
			}
		]
	}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "warn", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, "/tmp/test-server.log", cfg.LogPath)
}

func TestConfig_Validate_EmptyJsonPath(t *testing.T) {
	// Endpoint with no jsonPath and no folder should still pass validation
	// (jsonPath check is only for non-empty jsonPath)
	cfg := Config{
		Endpoints: []Endpoint{
			{Method: "POST", Path: "/webhook", Status: 200},
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "Valid config",
			config: Config{
				Endpoints: []Endpoint{
					{Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
				},
			},
			wantError: false,
		},
		{
			name: "Valid config with file server",
			config: Config{
				Endpoints: []Endpoint{
					{Path: "/static", Folder: testFolder},
				},
			},
			wantError: false,
		},
		{
			name: "No endpoints",
			config: Config{
				Endpoints: []Endpoint{},
			},
			wantError: true,
		},
		{
			name: "Empty path",
			config: Config{
				Endpoints: []Endpoint{
					{Method: "GET", Path: "", JsonPath: jsonFile, Status: 200},
				},
			},
			wantError: true,
		},
		{
			name: "Duplicate endpoint",
			config: Config{
				Endpoints: []Endpoint{
					{Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
					{Method: "GET", Path: "/test", JsonPath: jsonFile, Status: 200},
				},
			},
			wantError: true,
		},
		{
			name: "JSON file not found",
			config: Config{
				Endpoints: []Endpoint{
					{Method: "GET", Path: "/test", JsonPath: filepath.Join(tempDir, "notfound.json"), Status: 200},
				},
			},
			wantError: true,
		},
		{
			name: "Folder not found",
			config: Config{
				Endpoints: []Endpoint{
					{Path: "/static", Folder: filepath.Join(tempDir, "notfound")},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
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

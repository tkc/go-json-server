package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tkc/go-json-server/src/config"
	"github.com/tkc/go-json-server/src/logger"
)

func TestMainComponents(t *testing.T) {
	// This test doesn't run the main function itself but tests key components used by main

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "go-json-server-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test JSON file for the endpoint
	jsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(jsonFile, []byte(`{"message":"test"}`), 0644)
	assert.NoError(t, err)

	// Create a test config file
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"port": 8080,
		"logLevel": "info",
		"logFormat": "text",
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

	// Test loading config
	cfg, err := config.LoadConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)

	// Test creating logger
	logConfig := logger.LogConfig{
		Level:      logger.ParseLogLevel(cfg.LogLevel),
		Format:     logger.LogFormat(cfg.LogFormat),
		OutputPath: "stdout", // Use stdout for testing
	}
	log, err := logger.NewLogger(logConfig)
	assert.NoError(t, err)
	assert.NotNil(t, log)

	// Test config validation
	err = cfg.Validate()
	assert.NoError(t, err)
}

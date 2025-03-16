package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Error definitions
var (
	ErrNoEndpoints       = errors.New("no endpoints defined in configuration")
	ErrEmptyPath         = errors.New("endpoint with empty path found")
	ErrDuplicateEndpoint = errors.New("duplicate endpoint found")
	ErrJSONFileNotFound  = errors.New("JSON file not found for endpoint")
	ErrFolderNotFound    = errors.New("folder not found for endpoint")
)

// Endpoint represents a single API endpoint configuration
type Endpoint struct {
	Type     string `json:"type"`
	Method   string `json:"method"`
	Status   int    `json:"status"`
	Path     string `json:"path"`
	JsonPath string `json:"jsonPath"`
	Folder   string `json:"folder"`
}

// Config represents the main configuration structure
type Config struct {
	Host      string     `json:"host"`
	Port      int        `json:"port"`
	LogLevel  string     `json:"logLevel"`
	LogFormat string     `json:"logFormat"`
	LogPath   string     `json:"logPath"`
	Endpoints []Endpoint `json:"endpoints"`
	mu        sync.RWMutex
}

// LoadConfig loads configuration from a file path
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Default settings
	if config.Port == 0 {
		config.Port = 3000
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.LogFormat == "" {
		config.LogFormat = "text"
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return ErrNoEndpoints
	}

	// Check for duplicate paths and methods
	pathMethods := make(map[string]bool)
	for _, ep := range c.Endpoints {
		if ep.Path == "" {
			return fmt.Errorf("%w: empty path in endpoint", ErrEmptyPath)
		}

		// Skip method duplication check for file servers
		if ep.Folder != "" {
			// Check folder existence
			if _, err := os.Stat(ep.Folder); os.IsNotExist(err) {
				return fmt.Errorf("%w: %s for path %s", ErrFolderNotFound, ep.Folder, ep.Path)
			}
			continue
		}

		pathMethod := ep.Path + ":" + ep.Method
		if pathMethods[pathMethod] {
			return fmt.Errorf("%w: %s %s", ErrDuplicateEndpoint, ep.Method, ep.Path)
		}
		pathMethods[pathMethod] = true

		// Check JSON file existence
		if ep.JsonPath != "" && ep.Folder == "" {
			if _, err := os.Stat(ep.JsonPath); os.IsNotExist(err) {
				return fmt.Errorf("%w: %s for %s %s", ErrJSONFileNotFound, ep.JsonPath, ep.Method, ep.Path)
			}
		}
	}

	return nil
}

// Reload reloads the configuration from disk
func (c *Config) Reload(path string) error {
	newConfig, err := LoadConfig(path)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.Host = newConfig.Host
	c.Port = newConfig.Port
	c.LogLevel = newConfig.LogLevel
	c.LogFormat = newConfig.LogFormat
	c.LogPath = newConfig.LogPath
	c.Endpoints = newConfig.Endpoints

	return nil
}

// GetEndpoints returns a thread-safe copy of endpoints
func (c *Config) GetEndpoints() []Endpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to avoid race conditions
	endpoints := make([]Endpoint, len(c.Endpoints))
	copy(endpoints, c.Endpoints)

	return endpoints
}

// GetPort returns the port in a thread-safe manner
func (c *Config) GetPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Port
}

// GetHost returns the host in a thread-safe manner
func (c *Config) GetHost() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Host
}

// GetLogConfig returns logging configuration
func (c *Config) GetLogConfig() (level, format, path string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LogLevel, c.LogFormat, c.LogPath
}

// WatchConfig watches for changes in the config file and reloads when needed
func WatchConfig(configPath string, config *Config, reloadCh chan<- bool) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	go func() {
		defer watcher.Close()

		dir := filepath.Dir(configPath)
		if err := watcher.Add(dir); err != nil {
			fmt.Printf("Error watching directory %s: %v\n", dir, err)
			return
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Name == configPath && (event.Op&fsnotify.Write == fsnotify.Write) {
					// Add a small delay to wait for write completion
					time.Sleep(100 * time.Millisecond)

					fmt.Println("Config file changed, reloading...")
					if err := config.Reload(configPath); err != nil {
						fmt.Printf("Error reloading config: %v\n", err)
					} else if reloadCh != nil {
						reloadCh <- true
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Error watching config: %v\n", err)
			}
		}
	}()

	return nil
}

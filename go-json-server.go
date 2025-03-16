package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tkc/go-json-server/src/config"
	"github.com/tkc/go-json-server/src/handler"
	"github.com/tkc/go-json-server/src/logger"
	"github.com/tkc/go-json-server/src/middleware"
)

var (
	// Version of the application
	Version = "1.0.0"

	// Command line flags
	configPath = flag.String("config", "./api.json", "Path to the configuration file")
	port       = flag.Int("port", 0, "Server port (overrides config)")
	logLevel   = flag.String("log-level", "", "Log level: debug, info, warn, error, fatal (overrides config)")
	logFormat  = flag.String("log-format", "", "Log format: text, json (overrides config)")
	logPath    = flag.String("log-path", "", "Path to log file (overrides config)")
	cacheTTL   = flag.Int("cache-ttl", 300, "Cache TTL in seconds")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override configuration with command line flags
	if *port > 0 {
		cfg.Port = *port
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *logFormat != "" {
		cfg.LogFormat = *logFormat
	}
	if *logPath != "" {
		cfg.LogPath = *logPath
	}

	// Initialize logger
	logConfig := logger.LogConfig{
		Level:      logger.ParseLogLevel(cfg.LogLevel),
		Format:     logger.LogFormat(cfg.LogFormat),
		OutputPath: cfg.LogPath,
		TimeFormat: time.RFC3339,
	}

	log, err := logger.NewLogger(logConfig)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// Log startup information
	log.Info("Starting go-json-server", map[string]any{
		"version": Version,
		"port":    cfg.Port,
	})

	// Create server with response cache
	server := handler.NewServer(cfg, log, time.Duration(*cacheTTL)*time.Second)

	// Setup configuration hot-reloading
	reloadCh := make(chan bool)
	if err := config.WatchConfig(*configPath, cfg, reloadCh); err != nil {
		log.Error("Failed to watch config file", map[string]any{"error": err.Error()})
	}

	// Create HTTP server with middlewares
	srv := &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Port),
		Handler: middleware.Chain(
			middleware.RequestID(),
			middleware.Logger(log),
			middleware.CORS(),
			middleware.Timeout(30*time.Second),
			middleware.Recovery(log),
		)(http.HandlerFunc(server.HandleRequest)),
	}

	// Channel to listen for errors coming from the listener
	serverErrors := make(chan error, 1)

	// Start the server
	go func() {
		log.Info("Server listening", map[string]any{"address": srv.Addr})
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for config reload events
	go func() {
		for range reloadCh {
			log.Info("Configuration reloaded")

			// Clear the response cache when config changes
			server.ClearCache()
		}
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Error("Server error", map[string]any{"error": err.Error()})
		}

	case sig := <-shutdown:
		log.Info("Shutdown signal received", map[string]any{
			"signal": sig.String(),
		})

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Gracefully shutdown the server
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Graceful shutdown failed", map[string]any{"error": err.Error()})
			srv.Close()
		}
	}
}

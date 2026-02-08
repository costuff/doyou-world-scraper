package api

import (
	"doyou-world-scraper/internal/config"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TestPortConflicts tests handling of port conflicts
func TestPortConflicts(t *testing.T) {
	t.Run("Detect port already in use", func(t *testing.T) {
		// Start a listener on a random port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to start test listener: %v", err)
		}
		defer listener.Close()

		// Get the port that was allocated
		port := listener.Addr().(*net.TCPAddr).Port

		// Verify we can detect the port is in use
		testListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			testListener.Close()
			t.Error("Should fail to bind to already-used port")
		} else {
			t.Logf("Got expected error: %v", err)
		}
	})

	t.Run("Multiple listeners on different ports", func(t *testing.T) {
		// Start two listeners on random ports
		listener1, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to start first listener: %v", err)
		}
		defer listener1.Close()

		listener2, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to start second listener: %v", err)
		}
		defer listener2.Close()

		// Both should be on different ports
		port1 := listener1.Addr().(*net.TCPAddr).Port
		port2 := listener2.Addr().(*net.TCPAddr).Port

		if port1 == port2 {
			t.Error("Listeners should be on different ports")
		}
	})

	t.Run("Graceful handling of port conflict", func(t *testing.T) {
		// Occupy a port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to start listener: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		defer listener.Close()

		// Try to bind again - should get clear error
		_, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			t.Error("Should return error for port conflict")
		} else {
			// Error message should be informative
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("Error message should not be empty")
			}
			if !strings.Contains(errMsg, "address already in use") && !strings.Contains(errMsg, "bind") {
				t.Errorf("Error should mention address in use or bind, got: %v", errMsg)
			}
			t.Logf("Got expected error: %v", errMsg)
		}
	})
}

// TestServerStartup tests various server startup scenarios
func TestServerStartup(t *testing.T) {
	t.Run("Start server with valid configuration", func(t *testing.T) {
		cfg := &config.Config{
			DataDir:      t.TempDir(),
			CSVDir:       t.TempDir(),
			DownloadsDir: t.TempDir(),
			NASPath:      t.TempDir(),
			Port:         0, // Random port
		}

		app := setupTestAppForPortTests(cfg)

		// Start in goroutine
		done := make(chan error)
		go func() {
			done <- app.Listen(":0")
		}()

		// Give it time to start
		time.Sleep(100 * time.Millisecond)

		// Shutdown
		app.Shutdown()

		// Wait for shutdown
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("Server shutdown timeout")
		}
	})

	t.Run("Start with invalid data directory", func(t *testing.T) {
		cfg := &config.Config{
			DataDir:      "/nonexistent/path",
			CSVDir:       "/nonexistent/path/csv",
			DownloadsDir: "/nonexistent/path/downloads",
			NASPath:      "/nonexistent/nas",
			Port:         0,
		}

		// Validation should catch this
		err := cfg.Validate()
		if err == nil {
			t.Error("Should reject invalid directory configuration")
		}
	})

	t.Run("Start with missing directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			DataDir:      tmpDir,
			CSVDir:       tmpDir + "/csv",
			DownloadsDir: tmpDir + "/downloads",
			NASPath:      tmpDir,
			Port:         0,
		}

		// Should create missing directories
		err := cfg.EnsureDirectories()
		if err != nil {
			t.Errorf("Should create missing directories: %v", err)
		}

		// Verify they exist
		for _, dir := range []string{cfg.CSVDir, cfg.DownloadsDir} {
			if !fileExists(dir) {
				t.Errorf("Directory should exist: %s", dir)
			}
		}
	})
}

// TestEnvironmentVariables tests configuration from environment
func TestEnvironmentVariables(t *testing.T) {
	t.Run("Port from environment", func(t *testing.T) {
		t.Setenv("PORT", "8080")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if cfg.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", cfg.Port)
		}
	})

	t.Run("Data directory from environment", func(t *testing.T) {
		testDir := "/tmp/test-data"
		t.Setenv("DATA_DIR", testDir)

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if cfg.DataDir != testDir {
			t.Errorf("Expected data dir %s, got %s", testDir, cfg.DataDir)
		}
	})

	t.Run("Default values when env vars not set", func(t *testing.T) {
		// Clear env vars
		t.Setenv("PORT", "")
		t.Setenv("DATA_DIR", "")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Should have defaults
		if cfg.Port == 0 {
			t.Error("Should have default port")
		}
		if cfg.DataDir == "" {
			t.Error("Should have default data directory")
		}
	})
}

// Helper to set up test app for port tests
func setupTestAppForPortTests(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	handlers := NewHandlers(cfg)

	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "API running",
		})
	})

	// Add a simple test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("test")
	})

	_ = handlers // Use handlers to avoid "declared but not used" error

	return app
}

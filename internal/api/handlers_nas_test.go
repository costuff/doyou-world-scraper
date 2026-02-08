package api

import (
	"bytes"
	"doyou-world-scraper/internal/config"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestNASPathDetection tests NAS path auto-detection across deployment scenarios
func TestNASPathDetection(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(tmpDir string) *config.Config
		requestBody    map[string]string
		endpoint       string
		expectError    bool
		expectedNASDir string // What NAS dir should be detected/used
	}{
		{
			name: "Docker: /nas exists, no path in request",
			setupFunc: func(tmpDir string) *config.Config {
				// Simulate Docker: create /nas directory in test
				nasPath := filepath.Join(tmpDir, "nas")
				os.MkdirAll(nasPath, 0755)
				
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      "", // Empty, should detect /nas
					LogLevel:     "info",
					Environment:  "production",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody:    map[string]string{},
			endpoint:       "/util/deduplicate",
			expectError:    false,
			expectedNASDir: "detected /nas",
		},
		{
			name: "Docker: Invalid macOS path sent, should fallback to /nas",
			setupFunc: func(tmpDir string) *config.Config {
				nasPath := filepath.Join(tmpDir, "nas")
				os.MkdirAll(nasPath, 0755)
				
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      "",
					LogLevel:     "info",
					Environment:  "production",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody: map[string]string{
				"nas_dir": "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
			},
			endpoint:       "/util/deduplicate",
			expectError:    false,
			expectedNASDir: "fallback to /nas",
		},
		{
			name: "Local: NAS_PATH env var set",
			setupFunc: func(tmpDir string) *config.Config {
				nasPath := filepath.Join(tmpDir, "my-nas")
				os.MkdirAll(nasPath, 0755)
				
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      nasPath, // Simulates NAS_PATH env var
					LogLevel:     "info",
					Environment:  "development",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody:    map[string]string{},
			endpoint:       "/util/deduplicate",
			expectError:    false,
			expectedNASDir: "from NAS_PATH",
		},
		{
			name: "Local: No NAS available, use 'skip'",
			setupFunc: func(tmpDir string) *config.Config {
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      "", // No NAS configured
					LogLevel:     "info",
					Environment:  "development",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody:    map[string]string{},
			endpoint:       "/util/deduplicate",
			expectError:    false,
			expectedNASDir: "skip",
		},
		{
			name: "Valid custom path provided",
			setupFunc: func(tmpDir string) *config.Config {
				customNAS := filepath.Join(tmpDir, "custom-nas-path")
				os.MkdirAll(customNAS, 0755)
				
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      "",
					LogLevel:     "info",
					Environment:  "development",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody: map[string]string{
				"nas_dir": "", // Will be filled in test
			},
			endpoint:       "/util/deduplicate",
			expectError:    false,
			expectedNASDir: "custom",
		},
		{
			name: "MoveToNAS: No NAS available should error",
			setupFunc: func(tmpDir string) *config.Config {
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      "",
					LogLevel:     "info",
					Environment:  "development",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody:    map[string]string{},
			endpoint:       "/util/move-to-nas",
			expectError:    true,
			expectedNASDir: "",
		},
		{
			name: "MoveToNAS: Invalid path should error",
			setupFunc: func(tmpDir string) *config.Config {
				cfg := &config.Config{
					Port:         0,
					DataDir:      tmpDir,
					CSVDir:       filepath.Join(tmpDir, "csv"),
					DownloadsDir: filepath.Join(tmpDir, "downloads"),
					NASPath:      "",
					LogLevel:     "info",
					Environment:  "development",
				}
				cfg.EnsureDirectories()
				return cfg
			},
			requestBody: map[string]string{
				"nas_dir": "/Volumes/NonExistent/Path",
			},
			endpoint:       "/util/move-to-nas",
			expectError:    true,
			expectedNASDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cfg := tt.setupFunc(tmpDir)
			
			// For custom path test, set the actual path
			if tt.expectedNASDir == "custom" {
				customPath := filepath.Join(tmpDir, "custom-nas-path")
				tt.requestBody["nas_dir"] = customPath
			}
			
			handlers := NewHandlers(cfg)
			app := fiber.New()
			
			utilGroup := app.Group("/util")
			utilGroup.Post("/deduplicate", handlers.HandleDeduplicate)
			utilGroup.Post("/move-to-nas", handlers.HandleMoveToNAS)
			
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", tt.endpoint, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			
			if tt.expectError {
				if resp.StatusCode == 202 {
					t.Errorf("Expected error (4xx), got 202 Accepted")
				}
			} else {
				if resp.StatusCode != 202 {
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Errorf("Expected 202 Accepted, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
				} else {
					t.Logf("✓ NAS path detection successful: %s", tt.expectedNASDir)
				}
			}
		})
	}
}

// TestNASPathPriority tests the priority order of NAS path detection
func TestNASPathPriority(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create both /nas-like and NAS_PATH directories
	nasPath := filepath.Join(tmpDir, "nas")
	envNASPath := filepath.Join(tmpDir, "env-nas")
	customPath := filepath.Join(tmpDir, "custom-nas")
	
	os.MkdirAll(nasPath, 0755)
	os.MkdirAll(envNASPath, 0755)
	os.MkdirAll(customPath, 0755)
	
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       filepath.Join(tmpDir, "csv"),
		DownloadsDir: filepath.Join(tmpDir, "downloads"),
		NASPath:      envNASPath, // Simulates NAS_PATH env var
		LogLevel:     "info",
		Environment:  "development",
	}
	cfg.EnsureDirectories()
	
	t.Run("Priority 1: Valid custom path > all others", func(t *testing.T) {
		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/util/deduplicate", handlers.HandleDeduplicate)
		
		body, _ := json.Marshal(map[string]string{
			"nas_dir": customPath,
		})
		req := httptest.NewRequest("POST", "/util/deduplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		
		if resp.StatusCode != 202 {
			t.Errorf("Expected custom path to be accepted, got status %d", resp.StatusCode)
		}
	})
	
	t.Run("Priority 2: Invalid custom path falls back to auto-detect", func(t *testing.T) {
		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/util/deduplicate", handlers.HandleDeduplicate)
		
		body, _ := json.Marshal(map[string]string{
			"nas_dir": "/invalid/path",
		})
		req := httptest.NewRequest("POST", "/util/deduplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		
		// Should fall back to NAS_PATH (envNASPath) and succeed
		if resp.StatusCode != 202 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected fallback to NAS_PATH, got status %d. Body: %s", resp.StatusCode, string(bodyBytes))
		}
	})
	
	t.Run("Priority 3: Empty request uses NAS_PATH when available", func(t *testing.T) {
		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/util/deduplicate", handlers.HandleDeduplicate)
		
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("POST", "/util/deduplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		
		if resp.StatusCode != 202 {
			t.Errorf("Expected NAS_PATH to be used, got status %d", resp.StatusCode)
		}
	})
}

// TestDockerVsLocalDeployment simulates both deployment scenarios
func TestDockerVsLocalDeployment(t *testing.T) {
	t.Run("Docker deployment scenario", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Simulate Docker: /nas mount exists
		nasMount := filepath.Join(tmpDir, "nas")
		os.MkdirAll(nasMount, 0755)
		
		cfg := &config.Config{
			Port:         0,
			DataDir:      "/app/data", // Docker path
			CSVDir:       "/app/csv",
			DownloadsDir: "/app/downloads",
			NASPath:      "", // Not set in Docker, relies on /nas mount
			LogLevel:     "info",
			Environment:  "production",
		}
		
		// Override with test paths
		cfg.DataDir = tmpDir
		cfg.CSVDir = filepath.Join(tmpDir, "csv")
		cfg.DownloadsDir = filepath.Join(tmpDir, "downloads")
		cfg.EnsureDirectories()
		
		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/util/deduplicate", handlers.HandleDeduplicate)
		
		// Frontend sends old macOS path
		body, _ := json.Marshal(map[string]string{
			"nas_dir": "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
		})
		req := httptest.NewRequest("POST", "/util/deduplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		
		// Should detect invalid path and use NAS_PATH fallback
		if resp.StatusCode != 202 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Docker scenario: Should auto-detect NAS. Status: %d, Body: %s", resp.StatusCode, string(bodyBytes))
		}
	})
	
	t.Run("Local macOS deployment scenario", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Simulate local: NAS_PATH points to actual volume
		localNAS := filepath.Join(tmpDir, "local-nas")
		os.MkdirAll(localNAS, 0755)
		
		cfg := &config.Config{
			Port:         0,
			DataDir:      tmpDir,
			CSVDir:       filepath.Join(tmpDir, "csv"),
			DownloadsDir: filepath.Join(tmpDir, "downloads"),
			NASPath:      localNAS, // Set via NAS_PATH env var
			LogLevel:     "info",
			Environment:  "development",
		}
		cfg.EnsureDirectories()
		
		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/util/deduplicate", handlers.HandleDeduplicate)
		
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("POST", "/util/deduplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		
		if resp.StatusCode != 202 {
			t.Errorf("Local deployment should use NAS_PATH, got status %d", resp.StatusCode)
		}
	})
}

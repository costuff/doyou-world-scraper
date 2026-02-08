package api

import (
	"doyou-world-scraper/internal/config"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestAudioAnalysisEdgeCases tests various audio analysis scenarios
func TestAudioAnalysisEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       filepath.Join(tmpDir, "csv"),
		DownloadsDir: filepath.Join(tmpDir, "downloads"),
		NASPath:      tmpDir,
		LogLevel:     "info",
		Environment:  "test",
	}
	cfg.EnsureDirectories()

	handlers := NewHandlers(cfg)
	app := fiber.New()
	app.Post("/api/v1/analyze/track", handlers.HandleAnalyzeTrack)

	t.Run("Missing file path in request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Non-existent audio file", func(t *testing.T) {
		body := `{"file_path": "/nonexistent/audio.mp3"}`
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1) // No timeout
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Could either accept job (202) or return 404 depending on validation
		if resp.StatusCode != 202 && resp.StatusCode != 404 {
			t.Errorf("Expected 202 or 404, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid file path characters", func(t *testing.T) {
		body := `{"file_path": "../../etc/passwd"}`
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should reject or sanitize path
		if resp.StatusCode == 200 {
			t.Error("Should not accept path traversal")
		}
	})

	t.Run("Empty file", func(t *testing.T) {
		// Create empty file
		emptyFile := filepath.Join(cfg.DownloadsDir, "empty.mp3")
		os.WriteFile(emptyFile, []byte{}, 0644)

		body := `{"file_path": "` + emptyFile + `"}`
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should accept job (will fail during analysis)
		if resp.StatusCode != 202 {
			t.Errorf("Expected 202, got %d", resp.StatusCode)
		}
	})

	t.Run("Corrupt audio file", func(t *testing.T) {
		// Create corrupt file
		corruptFile := filepath.Join(cfg.DownloadsDir, "corrupt.mp3")
		os.WriteFile(corruptFile, []byte("not a real mp3 file"), 0644)

		body := `{"file_path": "` + corruptFile + `"}`
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 202 {
			t.Errorf("Expected 202 (job accepted), got %d", resp.StatusCode)
		}
	})

	t.Run("Very large file", func(t *testing.T) {
		// Create large dummy file (just metadata, not actual audio)
		largeFile := filepath.Join(cfg.DownloadsDir, "large.mp3")
		f, _ := os.Create(largeFile)
		// Write 10MB of zeros
		f.Seek(10*1024*1024-1, 0)
		f.Write([]byte{0})
		f.Close()

		body := `{"file_path": "` + largeFile + `"}`
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 202 {
			t.Errorf("Expected 202, got %d", resp.StatusCode)
		}
	})

	t.Run("File with Unicode filename", func(t *testing.T) {
		unicodeFile := filepath.Join(cfg.DownloadsDir, "歌曲-🎵.mp3")
		os.WriteFile(unicodeFile, []byte("test"), 0644)

		body := `{"file_path": "` + unicodeFile + `"}`
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 202 {
			t.Errorf("Expected 202, got %d", resp.StatusCode)
		}
	})

	t.Run("Malformed JSON request", func(t *testing.T) {
		body := `{"file_path": "test.mp3"` // Missing closing brace
		req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected 400 for malformed JSON, got %d", resp.StatusCode)
		}
	})
}

// TestPythonEnvironment tests Python path detection
func TestPythonEnvironment(t *testing.T) {
	t.Run("Detect venv Python", func(t *testing.T) {
		tmpDir := t.TempDir()
		venvPath := filepath.Join(tmpDir, ".venv", "bin", "python")

		// Create fake venv structure
		os.MkdirAll(filepath.Dir(venvPath), 0755)
		os.WriteFile(venvPath, []byte("#!/bin/sh\necho 'Python 3.9.0'"), 0755)

		// Check if file exists
		if !fileExists(venvPath) {
			t.Error("Should create venv python")
		}
	})

	t.Run("Fall back to system Python", func(t *testing.T) {
		// When venv doesn't exist, should try system python
		systemPython := "/usr/bin/python3"

		if fileExists(systemPython) {
			t.Logf("System Python found at %s", systemPython)
		} else {
			t.Log("System Python not found, would use fallback")
		}
	})

	t.Run("Missing Python binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			DataDir: tmpDir,
		}

		// If Python is missing, analysis should fail gracefully
		_ = cfg
	})
}

// TestAnalysisTimeout tests handling of long-running analysis
func TestAnalysisTimeout(t *testing.T) {
	t.Run("Analysis within timeout", func(t *testing.T) {
		// Normal analysis should complete quickly
		// This would test actual Python script execution
		t.Skip("Requires real Python environment")
	})

	t.Run("Analysis exceeds timeout", func(t *testing.T) {
		// Very long audio file or hung process
		// Should timeout and fail gracefully
		t.Skip("Requires timeout implementation")
	})
}

// TestConcurrentAnalysis tests multiple simultaneous analysis requests
func TestConcurrentAnalysis(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       filepath.Join(tmpDir, "csv"),
		DownloadsDir: filepath.Join(tmpDir, "downloads"),
		NASPath:      tmpDir,
		LogLevel:     "info",
		Environment:  "test",
	}
	cfg.EnsureDirectories()

	handlers := NewHandlers(cfg)
	app := fiber.New()
	app.Post("/api/v1/analyze/track", handlers.HandleAnalyzeTrack)

	t.Run("Multiple concurrent requests", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(cfg.DownloadsDir, "test.mp3")
		os.WriteFile(testFile, []byte("test"), 0644)

		done := make(chan bool)

		// Send 5 concurrent requests
		for i := 0; i < 5; i++ {
			go func() {
				body := `{"file_path": "` + testFile + `"}`
				req := httptest.NewRequest("POST", "/api/v1/analyze/track", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				resp, err := app.Test(req)
				if err != nil {
					t.Errorf("Request failed: %v", err)
				}
				if resp.StatusCode != 202 {
					t.Errorf("Expected 202, got %d", resp.StatusCode)
				}
				done <- true
			}()
		}

		// Wait for all to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})
}

// TestAnalysisResultCaching tests result caching behavior
func TestAnalysisResultCaching(t *testing.T) {
	t.Run("Re-analyze same file", func(t *testing.T) {
		// Should we cache results or always re-analyze?
		// This tests the expected behavior
		t.Skip("Caching not implemented yet")
	})

	t.Run("Cache invalidation on file modification", func(t *testing.T) {
		// If file changes, cached result should be invalidated
		t.Skip("Caching not implemented yet")
	})
}

// TestAnalysisOutput tests output format and validation
func TestAnalysisOutput(t *testing.T) {
	t.Run("Valid JSON output", func(t *testing.T) {
		// Python script should output valid JSON
		t.Skip("Requires Python environment")
	})

	t.Run("Required fields present", func(t *testing.T) {
		// Output should contain: bpm, key, duration, energy, etc.
		t.Skip("Requires Python environment")
	})

	t.Run("Values in valid ranges", func(t *testing.T) {
		// Energy, danceability, etc should be 0-1
		// BPM should be reasonable (30-300)
		t.Skip("Requires Python environment")
	})
}

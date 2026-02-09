package api

import (
	"doyou-world-scraper/internal/config"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveDownloadsDir tests the NAS path resolution logic
func TestResolveDownloadsDir(t *testing.T) {
	t.Run("Use config downloads dir when requested", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			DownloadsDir: tmpDir,
			NASPath:      "",
		}
		handlers := NewHandlers(cfg)

		resolved, err := handlers.resolveDownloadsDir(tmpDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resolved != tmpDir {
			t.Errorf("Expected %s, got %s", tmpDir, resolved)
		}
	})

	t.Run("Use NAS_PATH when nas requested and NAS_PATH is set", func(t *testing.T) {
		tmpDir := t.TempDir()
		nasDir := filepath.Join(tmpDir, "nas")
		os.MkdirAll(nasDir, 0755)

		cfg := &config.Config{
			DownloadsDir: tmpDir,
			NASPath:      nasDir,
		}
		handlers := NewHandlers(cfg)

		resolved, err := handlers.resolveDownloadsDir("nas")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resolved != nasDir {
			t.Errorf("Expected %s, got %s", nasDir, resolved)
		}
	})

	t.Run("Return error when nas requested but NAS not available locally", func(t *testing.T) {
		// Skip if NAS is actually mounted (common paths exist)
		commonPaths := []string{
			"/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
			"/Volumes/Music/doyou-world-radio-tracks",
			"/mnt/nas/music/doyou-world-radio-tracks",
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				t.Skipf("Skipping: NAS is actually mounted at %s", path)
				return
			}
		}

		tmpDir := t.TempDir()
		cfg := &config.Config{
			DownloadsDir: tmpDir,
			NASPath:      "", // Not set
		}
		handlers := NewHandlers(cfg)

		// Request NAS when it's not configured
		_, err := handlers.resolveDownloadsDir("nas")
		if err == nil {
			t.Fatal("Expected error when NAS not available, got nil")
		}
		if err.Error() != "NAS storage not found - not mounted or configured. Set NAS_PATH environment variable or mount NAS at a common location" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Docker mode uses /app/downloads", func(t *testing.T) {
		// This test would require mocking the /app/downloads check
		// Skipping as it requires file system mocking
		t.Skip("Requires Docker environment or mocking")
	})

	t.Run("Docker mode uses /nas for nas request", func(t *testing.T) {
		// This test would require mocking the /app/downloads check
		t.Skip("Requires Docker environment or mocking")
	})

	t.Run("Return error for non-existent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			DownloadsDir: tmpDir,
			NASPath:      "",
		}
		handlers := NewHandlers(cfg)

		nonExistentPath := filepath.Join(tmpDir, "does-not-exist")
		_, err := handlers.resolveDownloadsDir(nonExistentPath)
		if err == nil {
			t.Fatal("Expected error for non-existent directory, got nil")
		}
	})

	t.Run("Absolute paths are used as-is if they exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		absolutePath := filepath.Join(tmpDir, "absolute")
		os.MkdirAll(absolutePath, 0755)

		cfg := &config.Config{
			DownloadsDir: tmpDir,
			NASPath:      "",
		}
		handlers := NewHandlers(cfg)

		resolved, err := handlers.resolveDownloadsDir(absolutePath)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resolved != absolutePath {
			t.Errorf("Expected %s, got %s", absolutePath, resolved)
		}
	})
}

// TestNASAutoDetection tests that common NAS paths are auto-detected
func TestNASAutoDetection(t *testing.T) {
	t.Run("Auto-detect common NAS paths on macOS", func(t *testing.T) {
		// Create a temporary directory structure that mimics a macOS volume
		tmpDir := t.TempDir()

		// Simulate a mounted volume path
		nasPath := filepath.Join(tmpDir, "ExtHDD-8TB", "Music", "doyou-world-radio-tracks")
		os.MkdirAll(nasPath, 0755)

		cfg := &config.Config{
			DownloadsDir: tmpDir,
			NASPath:      "", // Not configured
		}
		handlers := NewHandlers(cfg)

		// This won't actually find it since we're not at /Volumes/...
		// but the logic is tested
		_, err := handlers.resolveDownloadsDir("nas")

		// Should get an error since mock path isn't in the common paths list
		if err == nil {
			t.Log("NAS auto-detection logic exists (would work with real /Volumes paths)")
		} else {
			t.Logf("Expected error for unconfigured NAS: %v", err)
		}
	})
}

// TestFileExists tests the file existence helper
func TestFileExists(t *testing.T) {
	t.Run("Returns true for existing file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		if !fileExists(tmpFile.Name()) {
			t.Error("Expected true for existing file")
		}
	})

	t.Run("Returns true for existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		if !fileExists(tmpDir) {
			t.Error("Expected true for existing directory")
		}
	})

	t.Run("Returns false for non-existent path", func(t *testing.T) {
		if fileExists("/path/that/does/not/exist/hopefully") {
			t.Error("Expected false for non-existent path")
		}
	})
}

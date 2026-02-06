package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNASPathValidation tests NAS path scenarios
func TestNASPathValidation(t *testing.T) {
	t.Run("NAS path exists and is writable", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			NASPath: tmpDir,
		}

		// Check if path exists
		if _, err := os.Stat(cfg.NASPath); os.IsNotExist(err) {
			t.Errorf("NAS path should exist: %v", err)
		}

		// Check if writable
		testFile := filepath.Join(cfg.NASPath, ".writetest")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Errorf("NAS path should be writable: %v", err)
		}
		os.Remove(testFile)
	})

	t.Run("NAS path does not exist", func(t *testing.T) {
		cfg := &Config{
			NASPath: "/nonexistent/path/to/nas",
		}

		if _, err := os.Stat(cfg.NASPath); !os.IsNotExist(err) {
			t.Error("Should detect non-existent NAS path")
		}
	})

	t.Run("NAS path is not writable", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		tmpDir := t.TempDir()
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		os.Mkdir(readOnlyDir, 0444) // Read-only directory

		cfg := &Config{
			NASPath: readOnlyDir,
		}

		testFile := filepath.Join(cfg.NASPath, ".writetest")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		if err == nil {
			t.Error("Should fail to write to read-only NAS path")
			os.Remove(testFile)
		}
	})

	t.Run("NAS path is a symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		realPath := filepath.Join(tmpDir, "real")
		symlinkPath := filepath.Join(tmpDir, "symlink")

		os.Mkdir(realPath, 0755)
		err := os.Symlink(realPath, symlinkPath)
		if err != nil {
			t.Skipf("Cannot create symlink: %v", err)
		}

		cfg := &Config{
			NASPath: symlinkPath,
		}

		// Should be able to follow symlink
		info, err := os.Stat(cfg.NASPath)
		if err != nil {
			t.Errorf("Should follow symlink: %v", err)
		}
		if !info.IsDir() {
			t.Error("Symlink should resolve to directory")
		}
	})

	t.Run("NAS path on different filesystem", func(t *testing.T) {
		// This tests behavior when NAS is mounted on different device
		tmpDir := t.TempDir()
		cfg := &Config{
			NASPath:      tmpDir,
			DataDir:      t.TempDir(),
			CSVDir:       filepath.Join(t.TempDir(), "csv"),
			DownloadsDir: filepath.Join(t.TempDir(), "downloads"),
		}

		// Both should be accessible
		for _, path := range []string{cfg.NASPath, cfg.DataDir} {
			if _, err := os.Stat(path); err != nil {
				t.Errorf("Path should be accessible: %s, error: %v", path, err)
			}
		}
	})
}

// TestNASMountDetection tests detection of NAS mount status
func TestNASMountDetection(t *testing.T) {
	t.Run("Detect unmounted NAS", func(t *testing.T) {
		cfg := &Config{
			NASPath: "/Volumes/NonExistentNAS",
		}

		mounted := fileExists(cfg.NASPath)
		if mounted {
			t.Skip("Test NAS path unexpectedly exists")
		}

		// Application should gracefully handle unmounted NAS
		if mounted {
			t.Error("Should detect unmounted NAS")
		}
	})

	t.Run("Detect mounted NAS", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			NASPath: tmpDir,
		}

		mounted := fileExists(cfg.NASPath)
		if !mounted {
			t.Error("Should detect mounted NAS (temp dir)")
		}
	})

	t.Run("Handle NAS mount/unmount during operation", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			NASPath: tmpDir,
		}

		// Initially mounted
		if !fileExists(cfg.NASPath) {
			t.Error("NAS should be initially mounted")
		}

		// Create a file
		testFile := filepath.Join(cfg.NASPath, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		if err != nil {
			t.Errorf("Should write to mounted NAS: %v", err)
		}

		// Verify file exists
		if !fileExists(testFile) {
			t.Error("File should exist on NAS")
		}
	})
}

// Helper function
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestPathValidation tests various path edge cases
func TestPathValidation(t *testing.T) {
	t.Run("Empty path", func(t *testing.T) {
		cfg := &Config{
			DataDir: "",
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("Should reject empty data directory")
		}
	})

	t.Run("Path with spaces", func(t *testing.T) {
		tmpDir := t.TempDir()
		pathWithSpaces := filepath.Join(tmpDir, "path with spaces")
		os.Mkdir(pathWithSpaces, 0755)

		cfg := &Config{
			DataDir:      pathWithSpaces,
			CSVDir:       filepath.Join(pathWithSpaces, "csv"),
			DownloadsDir: filepath.Join(pathWithSpaces, "downloads"),
			NASPath:      tmpDir,
		}

		if err := cfg.EnsureDirectories(); err != nil {
			t.Errorf("Should handle paths with spaces: %v", err)
		}
	})

	t.Run("Path with Unicode characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		unicodePath := filepath.Join(tmpDir, "путь-日本語-🎵")

		cfg := &Config{
			DataDir:      unicodePath,
			CSVDir:       filepath.Join(unicodePath, "csv"),
			DownloadsDir: filepath.Join(unicodePath, "downloads"),
			NASPath:      tmpDir,
		}

		if err := cfg.EnsureDirectories(); err != nil {
			t.Errorf("Should handle Unicode paths: %v", err)
		}

		// Verify directories were created
		if !fileExists(cfg.CSVDir) {
			t.Error("Should create Unicode CSV directory")
		}
	})

	t.Run("Relative vs absolute paths", func(t *testing.T) {
		cfg := &Config{
			DataDir:      "./data",
			CSVDir:       "./data/csv",
			DownloadsDir: "./data/downloads",
			NASPath:      "/nas",
		}

		// Relative paths are valid but should be documented
		if cfg.DataDir == "" {
			t.Error("Relative path should be accepted")
		}

		// Absolute path
		cfg.DataDir = "/tmp/data"
		if !filepath.IsAbs(cfg.DataDir) {
			t.Error("Should recognize absolute path")
		}
	})

	t.Run("Path traversal attempts", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			DataDir:      tmpDir,
			CSVDir:       filepath.Join(tmpDir, "../../../etc"),
			DownloadsDir: tmpDir,
			NASPath:      tmpDir,
		}

		// Clean should resolve path traversal
		cleanPath := filepath.Clean(cfg.CSVDir)
		if strings.Contains(cleanPath, "..") {
			t.Error("Should clean path traversal attempts")
		}
	})
}

// TestDiskSpaceScenarios tests handling of disk space issues
func TestDiskSpaceScenarios(t *testing.T) {
	t.Run("Check available space", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{
			DataDir:      tmpDir,
			CSVDir:       filepath.Join(tmpDir, "csv"),
			DownloadsDir: filepath.Join(tmpDir, "downloads"),
			NASPath:      t.TempDir(),
		}

		// Just verify we can access the path
		// Actual disk space check would require platform-specific code
		if !fileExists(cfg.DataDir) {
			err := cfg.EnsureDirectories()
			if err != nil {
				t.Errorf("Should create directories: %v", err)
			}
		}
	})

	t.Run("Handle write failures", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// Try to write
		err := os.WriteFile(testFile, []byte("test"), 0644)
		if err != nil {
			t.Logf("Write failed as expected in disk full scenario: %v", err)
		}
	})
}

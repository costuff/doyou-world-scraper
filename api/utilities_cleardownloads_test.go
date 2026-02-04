package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearDownloads(t *testing.T) {
	t.Run("successfully clears all MP3 files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test MP3 files
		files := []string{"track1.mp3", "track2.mp3", "track3.mp3"}
		for _, file := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test mp3 data"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		// Create non-MP3 files that should not be deleted
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("do not delete"), 0644); err != nil {
			t.Fatalf("Failed to create txt file: %v", err)
		}

		result, err := ClearDownloads(tmpDir)
		if err != nil {
			t.Fatalf("ClearDownloads failed: %v", err)
		}

		if result.TotalFiles != 3 {
			t.Errorf("Expected 3 total files, got %d", result.TotalFiles)
		}
		if result.DeletedFiles != 3 {
			t.Errorf("Expected 3 deleted files, got %d", result.DeletedFiles)
		}
		if len(result.Errors) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
		}

		// Verify MP3 files are deleted
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(tmpDir, file)); !os.IsNotExist(err) {
				t.Errorf("MP3 file %s should be deleted", file)
			}
		}

		// Verify non-MP3 file still exists
		if _, err := os.Stat(filepath.Join(tmpDir, "readme.txt")); os.IsNotExist(err) {
			t.Error("Non-MP3 file should not be deleted")
		}
	})

	t.Run("handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := ClearDownloads(tmpDir)
		if err != nil {
			t.Fatalf("ClearDownloads failed: %v", err)
		}

		if result.TotalFiles != 0 {
			t.Errorf("Expected 0 total files, got %d", result.TotalFiles)
		}
		if result.DeletedFiles != 0 {
			t.Errorf("Expected 0 deleted files, got %d", result.DeletedFiles)
		}
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		_, err := ClearDownloads("/nonexistent/directory")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})

	t.Run("handles nested directories with MP3 files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create nested structure
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		// Create MP3 in root and subdirectory
		files := []string{
			filepath.Join(tmpDir, "root.mp3"),
			filepath.Join(subDir, "nested.mp3"),
		}
		for _, file := range files {
			if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}
		}

		result, err := ClearDownloads(tmpDir)
		if err != nil {
			t.Fatalf("ClearDownloads failed: %v", err)
		}

		if result.TotalFiles != 2 {
			t.Errorf("Expected 2 total files, got %d", result.TotalFiles)
		}
		if result.DeletedFiles != 2 {
			t.Errorf("Expected 2 deleted files, got %d", result.DeletedFiles)
		}
	})

	t.Run("handles case-insensitive MP3 extensions", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create files with different case extensions
		files := []string{"track1.mp3", "track2.MP3", "track3.Mp3"}
		for _, file := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		result, err := ClearDownloads(tmpDir)
		if err != nil {
			t.Fatalf("ClearDownloads failed: %v", err)
		}

		if result.TotalFiles != 3 {
			t.Errorf("Expected 3 total files, got %d", result.TotalFiles)
		}
		if result.DeletedFiles != 3 {
			t.Errorf("Expected 3 deleted files, got %d", result.DeletedFiles)
		}
	})
}

func TestClearDownloadsEndpoint(t *testing.T) {
	app := setupTestApp()

	t.Run("successfully starts clear downloads job", func(t *testing.T) {
		// Create test downloads directory
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.mp3")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Note: The endpoint uses hardcoded paths, so we can't easily test with temp dir
		// But we can verify the endpoint structure works
		req := createTestRequest("POST", "/api/v1/util/clear-downloads", nil)
		resp := performRequest(app, req)

		if resp.StatusCode != 202 {
			t.Errorf("Expected status 202, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		parseResponse(resp, &result)

		if result["message"] != "Clearing downloaded files" {
			t.Errorf("Unexpected message: %v", result["message"])
		}

		job := result["job"].(map[string]interface{})
		if job["type"] != "clear_downloads" {
			t.Errorf("Expected job type 'clear_downloads', got %v", job["type"])
		}
		if job["status"] != "running" {
			t.Errorf("Expected status 'running', got %v", job["status"])
		}
	})
}

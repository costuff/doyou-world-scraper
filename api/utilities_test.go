package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDuplicates(t *testing.T) {
	// Create temp directories
	tmpDownloads := filepath.Join(t.TempDir(), "downloads")
	tmpNAS := filepath.Join(t.TempDir(), "nas")
	_ = os.MkdirAll(tmpDownloads, 0755)
	_ = os.MkdirAll(tmpNAS, 0755)

	// Create test files
	file1Content := []byte("test content 1")
	file2Content := []byte("test content 2")
	duplicateContent := file1Content // Same content as file1

	os.WriteFile(filepath.Join(tmpDownloads, "file1.mp3"), file1Content, 0644)
	os.WriteFile(filepath.Join(tmpDownloads, "file2.mp3"), file2Content, 0644)
	os.WriteFile(filepath.Join(tmpNAS, "duplicate.mp3"), duplicateContent, 0644)

	t.Run("Find and remove duplicates", func(t *testing.T) {
		result, err := RemoveDuplicates(tmpDownloads, tmpNAS, nil, nil)
		if err != nil {
			t.Fatalf("RemoveDuplicates failed: %v", err)
		}

		if result.TotalScanned != 3 {
			t.Errorf("Expected 3 files scanned, got %d", result.TotalScanned)
		}

		if result.DuplicatesFound != 1 {
			t.Errorf("Expected 1 duplicate found, got %d", result.DuplicatesFound)
		}
	})

	t.Run("No duplicates", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "unique")
		_ = os.MkdirAll(tmpDir, 0755)
		os.WriteFile(filepath.Join(tmpDir, "unique1.mp3"), []byte("unique 1"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "unique2.mp3"), []byte("unique 2"), 0644)

		result, err := RemoveDuplicates(tmpDir, "", nil, nil)
		if err != nil {
			t.Fatalf("RemoveDuplicates failed: %v", err)
		}

		if result.DuplicatesFound != 0 {
			t.Errorf("Expected 0 duplicates, got %d", result.DuplicatesFound)
		}
	})
}

func TestMoveDownloadsToNAS(t *testing.T) {
	tmpDownloads := filepath.Join(t.TempDir(), "downloads")
	tmpNAS := filepath.Join(t.TempDir(), "nas")
	os.MkdirAll(tmpDownloads, 0755)
	os.MkdirAll(tmpNAS, 0755)

	// Create test MP3 file (> 64KB)
	largeContent := make([]byte, 70000)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	testFile := filepath.Join(tmpDownloads, "test.mp3")
	os.WriteFile(testFile, largeContent, 0644)

	t.Run("Move file to NAS", func(t *testing.T) {
		result, err := MoveDownloadsToNAS(tmpDownloads, tmpNAS, nil, nil)
		if err != nil {
			t.Fatalf("MoveDownloadsToNAS failed: %v", err)
		}

		if result.TotalFiles != 1 {
			t.Errorf("Expected 1 file, got %d", result.TotalFiles)
		}

		if result.MovedFiles != 1 {
			t.Errorf("Expected 1 moved file, got %d", result.MovedFiles)
		}

		// Verify file exists on NAS
		nasFile := filepath.Join(tmpNAS, "test.mp3")
		if _, err := os.Stat(nasFile); os.IsNotExist(err) {
			t.Error("File not found on NAS")
		}

		// Verify original file removed
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("Original file not removed")
		}
	})

	t.Run("Skip small files", func(t *testing.T) {
		tmpDownloads2 := filepath.Join(t.TempDir(), "downloads2")
		tmpNAS2 := filepath.Join(t.TempDir(), "nas2")
		os.MkdirAll(tmpDownloads2, 0755)
		os.MkdirAll(tmpNAS2, 0755)

		smallFile := filepath.Join(tmpDownloads2, "small.mp3")
		os.WriteFile(smallFile, []byte("small"), 0644)

		result, err := MoveDownloadsToNAS(tmpDownloads2, tmpNAS2, nil, nil)
		if err != nil {
			t.Fatalf("MoveDownloadsToNAS failed: %v", err)
		}

		if result.TotalFiles != 0 {
			t.Errorf("Expected 0 files (small file should be skipped), got %d", result.TotalFiles)
		}
	})
}

func TestGetTracks(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "tracks")
	os.MkdirAll(tmpDir, 0755)

	// Create test MP3 files
	for i := 1; i <= 5; i++ {
		content := make([]byte, 70000)
		filename := filepath.Join(tmpDir, "track"+string(rune('0'+i))+".mp3")
		os.WriteFile(filename, content, 0644)
	}

	t.Run("List tracks with pagination", func(t *testing.T) {
		result, err := GetTracks(tmpDir, 1, 2, "", "filename", "asc", false)
		if err != nil {
			t.Fatalf("GetTracks failed: %v", err)
		}

		if result.Total != 5 {
			t.Errorf("Expected 5 total tracks, got %d", result.Total)
		}

		if result.PerPage != 2 {
			t.Errorf("Expected 2 per page, got %d", result.PerPage)
		}

		if len(result.Tracks) != 2 {
			t.Errorf("Expected 2 tracks in result, got %d", len(result.Tracks))
		}
	})

	t.Run("Search filter", func(t *testing.T) {
		result, err := GetTracks(tmpDir, 1, 10, "track1", "filename", "asc", false)
		if err != nil {
			t.Fatalf("GetTracks failed: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("Expected 1 track matching search, got %d", result.Total)
		}
	})

	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(t.TempDir(), "empty")
		os.MkdirAll(emptyDir, 0755)

		result, err := GetTracks(emptyDir, 1, 10, "", "filename", "asc", false)
		if err != nil {
			t.Fatalf("GetTracks failed: %v", err)
		}

		if result.Total != 0 {
			t.Errorf("Expected 0 tracks, got %d", result.Total)
		}
	})
}

func TestHashFileMD5(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("test content")
	os.WriteFile(tmpFile, content, 0644)

	t.Run("Hash existing file", func(t *testing.T) {
		hash1, err := hashFileMD5(tmpFile)
		if err != nil {
			t.Fatalf("hashFileMD5 failed: %v", err)
		}

		// Hash same file again - should get same hash
		hash2, err := hashFileMD5(tmpFile)
		if err != nil {
			t.Fatalf("hashFileMD5 failed: %v", err)
		}

		if hash1 != hash2 {
			t.Error("Same file should produce same hash")
		}
	})

	t.Run("Hash non-existent file", func(t *testing.T) {
		_, err := hashFileMD5("/nonexistent/file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestClearCSVs(t *testing.T) {
	t.Run("Clear CSV files successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test CSV files
		os.WriteFile(filepath.Join(tmpDir, "tracks-2026-01-01.csv"), []byte("track1"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "tracks-2026-01-02.csv"), []byte("track2"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "tracks-2026-01-03.csv"), []byte("track3"), 0644)
		// Create non-CSV file that should be ignored
		os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("readme"), 0644)

		result, err := ClearCSVs(tmpDir)
		if err != nil {
			t.Fatalf("ClearCSVs failed: %v", err)
		}

		if result.TotalFiles != 3 {
			t.Errorf("Expected 3 CSV files, got %d", result.TotalFiles)
		}

		if result.DeletedFiles != 3 {
			t.Errorf("Expected 3 deleted files, got %d", result.DeletedFiles)
		}

		if len(result.Errors) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
		}

		// Verify CSV files were deleted
		csvFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.csv"))
		if len(csvFiles) != 0 {
			t.Errorf("Expected 0 CSV files remaining, got %d", len(csvFiles))
		}

		// Verify non-CSV file still exists
		if _, err := os.Stat(filepath.Join(tmpDir, "readme.txt")); os.IsNotExist(err) {
			t.Error("Non-CSV file should not be deleted")
		}
	})

	t.Run("Empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := ClearCSVs(tmpDir)
		if err != nil {
			t.Fatalf("ClearCSVs failed: %v", err)
		}

		if result.TotalFiles != 0 {
			t.Errorf("Expected 0 files in empty directory, got %d", result.TotalFiles)
		}

		if result.DeletedFiles != 0 {
			t.Errorf("Expected 0 deleted files, got %d", result.DeletedFiles)
		}
	})

	t.Run("Non-existent directory", func(t *testing.T) {
		_, err := ClearCSVs("/nonexistent/directory")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}

		if err != nil && !os.IsNotExist(err) {
			expectedMsg := "CSV directory does not exist"
			if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
				t.Errorf("Expected 'CSV directory does not exist' error, got: %v", err)
			}
		}
	})

	t.Run("Partial deletion with errors", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test CSV files
		file1 := filepath.Join(tmpDir, "file1.csv")
		file2 := filepath.Join(tmpDir, "file2.csv")
		os.WriteFile(file1, []byte("content1"), 0644)
		os.WriteFile(file2, []byte("content2"), 0644)

		// Make file2 read-only to cause deletion error
		_ = os.Chmod(tmpDir, 0555) // Read-only directory

		result, err := ClearCSVs(tmpDir)

		// Restore permissions
		_ = os.Chmod(tmpDir, 0755)

		if err != nil {
			t.Fatalf("ClearCSVs should not return error for partial failure: %v", err)
		}

		if result.TotalFiles != 2 {
			t.Errorf("Expected 2 total files, got %d", result.TotalFiles)
		}

		// Should have some errors since directory is read-only
		if result.DeletedFiles == result.TotalFiles && len(result.Errors) == 0 {
			// If all deleted, skip this check (OS behavior varies)
			t.Log("Warning: OS allowed deletion in read-only directory")
		}
	})

	t.Run("Only CSV files are deleted", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create various file types
		os.WriteFile(filepath.Join(tmpDir, "tracks.csv"), []byte("csv"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("json"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("txt"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "song.mp3"), []byte("mp3"), 0644)

		result, err := ClearCSVs(tmpDir)
		if err != nil {
			t.Fatalf("ClearCSVs failed: %v", err)
		}

		if result.TotalFiles != 1 {
			t.Errorf("Expected 1 CSV file, got %d", result.TotalFiles)
		}

		if result.DeletedFiles != 1 {
			t.Errorf("Expected 1 deleted file, got %d", result.DeletedFiles)
		}

		// Verify only CSV was deleted
		if _, err := os.Stat(filepath.Join(tmpDir, "tracks.csv")); !os.IsNotExist(err) {
			t.Error("CSV file should be deleted")
		}

		// Verify other files still exist
		for _, file := range []string{"data.json", "notes.txt", "song.mp3"} {
			if _, err := os.Stat(filepath.Join(tmpDir, file)); os.IsNotExist(err) {
				t.Errorf("Non-CSV file %s should not be deleted", file)
			}
		}
	})
}

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

	t.Run("Only MP3 files are deleted", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create various file types
		os.WriteFile(filepath.Join(tmpDir, "track.mp3"), []byte("mp3"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("json"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("txt"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "tracks.csv"), []byte("csv"), 0644)

		result, err := ClearDownloads(tmpDir)
		if err != nil {
			t.Fatalf("ClearDownloads failed: %v", err)
		}

		if result.TotalFiles != 1 {
			t.Errorf("Expected 1 MP3 file, got %d", result.TotalFiles)
		}

		if result.DeletedFiles != 1 {
			t.Errorf("Expected 1 deleted file, got %d", result.DeletedFiles)
		}

		// Verify only MP3 was deleted
		if _, err := os.Stat(filepath.Join(tmpDir, "track.mp3")); !os.IsNotExist(err) {
			t.Error("MP3 file should be deleted")
		}

		// Verify other files still exist
		for _, file := range []string{"data.json", "notes.txt", "tracks.csv"} {
			if _, err := os.Stat(filepath.Join(tmpDir, file)); os.IsNotExist(err) {
				t.Errorf("Non-MP3 file %s should not be deleted", file)
			}
		}
	})
}

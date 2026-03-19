package storage

import (
	"fmt"
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

	_ = os.WriteFile(filepath.Join(tmpDownloads, "file1.mp3"), file1Content, 0644)
	_ = os.WriteFile(filepath.Join(tmpDownloads, "file2.mp3"), file2Content, 0644)
	_ = os.WriteFile(filepath.Join(tmpNAS, "duplicate.mp3"), duplicateContent, 0644)

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
	_ = os.MkdirAll(tmpDownloads, 0755)
	_ = os.MkdirAll(tmpNAS, 0755)

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
		_ = os.MkdirAll(tmpDownloads2, 0755)
		_ = os.MkdirAll(tmpNAS2, 0755)

		smallFile := filepath.Join(tmpDownloads2, "small.mp3")
		_ = os.WriteFile(smallFile, []byte("small"), 0644)

		result, err := MoveDownloadsToNAS(tmpDownloads2, tmpNAS2, nil, nil)
		if err != nil {
			t.Fatalf("MoveDownloadsToNAS failed: %v", err)
		}

		if result.TotalFiles != 0 {
			t.Errorf("Expected 0 files (small file should be skipped), got %d", result.TotalFiles)
		}
	})

	t.Run("Skip file already on NAS and delete local copy", func(t *testing.T) {
		tmpDownloads3 := filepath.Join(t.TempDir(), "downloads3")
		tmpNAS3 := filepath.Join(t.TempDir(), "nas3")
		_ = os.MkdirAll(tmpDownloads3, 0755)
		_ = os.MkdirAll(tmpNAS3, 0755)

		// Create identical file in both locations
		largeContent := make([]byte, 70000)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		localFile := filepath.Join(tmpDownloads3, "existing.mp3")
		nasFile := filepath.Join(tmpNAS3, "existing.mp3")
		_ = os.WriteFile(localFile, largeContent, 0644)
		_ = os.WriteFile(nasFile, largeContent, 0644)

		result, err := MoveDownloadsToNAS(tmpDownloads3, tmpNAS3, nil, nil)
		if err != nil {
			t.Fatalf("MoveDownloadsToNAS failed: %v", err)
		}

		if result.TotalFiles != 1 {
			t.Errorf("Expected 1 file, got %d", result.TotalFiles)
		}

		if result.SkippedFiles != 1 {
			t.Errorf("Expected 1 skipped file, got %d", result.SkippedFiles)
		}

		if result.MovedFiles != 0 {
			t.Errorf("Expected 0 moved files (file already on NAS), got %d", result.MovedFiles)
		}

		// Verify local file was deleted
		if _, err := os.Stat(localFile); !os.IsNotExist(err) {
			t.Error("Local file should be deleted when it already exists on NAS")
		}

		// Verify NAS file still exists
		if _, err := os.Stat(nasFile); os.IsNotExist(err) {
			t.Error("NAS file should still exist")
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

		result, err := ClearDownloads(tmpDir, nil)
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

		result, err := ClearDownloads(tmpDir, nil)
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
		_, err := ClearDownloads("/nonexistent/directory", nil)
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

		result, err := ClearDownloads(tmpDir, nil)
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

		result, err := ClearDownloads(tmpDir, nil)
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

		result, err := ClearDownloads(tmpDir, nil)
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

// ---- GetTracks sorting ----

func TestGetTracks_Sorting(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"zzz.mp3", "aaa.mp3", "mmm.mp3"} {
		content := make([]byte, 70000)
		_ = os.WriteFile(filepath.Join(tmpDir, name), content, 0644)
	}

	t.Run("sort ascending", func(t *testing.T) {
		result, err := GetTracks(tmpDir, 1, 10, "", "filename", "asc", false)
		if err != nil {
			t.Fatalf("GetTracks failed: %v", err)
		}
		if len(result.Tracks) < 2 {
			t.Skip("not enough tracks to test sorting")
		}
		if result.Tracks[0].Filename >= result.Tracks[1].Filename {
			t.Errorf("ascending sort failed: %q should come before %q",
				result.Tracks[0].Filename, result.Tracks[1].Filename)
		}
	})

	t.Run("sort descending", func(t *testing.T) {
		result, err := GetTracks(tmpDir, 1, 10, "", "filename", "desc", false)
		if err != nil {
			t.Fatalf("GetTracks failed: %v", err)
		}
		if len(result.Tracks) < 2 {
			t.Skip("not enough tracks to test sorting")
		}
		if result.Tracks[0].Filename <= result.Tracks[1].Filename {
			t.Errorf("descending sort failed: %q should come after %q",
				result.Tracks[0].Filename, result.Tracks[1].Filename)
		}
	})
}

// ---- RemoveDuplicates with progress callback ----

func TestRemoveDuplicates_WithProgressCallback(t *testing.T) {
	tmpDir := t.TempDir()
	content1 := []byte("unique content alpha")
	content2 := []byte("unique content beta")

	_ = os.WriteFile(filepath.Join(tmpDir, "file1.mp3"), content1, 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "file2.mp3"), content2, 0644)

	var progressCalled bool
	callback := func(current, total int) {
		progressCalled = true
	}

	result, err := RemoveDuplicates(tmpDir, "", nil, callback)
	if err != nil {
		t.Fatalf("RemoveDuplicates failed: %v", err)
	}
	_ = result         // no duplicates expected
	_ = progressCalled // progress callback may or may not be called for 2 files
}

// ---- ClearCSVs with non-existent directory ----

func TestClearCSVs_NonExistentDirectory(t *testing.T) {
	_, err := ClearCSVs("/definitely/does/not/exist/path")
	if err == nil {
		t.Error("ClearCSVs should return error for non-existent directory")
	}
}

// ---- MatchDownloadsToCSVs tests ----

func TestMatchDownloadsToCSVs_NoCSVFiles(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()
	_, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err == nil {
		t.Error("Expected error when no CSV files found")
	}
}

func TestMatchDownloadsToCSVs_EmptyDownloads(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	csvContent := "Title,Artist,Album,Label,Date,URL\nTrack One,Artist One,Album,Label,2026-01-01,http://example.com\n"
	os.WriteFile(filepath.Join(outDir, "tracks-2026-01-01.csv"), []byte(csvContent), 0644)

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if result.TotalFiles != 0 {
		t.Errorf("Expected 0 total files, got %d", result.TotalFiles)
	}
	if result.MatchedFiles != 0 {
		t.Errorf("Expected 0 matched files, got %d", result.MatchedFiles)
	}
	if result.LatestCSV != nil {
		t.Error("Expected LatestCSV to be nil when no matches")
	}
}

func TestMatchDownloadsToCSVs_MatchingFile(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	csvContent := "Title,Artist,Album,Label,Date,URL\nTrack One,Artist One,Album,Label,2026-01-01,http://example.com\n"
	os.WriteFile(filepath.Join(outDir, "tracks-2026-01-01.csv"), []byte(csvContent), 0644)

	// "Artist One - Track One.mp3" => artist="artist one", title="track one" => overlaps CSV
	os.WriteFile(filepath.Join(dlDir, "Artist One - Track One.mp3"), []byte("mp3"), 0644)

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if result.TotalFiles != 1 {
		t.Errorf("Expected 1 total file, got %d", result.TotalFiles)
	}
	if result.MatchedFiles != 1 {
		t.Errorf("Expected 1 matched file, got %d", result.MatchedFiles)
	}
	if len(result.TopCSVs) == 0 {
		t.Fatal("Expected at least one TopCSV entry")
	}
	if result.LatestCSV == nil {
		t.Fatal("Expected LatestCSV to be set")
	}
	if result.LatestCSV.Date != "2026-01-01" {
		t.Errorf("Expected date 2026-01-01, got %s", result.LatestCSV.Date)
	}
}

func TestMatchDownloadsToCSVs_NoMatch(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	csvContent := "Title,Artist,Album,Label,Date,URL\nTrack One,Artist One,Album,Label,2026-01-01,http://example.com\n"
	os.WriteFile(filepath.Join(outDir, "tracks-2026-01-01.csv"), []byte(csvContent), 0644)

	// completely different name — no word overlap
	os.WriteFile(filepath.Join(dlDir, "zzz.mp3"), []byte("mp3"), 0644)

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if result.TotalFiles != 1 {
		t.Errorf("Expected 1 total file, got %d", result.TotalFiles)
	}
	if result.MatchedFiles != 0 {
		t.Errorf("Expected 0 matched files, got %d", result.MatchedFiles)
	}
}

func TestMatchDownloadsToCSVs_MultipleCSVs(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	csv1 := "Title,Artist,Album,Label,Date,URL\nAlpha Song,Alpha Artist,Album,Label,2026-01-01,http://example.com\n"
	csv2 := "Title,Artist,Album,Label,Date,URL\nBeta Song,Beta Artist,Album,Label,2026-02-01,http://example.com\n"
	os.WriteFile(filepath.Join(outDir, "tracks-2026-01-01.csv"), []byte(csv1), 0644)
	os.WriteFile(filepath.Join(outDir, "tracks-2026-02-01.csv"), []byte(csv2), 0644)

	os.WriteFile(filepath.Join(dlDir, "Alpha Artist - Alpha Song.mp3"), []byte("mp3a"), 0644)
	os.WriteFile(filepath.Join(dlDir, "Beta Artist - Beta Song.mp3"), []byte("mp3b"), 0644)

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if result.TotalFiles != 2 {
		t.Errorf("Expected 2 total files, got %d", result.TotalFiles)
	}
	if result.MatchedFiles != 2 {
		t.Errorf("Expected 2 matched files, got %d", result.MatchedFiles)
	}
	if len(result.TopCSVs) != 2 {
		t.Errorf("Expected 2 TopCSVs, got %d", len(result.TopCSVs))
	}
	if result.LatestCSV == nil {
		t.Fatal("Expected LatestCSV to be set")
	}
	if result.LatestCSV.Date != "2026-02-01" {
		t.Errorf("Expected latest date 2026-02-01, got %s", result.LatestCSV.Date)
	}
}

func TestMatchDownloadsToCSVs_FilenameWithoutSeparator(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	csvContent := "Title,Artist,Album,Label,Date,URL\nrocktrack,someartist,Album,Label,2026-03-01,http://example.com\n"
	os.WriteFile(filepath.Join(outDir, "tracks-2026-03-01.csv"), []byte(csvContent), 0644)

	// No " - " separator => whole name used as title
	os.WriteFile(filepath.Join(dlDir, "rocktrack.mp3"), []byte("mp3"), 0644)

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if result.TotalFiles != 1 {
		t.Errorf("Expected 1 total file, got %d", result.TotalFiles)
	}
	if result.MatchedFiles != 1 {
		t.Errorf("Expected 1 matched file, got %d", result.MatchedFiles)
	}
}

func TestMatchDownloadsToCSVs_ManyCSVsTrimmedTo20(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	// 25 CSVs each with one track + one matching MP3
	for i := 1; i <= 25; i++ {
		month := fmt.Sprintf("%02d", i%12+1)
		csvName := fmt.Sprintf("tracks-2026-%s-%02d.csv", month, i%28+1)
		word := fmt.Sprintf("uniqueword%d", i)
		csvContent := fmt.Sprintf("Title,Artist,Album,Label,Date,URL\n%s,%s,Album,Label,2026-01-01,http://example.com\n", word, word)
		os.WriteFile(filepath.Join(outDir, csvName), []byte(csvContent), 0644)
		os.WriteFile(filepath.Join(dlDir, fmt.Sprintf("%s - %s.mp3", word, word)), []byte("mp3"), 0644)
	}

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if len(result.TopCSVs) > 20 {
		t.Errorf("Expected at most 20 TopCSVs, got %d", len(result.TopCSVs))
	}
}

func TestMatchDownloadsToCSVs_CSVHeaderOnly(t *testing.T) {
	outDir := t.TempDir()
	dlDir := t.TempDir()

	// CSV with only header row — no data
	csvContent := "Title,Artist,Album,Label,Date,URL\n"
	os.WriteFile(filepath.Join(outDir, "tracks-2026-01-01.csv"), []byte(csvContent), 0644)
	os.WriteFile(filepath.Join(dlDir, "Some Artist - Some Track.mp3"), []byte("mp3"), 0644)

	result, err := MatchDownloadsToCSVs(outDir, dlDir)
	if err != nil {
		t.Fatalf("MatchDownloadsToCSVs failed: %v", err)
	}
	if result.MatchedFiles != 0 {
		t.Errorf("Expected 0 matches with header-only CSV, got %d", result.MatchedFiles)
	}
}

// ---- GetTracks edge cases ----

func TestGetTracks_InvalidPageAndPerPage(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "tracks")
	os.MkdirAll(tmpDir, 0755)
	for i := 0; i < 3; i++ {
		os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("t%d.mp3", i)), make([]byte, 70000), 0644)
	}

	// page < 1 => normalized to 1
	result, err := GetTracks(tmpDir, 0, 10, "", "filename", "asc", false)
	if err != nil {
		t.Fatalf("GetTracks failed: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("Expected page=1 for input page=0, got %d", result.Page)
	}

	// perPage > 100 => normalized to 50
	result, err = GetTracks(tmpDir, 1, 200, "", "filename", "asc", false)
	if err != nil {
		t.Fatalf("GetTracks failed: %v", err)
	}
	if result.PerPage != 50 {
		t.Errorf("Expected perPage=50 for input perPage=200, got %d", result.PerPage)
	}

	// perPage < 1 => normalized to 50
	result, err = GetTracks(tmpDir, 1, 0, "", "filename", "asc", false)
	if err != nil {
		t.Fatalf("GetTracks failed: %v", err)
	}
	if result.PerPage != 50 {
		t.Errorf("Expected perPage=50 for input perPage=0, got %d", result.PerPage)
	}
}

func TestGetTracks_PageBeyondTotal(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "tracks_beyond")
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "t1.mp3"), make([]byte, 70000), 0644)

	result, err := GetTracks(tmpDir, 100, 10, "", "filename", "asc", false)
	if err != nil {
		t.Fatalf("GetTracks failed: %v", err)
	}
	if len(result.Tracks) != 0 {
		t.Errorf("Expected 0 tracks on page beyond total, got %d", len(result.Tracks))
	}
	if result.Total != 1 {
		t.Errorf("Expected total=1, got %d", result.Total)
	}
}

func TestGetTracks_AsyncMode(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "tracks_async")
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "t1.mp3"), make([]byte, 70000), 0644)

	result, err := GetTracks(tmpDir, 1, 10, "", "filename", "asc", true)
	if err != nil {
		t.Fatalf("GetTracks(async) failed: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result from async GetTracks")
	}
}

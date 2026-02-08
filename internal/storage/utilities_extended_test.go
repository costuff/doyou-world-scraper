package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNormalizeString tests the normalize function
func TestNormalizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"UPPER CASE", "upper case"},
		{"MixedCase123", "mixedcase123"},
		{"  extra   spaces  ", "extra spaces"},
		{"Café", "café"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalize(tt.input)
			if result != tt.expected {
				t.Errorf("normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestWordSet tests the wordSet function
func TestWordSet(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"one two three", []string{"one", "two", "three"}},
		{"repeated repeated word", []string{"repeated", "word"}},
		{"", []string{}},
		{"  multiple   spaces  ", []string{"multiple", "spaces"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := wordSet(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("wordSet(%q) has %d words, want %d", tt.input, len(result), len(tt.expected))
			}
			for _, word := range tt.expected {
				if _, ok := result[word]; !ok {
					t.Errorf("wordSet(%q) missing word %q", tt.input, word)
				}
			}
		})
	}
}

// TestOverlapCount tests the overlapCount function
func TestOverlapCount(t *testing.T) {
	tests := []struct {
		name     string
		set1     map[string]struct{}
		set2     map[string]struct{}
		expected int
	}{
		{
			name:     "No overlap",
			set1:     map[string]struct{}{"a": {}, "b": {}},
			set2:     map[string]struct{}{"c": {}, "d": {}},
			expected: 0,
		},
		{
			name:     "Partial overlap",
			set1:     map[string]struct{}{"a": {}, "b": {}, "c": {}},
			set2:     map[string]struct{}{"b": {}, "c": {}, "d": {}},
			expected: 2,
		},
		{
			name:     "Complete overlap",
			set1:     map[string]struct{}{"a": {}, "b": {}},
			set2:     map[string]struct{}{"a": {}, "b": {}},
			expected: 2,
		},
		{
			name:     "Empty sets",
			set1:     map[string]struct{}{},
			set2:     map[string]struct{}{},
			expected: 0,
		},
		{
			name:     "One empty set",
			set1:     map[string]struct{}{"a": {}},
			set2:     map[string]struct{}{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := overlapCount(tt.set1, tt.set2)
			if result != tt.expected {
				t.Errorf("overlapCount() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestApplySearchFilter tests search filtering
func TestApplySearchFilter(t *testing.T) {
	tracks := []TrackInfo{
		{Filename: "track1.mp3", Title: "Amazing Song", Artist: "Cool Artist"},
		{Filename: "track2.mp3", Title: "Another Track", Artist: "Different Artist"},
		{Filename: "track3.mp3", Title: "Best Song Ever", Artist: "Cool Artist"},
		{Filename: "track4.mp3", Title: "Random Title", Artist: "Random Artist"},
	}

	tests := []struct {
		name     string
		search   string
		expected int
	}{
		{"Empty search returns all", "", 4},
		{"Artist search", "cool artist", 2},
		{"Title search", "song", 2},
		{"Filename search", "track1", 1},
		{"No matches", "nonexistent", 0},
		{"Case insensitive", "AMAZING", 1},
		{"Partial match", "best", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := applySearchFilter(tracks, tt.search)
			if len(filtered) != tt.expected {
				t.Errorf("applySearchFilter(%q) returned %d tracks, want %d", tt.search, len(filtered), tt.expected)
			}
		})
	}
}

// TestSortTracks tests track sorting
func TestSortTracks(t *testing.T) {
	tracks := []TrackInfo{
		{Filename: "c.mp3", Title: "C", Artist: "Artist C", FileSize: 3000000, Duration: 300},
		{Filename: "a.mp3", Title: "A", Artist: "Artist A", FileSize: 1000000, Duration: 100},
		{Filename: "b.mp3", Title: "B", Artist: "Artist B", FileSize: 2000000, Duration: 200},
	}

	t.Run("Sort by filename asc", func(t *testing.T) {
		testTracks := make([]TrackInfo, len(tracks))
		copy(testTracks, tracks)
		sortTracks(testTracks, "filename", "asc")
		if testTracks[0].Filename != "a.mp3" {
			t.Error("First track should be a.mp3")
		}
	})

	t.Run("Sort by filename desc", func(t *testing.T) {
		testTracks := make([]TrackInfo, len(tracks))
		copy(testTracks, tracks)
		sortTracks(testTracks, "filename", "desc")
		if testTracks[0].Filename != "c.mp3" {
			t.Error("First track should be c.mp3")
		}
	})

	t.Run("Sort by title", func(t *testing.T) {
		testTracks := make([]TrackInfo, len(tracks))
		copy(testTracks, tracks)
		sortTracks(testTracks, "title", "asc")
		if testTracks[0].Title != "A" {
			t.Error("First track should have title A")
		}
	})

	t.Run("Sort by artist", func(t *testing.T) {
		testTracks := make([]TrackInfo, len(tracks))
		copy(testTracks, tracks)
		sortTracks(testTracks, "artist", "asc")
		if testTracks[0].Artist != "Artist A" {
			t.Error("First track should have Artist A")
		}
	})

	t.Run("Sort by size", func(t *testing.T) {
		testTracks := make([]TrackInfo, len(tracks))
		copy(testTracks, tracks)
		sortTracks(testTracks, "size", "asc")
		if testTracks[0].FileSize != 1000000 {
			t.Error("First track should have smallest size")
		}
	})

	t.Run("Sort by duration", func(t *testing.T) {
		testTracks := make([]TrackInfo, len(tracks))
		copy(testTracks, tracks)
		sortTracks(testTracks, "duration", "desc")
		if testTracks[0].Duration != 300 {
			t.Error("First track should have longest duration")
		}
	})
}

// TestCopyFileErrors tests copyFile error cases
func TestCopyFileErrors(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Source file doesn't exist", func(t *testing.T) {
		src := filepath.Join(tmpDir, "nonexistent.txt")
		dst := filepath.Join(tmpDir, "dest.txt")
		err := copyFile(src, dst)
		if err == nil {
			t.Error("Expected error for non-existent source")
		}
	})

	t.Run("Destination directory doesn't exist", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source.txt")
		os.WriteFile(src, []byte("test"), 0644)

		dst := filepath.Join(tmpDir, "nonexistent", "dest.txt")
		err := copyFile(src, dst)
		if err == nil {
			t.Error("Expected error for non-existent destination directory")
		}
	})

	t.Run("Successful copy", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source2.txt")
		content := []byte("test content")
		os.WriteFile(src, content, 0644)

		dst := filepath.Join(tmpDir, "dest2.txt")
		err := copyFile(src, dst)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify content copied correctly
		dstContent, _ := os.ReadFile(dst)
		if string(dstContent) != string(content) {
			t.Error("File content not copied correctly")
		}
	})
}

// TestHashFileMD5Extended tests additional MD5 hashing scenarios
func TestHashFileMD5Extended(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Empty file", func(t *testing.T) {
		file := filepath.Join(tmpDir, "empty.txt")
		os.WriteFile(file, []byte(""), 0644)

		_, err := hashFileMD5(file)
		if err != nil {
			t.Errorf("Expected no error for empty file, got: %v", err)
		}
	})

	t.Run("Large file", func(t *testing.T) {
		file := filepath.Join(tmpDir, "large.txt")
		// Create 1MB file
		content := make([]byte, 1024*1024)
		for i := range content {
			content[i] = byte(i % 256)
		}
		os.WriteFile(file, content, 0644)

		_, err := hashFileMD5(file)
		if err != nil {
			t.Errorf("Expected no error for large file, got: %v", err)
		}
	})
}

// TestClearCSVsEdgeCases tests ClearCSVs edge cases
func TestClearCSVsEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Non-existent directory", func(t *testing.T) {
		nonExistent := filepath.Join(tmpDir, "nonexistent")
		result, err := ClearCSVs(nonExistent)
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
		if result != nil {
			t.Error("Result should be nil for error case")
		}
	})

	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.MkdirAll(emptyDir, 0755)

		result, err := ClearCSVs(emptyDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.DeletedFiles != 0 {
			t.Error("Should delete 0 files from empty directory")
		}
	})

	t.Run("Directory with non-CSV files", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "mixed")
		os.MkdirAll(dir, 0755)

		os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0644)

		result, err := ClearCSVs(dir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.DeletedFiles != 0 {
			t.Error("Should not delete non-CSV files")
		}

		// Verify non-CSV files still exist
		if _, err := os.Stat(filepath.Join(dir, "test.txt")); os.IsNotExist(err) {
			t.Error("txt file should not be deleted")
		}
	})
}

// TestClearDownloadsEdgeCases tests ClearDownloads edge cases
func TestClearDownloadsEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Non-existent directory", func(t *testing.T) {
		nonExistent := filepath.Join(tmpDir, "nonexistent")
		result, err := ClearDownloads(nonExistent)
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
		if result != nil {
			t.Error("Result should be nil for error case")
		}
	})

	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.MkdirAll(emptyDir, 0755)

		result, err := ClearDownloads(emptyDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.DeletedFiles != 0 {
			t.Error("Should delete 0 files from empty directory")
		}
	})

	t.Run("Preserves failed_songs.csv", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "preserve")
		os.MkdirAll(dir, 0755)

		failedCSV := filepath.Join(dir, "failed_songs.csv")
		os.WriteFile(failedCSV, []byte("track,artist\n"), 0644)
		os.WriteFile(filepath.Join(dir, "track1.mp3"), []byte("audio"), 0644)

		result, err := ClearDownloads(dir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// failed_songs.csv should still exist
		if _, err := os.Stat(failedCSV); os.IsNotExist(err) {
			t.Error("failed_songs.csv should not be deleted")
		}

		// MP3 should be deleted
		if _, err := os.Stat(filepath.Join(dir, "track1.mp3")); !os.IsNotExist(err) {
			t.Error("MP3 file should be deleted")
		}

		if result.DeletedFiles != 1 {
			t.Errorf("Should delete 1 file, deleted %d", result.DeletedFiles)
		}
	})
}

// TestTrackScanStatus tests track scanning status management
func TestTrackScanStatus(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Initial status", func(t *testing.T) {
		status := GetTrackScanStatus(tmpDir)
		if status.InProgress {
			t.Error("Initial status should not be in progress")
		}
	})

	t.Run("Update progress", func(t *testing.T) {
		updateTrackScanProgress(tmpDir, 50, true, "")
		status := GetTrackScanStatus(tmpDir)
		if !status.InProgress {
			t.Error("Status should be in progress")
		}
		if status.Scanned != 50 {
			t.Errorf("Expected 50 scanned, got %d", status.Scanned)
		}
	})

	t.Run("Complete scan", func(t *testing.T) {
		updateTrackScanProgress(tmpDir, 100, false, "")
		status := GetTrackScanStatus(tmpDir)
		if status.InProgress {
			t.Error("Status should not be in progress")
		}
	})

	t.Run("Error status", func(t *testing.T) {
		updateTrackScanProgress(tmpDir, 25, false, "test error")
		status := GetTrackScanStatus(tmpDir)
		if status.Error != "test error" {
			t.Errorf("Expected error message, got %q", status.Error)
		}
	})
}

// TestCachedTracks tests track caching
func TestCachedTracks(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("No cache initially", func(t *testing.T) {
		_, ok, _ := getCachedTracks(tmpDir)
		if ok {
			t.Error("Should have no cached tracks initially")
		}
	})

	t.Run("Set and get cache", func(t *testing.T) {
		tracks := []TrackInfo{
			{Filename: "test1.mp3", Title: "Test 1"},
			{Filename: "test2.mp3", Title: "Test 2"},
		}

		setCachedTracks(tmpDir, tracks)

		cached, ok, stale := getCachedTracks(tmpDir)
		if !ok {
			t.Error("Should have cached tracks")
		}
		if stale {
			t.Error("Cache should not be stale immediately")
		}
		if len(cached) != 2 {
			t.Errorf("Expected 2 cached tracks, got %d", len(cached))
		}
	})
}

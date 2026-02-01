package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_file.mp3", "normal_file.mp3"},
		{"file/with/slashes", "file_with_slashes"},
		{"file\\with\\backslashes", "file_with_backslashes"},
		{"file:with:colons", "file_with_colons"},
		{"file*with*asterisks", "file_with_asterisks"},
		{"file?with?questions", "file_with_questions"},
		{`file"with"quotes`, "file_with_quotes"},
		{"file<with>brackets", "file_with_brackets"},
		{"file|with|pipes", "file_with_pipes"},
		{"file/\\:*?\"<>|all", "file_________all"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPlayableFromAnchors(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "SoundCloud link (highest priority)",
			html:     `<a href="https://soundcloud.com/artist/track">SoundCloud</a><a href="https://youtube.com/watch?v=123">YouTube</a>`,
			expected: "https://soundcloud.com/artist/track",
		},
		{
			name:     "YouTube link",
			html:     `<a href="https://youtube.com/watch?v=123">YouTube</a><a href="https://spotify.com/track/456">Spotify</a>`,
			expected: "https://youtube.com/watch?v=123",
		},
		{
			name:     "YouTube short link",
			html:     `<a href="https://youtu.be/abc123">YouTube</a>`,
			expected: "https://youtu.be/abc123",
		},
		{
			name:     "YouTube Music link",
			html:     `<a href="https://music.youtube.com/watch?v=xyz">YT Music</a><a href="https://spotify.com/track/789">Spotify</a>`,
			expected: "https://music.youtube.com/watch?v=xyz",
		},
		{
			name:     "Spotify link",
			html:     `<a href="https://spotify.com/track/abc">Spotify</a><a href="https://music.apple.com/album/123">Apple</a>`,
			expected: "https://spotify.com/track/abc",
		},
		{
			name:     "Apple Music link",
			html:     `<a href="https://music.apple.com/us/album/song/123456">Apple Music</a>`,
			expected: "https://music.apple.com/us/album/song/123456",
		},
		{
			name:     "No playable links",
			html:     `<a href="https://example.com">Example</a>`,
			expected: "",
		},
		{
			name:     "Empty HTML",
			html:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPlayableFromAnchors(tt.html)
			if result != tt.expected {
				t.Errorf("extractPlayableFromAnchors() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResolvePlayableURL(t *testing.T) {
	t.Run("Non lis.tn URL returns as-is", func(t *testing.T) {
		url := "https://soundcloud.com/artist/track"
		result, err := resolvePlayableURL(url)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != url {
			t.Errorf("Expected %q, got %q", url, result)
		}
	})

	t.Run("Direct SoundCloud URL", func(t *testing.T) {
		url := "https://soundcloud.com/artist/track-name"
		result, err := resolvePlayableURL(url)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != url {
			t.Errorf("Expected %q, got %q", url, result)
		}
	})

	t.Run("Direct YouTube URL", func(t *testing.T) {
		url := "https://youtube.com/watch?v=dQw4w9WgXcQ"
		result, err := resolvePlayableURL(url)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != url {
			t.Errorf("Expected %q, got %q", url, result)
		}
	})

	// Skip lis.tn tests as they require network/browser automation
	t.Run("lis.tn URL requires network", func(t *testing.T) {
		t.Skip("Skipping network-dependent test")
	})
}

func TestDownloadFromCSV(t *testing.T) {
	t.Run("Missing CSV file", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "nonexistent.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		_, err := DownloadFromCSV(csvPath, 5, outDir, "yt-dlp", nil, nil)
		if err == nil {
			t.Error("Expected error for missing CSV file")
		}
		if !strings.Contains(err.Error(), "error opening CSV") {
			t.Errorf("Expected 'error opening CSV' error, got: %v", err)
		}
	})

	t.Run("Empty CSV file", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "empty.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		// Create empty CSV
		os.WriteFile(csvPath, []byte("Title,Artist,Album,Label,Date,Listen\n"), 0644)

		result, err := DownloadFromCSV(csvPath, 5, outDir, "yt-dlp", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.TotalSongs != 0 {
			t.Errorf("Expected 0 songs, got %d", result.TotalSongs)
		}
	})

	t.Run("CSV with job cancellation", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "test.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		// Create CSV with one track
		csvContent := "Title,Artist,Album,Label,Date,Listen\nTest Track,Test Artist,,,2026,https://soundcloud.com/test/track\n"
		os.WriteFile(csvPath, []byte(csvContent), 0644)

		// Create a cancelled job
		job := NewJob("test_download", nil)
		job.Cancel()

		result, err := DownloadFromCSV(csvPath, 5, outDir, "yt-dlp", job, nil)
		if err == nil || !strings.Contains(err.Error(), "job cancelled") {
			t.Errorf("Expected 'job cancelled' error, got: %v", err)
		}

		if result == nil {
			t.Error("Expected result even on cancellation")
		}
	})
}

func TestDownloadFromFolder(t *testing.T) {
	t.Run("No CSV files in folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		outDir := filepath.Join(tmpDir, "downloads")

		_, err := DownloadFromFolder(tmpDir, 5, outDir, "yt-dlp", nil, nil)
		if err == nil {
			t.Error("Expected error for empty folder")
		}
		if !strings.Contains(err.Error(), "no CSV files found") {
			t.Errorf("Expected 'no CSV files found' error, got: %v", err)
		}
	})

	t.Run("Folder with empty CSVs", func(t *testing.T) {
		tmpDir := t.TempDir()
		outDir := filepath.Join(tmpDir, "downloads")

		// Create multiple empty CSV files
		for i := 1; i <= 3; i++ {
			csvPath := filepath.Join(tmpDir, fmt.Sprintf("tracks-%d.csv", i))
			os.WriteFile(csvPath, []byte("Title,Artist,Album,Label,Date,Listen\n"), 0644)
		}

		result, err := DownloadFromFolder(tmpDir, 5, outDir, "yt-dlp", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.TotalSongs != 0 {
			t.Errorf("Expected 0 total songs, got %d", result.TotalSongs)
		}
	})

	t.Run("Job cancellation stops processing", func(t *testing.T) {
		tmpDir := t.TempDir()
		outDir := filepath.Join(tmpDir, "downloads")

		// Create multiple CSV files
		for i := 1; i <= 3; i++ {
			csvPath := filepath.Join(tmpDir, fmt.Sprintf("tracks-%d.csv", i))
			content := "Title,Artist,Album,Label,Date,Listen\nTrack,Artist,,,2026,https://soundcloud.com/test\n"
			os.WriteFile(csvPath, []byte(content), 0644)
		}

		// Create a cancelled job
		job := NewJob("test_download_folder", nil)
		job.Cancel()

		result, err := DownloadFromFolder(tmpDir, 5, outDir, "yt-dlp", job, nil)
		if err == nil || !strings.Contains(err.Error(), "job cancelled") {
			t.Errorf("Expected 'job cancelled' error, got: %v", err)
		}

		if result == nil {
			t.Error("Expected result even on cancellation")
		}
	})
}

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
		_ = os.WriteFile(csvPath, []byte("Title,Artist,Album,Label,Date,Listen\n"), 0644)

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
		_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

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
			_ = os.WriteFile(csvPath, []byte("Title,Artist,Album,Label,Date,Listen\n"), 0644)
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
			_ = os.WriteFile(csvPath, []byte(content), 0644)
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

func TestDownloadFromCSV_ProgressTracking(t *testing.T) {
	t.Run("Progress callback invoked per song", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "test.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		// Create CSV with 3 tracks
		csvContent := `Title,Artist,Album,Label,Date,Listen
Track1,Artist1,,,2026,https://soundcloud.com/test/track1
Track2,Artist2,,,2026,https://soundcloud.com/test/track2
Track3,Artist3,,,2026,https://soundcloud.com/test/track3
`
		_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

		progressCalls := []struct {
			current, total int
			title, artist  string
		}{}
		progressCallback := func(current, total int, title, artist string) {
			progressCalls = append(progressCalls, struct {
				current, total int
				title, artist  string
			}{current, total, title, artist})
		}

		// This will fail to download (no yt-dlp), but should still track progress
		_, _ = DownloadFromCSV(csvPath, 5, outDir, "fake-yt-dlp-that-doesnt-exist", nil, progressCallback)

		// Verify progress was called for each song
		if len(progressCalls) != 3 {
			t.Errorf("Expected 3 progress callbacks, got %d", len(progressCalls))
		}

		// Verify incremental progress (1/3, 2/3, 3/3) with song info
		expectedSongs := []struct{ title, artist string }{
			{"Track1", "Artist1"},
			{"Track2", "Artist2"},
			{"Track3", "Artist3"},
		}
		for i, call := range progressCalls {
			expectedCurrent := i + 1
			expectedTotal := 3
			if call.current != expectedCurrent || call.total != expectedTotal {
				t.Errorf("Progress call %d: expected (%d/%d), got (%d/%d)",
					i, expectedCurrent, expectedTotal, call.current, call.total)
			}
			if call.title != expectedSongs[i].title || call.artist != expectedSongs[i].artist {
				t.Errorf("Progress call %d: expected (%s - %s), got (%s - %s)",
					i, expectedSongs[i].title, expectedSongs[i].artist, call.title, call.artist)
			}
		}
	})
}

func TestDownloadFromFolder_GranularProgress(t *testing.T) {
	t.Run("Progress tracks all songs across all CSVs", func(t *testing.T) {
		tmpDir := t.TempDir()
		outDir := filepath.Join(tmpDir, "downloads")

		// Create 2 CSV files with different song counts
		csv1 := filepath.Join(tmpDir, "tracks-1.csv")
		csv1Content := `Title,Artist,Album,Label,Date,Listen
Track1,Artist1,,,2026,https://soundcloud.com/test/track1
Track2,Artist2,,,2026,https://soundcloud.com/test/track2
`
		_ = os.WriteFile(csv1, []byte(csv1Content), 0644)

		csv2 := filepath.Join(tmpDir, "tracks-2.csv")
		csv2Content := `Title,Artist,Album,Label,Date,Listen
Track3,Artist3,,,2026,https://soundcloud.com/test/track3
Track4,Artist4,,,2026,https://soundcloud.com/test/track4
Track5,Artist5,,,2026,https://soundcloud.com/test/track5
`
		_ = os.WriteFile(csv2, []byte(csv2Content), 0644)

		progressCalls := []struct {
			current, total int
			title, artist  string
		}{}
		progressCallback := func(current, total int, title, artist string) {
			progressCalls = append(progressCalls, struct {
				current, total int
				title, artist  string
			}{current, total, title, artist})
		}

		// This will fail to download (no yt-dlp), but should still track progress
		_, _ = DownloadFromFolder(tmpDir, 5, outDir, "fake-yt-dlp-that-doesnt-exist", nil, progressCallback)

		// Verify progress was called for all 5 songs (2 from CSV1 + 3 from CSV2)
		if len(progressCalls) != 5 {
			t.Errorf("Expected 5 progress callbacks (all songs), got %d", len(progressCalls))
		}

		// Verify total is always 5 (total songs across all CSVs)
		for i, call := range progressCalls {
			if call.total != 5 {
				t.Errorf("Progress call %d: expected total=5, got %d", i, call.total)
			}
		}

		// Verify progress increments correctly (1/5, 2/5, 3/5, 4/5, 5/5)
		for i, call := range progressCalls {
			expectedCurrent := i + 1
			if call.current != expectedCurrent {
				t.Errorf("Progress call %d: expected current=%d, got %d",
					i, expectedCurrent, call.current)
			}
		}

		// Verify song info is passed correctly
		expectedSongs := []struct{ title, artist string }{
			{"Track1", "Artist1"},
			{"Track2", "Artist2"},
			{"Track3", "Artist3"},
			{"Track4", "Artist4"},
			{"Track5", "Artist5"},
		}
		for i, call := range progressCalls {
			if call.title != expectedSongs[i].title || call.artist != expectedSongs[i].artist {
				t.Errorf("Progress call %d: expected (%s - %s), got (%s - %s)",
					i, expectedSongs[i].title, expectedSongs[i].artist, call.title, call.artist)
			}
		}
	})
}

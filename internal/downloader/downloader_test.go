package downloader

import (
	"doyou-world-scraper/internal/jobs"
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
		job := jobs.NewJob("test_download", nil)
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
		job := jobs.NewJob("test_download_folder", nil)
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

// TestFindPlayableURLInHTML tests the pure HTML-matching helper directly.
func TestFindPlayableURLInHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "SoundCloud anchor (highest priority)",
			html:     `<a href="https://soundcloud.com/artist/track">SC</a>`,
			expected: "https://soundcloud.com/artist/track",
		},
		{
			name:     "YouTube anchor",
			html:     `<a href="https://youtube.com/watch?v=abc">YT</a>`,
			expected: "https://youtube.com/watch?v=abc",
		},
		{
			name:     "SoundCloud URL in raw text - regex fallback",
			html:     `<p>Listen on https://soundcloud.com/artist/track-name" today</p>`,
			expected: `https://soundcloud.com/artist/track-name`,
		},
		{
			name:     "YouTube URL in raw text - regex fallback",
			html:     `<p>Watch at https://youtube.com/watch?v=dQw4w9WgXcQ" now</p>`,
			expected: `https://youtube.com/watch?v=dQw4w9WgXcQ`,
		},
		{
			name:     "youtu.be short link in raw text",
			html:     `<p>https://youtu.be/abc123" here</p>`,
			expected: `https://youtu.be/abc123`,
		},
		{
			name:     "No playable URLs",
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
			result := findPlayableURLInHTML(tt.html)
			if result != tt.expected {
				t.Errorf("findPlayableURLInHTML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestResolvePlayableURL_WithMock tests the lis.tn resolution logic by injecting
// a mock htmlPageFetcher to avoid network/browser dependencies.
func TestResolvePlayableURL_WithMock(t *testing.T) {
	// Save and restore the package-level fetcher.
	origFetcher := htmlPageFetcher
	t.Cleanup(func() { htmlPageFetcher = origFetcher })

	t.Run("Fetcher error is propagated", func(t *testing.T) {
		htmlPageFetcher = func(_ string) (string, []string, error) {
			return "", nil, fmt.Errorf("connection refused")
		}
		_, err := resolvePlayableURL("https://lis.tn/abc123")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "chromedp error") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("SoundCloud anchor found in page HTML", func(t *testing.T) {
		htmlPageFetcher = func(_ string) (string, []string, error) {
			return `<a href="https://soundcloud.com/artist/track">SC</a>`, nil, nil
		}
		result, err := resolvePlayableURL("https://lis.tn/abc123")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != "https://soundcloud.com/artist/track" {
			t.Errorf("Expected SoundCloud URL, got %q", result)
		}
	})

	t.Run("YouTube URL found via regex", func(t *testing.T) {
		htmlPageFetcher = func(_ string) (string, []string, error) {
			return `<p>https://youtube.com/watch?v=abc123" </p>`, nil, nil
		}
		result, err := resolvePlayableURL("https://lis.tn/xyz")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.Contains(result, "youtube.com") {
			t.Errorf("Expected YouTube URL, got %q", result)
		}
	})

	t.Run("No playable URL found returns error", func(t *testing.T) {
		htmlPageFetcher = func(_ string) (string, []string, error) {
			return `<p>No music here</p>`, nil, nil
		}
		_, err := resolvePlayableURL("https://lis.tn/nothing")
		if err == nil {
			t.Fatal("Expected error when no URL found")
		}
		if !strings.Contains(err.Error(), "no playable URL found") {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Iframe fallback with SoundCloud URL", func(t *testing.T) {
		origIframe := iframeHTMLFetcher
		t.Cleanup(func() { iframeHTMLFetcher = origIframe })

		// Main page has no playable URL but has iframe sources
		htmlPageFetcher = func(_ string) (string, []string, error) {
			return `<p>No direct link</p>`, []string{"https://widget.example.com/player"}, nil
		}
		// Iframe contains the SoundCloud link
		iframeHTMLFetcher = func(_ string) (string, error) {
			return `<a href="https://soundcloud.com/artist/track-from-iframe">SC</a>`, nil
		}

		result, err := resolvePlayableURL("https://lis.tn/iframe-test")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != "https://soundcloud.com/artist/track-from-iframe" {
			t.Errorf("Expected SoundCloud URL from iframe, got %q", result)
		}
	})

	t.Run("Iframe fetch error is skipped", func(t *testing.T) {
		origIframe := iframeHTMLFetcher
		t.Cleanup(func() { iframeHTMLFetcher = origIframe })

		htmlPageFetcher = func(_ string) (string, []string, error) {
			return `<p>No direct link</p>`, []string{"https://failing-iframe.com"}, nil
		}
		iframeHTMLFetcher = func(_ string) (string, error) {
			return "", fmt.Errorf("iframe connection refused")
		}

		_, err := resolvePlayableURL("https://lis.tn/iframe-fail")
		if err == nil {
			t.Fatal("Expected error (no URL found after skipping failed iframe)")
		}
		if !strings.Contains(err.Error(), "no playable URL found") {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Empty iframe src is skipped", func(t *testing.T) {
		origIframe := iframeHTMLFetcher
		t.Cleanup(func() { iframeHTMLFetcher = origIframe })

		htmlPageFetcher = func(_ string) (string, []string, error) {
			return `<p>No links</p>`, []string{""}, nil
		}
		iframeHTMLFetcher = func(_ string) (string, error) {
			t.Error("iframeHTMLFetcher should not be called for empty src")
			return "", nil
		}

		_, err := resolvePlayableURL("https://lis.tn/empty-iframe")
		if err == nil || !strings.Contains(err.Error(), "no playable URL found") {
			t.Errorf("Expected 'no playable URL found', got: %v", err)
		}
	})
}

// TestDownloadFromCSV_EdgeCases tests branches not covered by the main test.
func TestDownloadFromCSV_EdgeCases(t *testing.T) {
	t.Run("Short row skipped when col beyond record length", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "short.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		// col=5, but each row only has 2 fields
		csvContent := "Title,Artist\nTrack1,Artist1\nTrack2,Artist2\n"
		_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

		result, err := DownloadFromCSV(csvPath, 5, outDir, "fake-yt-dlp", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.TotalSongs != 0 {
			t.Errorf("Expected 0 songs (short rows skipped), got %d", result.TotalSongs)
		}
	})

	t.Run("Empty listen URL skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "empty_links.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		csvContent := "Title,Artist,Album,Label,Date,Listen\nTrack1,Artist1,,,2026,\n"
		_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

		result, err := DownloadFromCSV(csvPath, 5, outDir, "fake-yt-dlp", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.TotalSongs != 0 {
			t.Errorf("Expected 0 songs (empty links skipped), got %d", result.TotalSongs)
		}
	})

	t.Run("Already downloaded file skips download attempt", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "test.csv")
		outDir := filepath.Join(tmpDir, "downloads")
		_ = os.MkdirAll(outDir, 0755)

		// Pre-create the expected output file so it appears "already downloaded"
		preCreated := filepath.Join(outDir, "TrackAlreadyThere.mp3")
		_ = os.WriteFile(preCreated, []byte("fake mp3"), 0644)

		csvContent := "Title,Artist,Album,Label,Date,Listen\nTrackAlreadyThere,Artist1,,,2026,https://soundcloud.com/test/track\n"
		_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

		// Save and restore mock fetcher to avoid chromedp
		origFetcher := htmlPageFetcher
		t.Cleanup(func() { htmlPageFetcher = origFetcher })
		htmlPageFetcher = func(link string) (string, []string, error) {
			return `<a href="` + link + `">direct</a>`, nil, nil
		}

		result, err := DownloadFromCSV(csvPath, 5, outDir, "fake-yt-dlp", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.TotalSongs != 1 {
			t.Errorf("Expected 1 song, got %d", result.TotalSongs)
		}
		if result.Downloaded != 1 {
			t.Errorf("Expected 1 already-downloaded, got %d", result.Downloaded)
		}
	})

	t.Run("Successful download increments downloaded count", func(t *testing.T) {
		tmpDir := t.TempDir()
		csvPath := filepath.Join(tmpDir, "test.csv")
		outDir := filepath.Join(tmpDir, "downloads")

		// Use a URL that resolves without chromedp (not lis.tn)
		csvContent := "Title,Artist,Album,Label,Date,Listen\nMyTrack,Artist1,,,2026,https://soundcloud.com/test/track\n"
		_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

		// Use "echo" as yt-dlp — it always succeeds and ignores all flags
		result, err := DownloadFromCSV(csvPath, 5, outDir, "echo", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.TotalSongs != 1 {
			t.Errorf("Expected 1 song, got %d", result.TotalSongs)
		}
		if result.Downloaded != 1 {
			t.Errorf("Expected 1 downloaded (via echo), got %d", result.Downloaded)
		}
	})
}

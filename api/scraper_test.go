package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScrapeSingleDate(t *testing.T) {
	t.Run("Valid date scraping", func(t *testing.T) {
		// Skip network-dependent tests in CI or short mode
		if testing.Short() {
			t.Skip("Skipping network-dependent test in short mode")
		}

		// Test with a recent date (7 days ago)
		date := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

		tracks, err := ScrapeSingleDate(date)
		if err != nil {
			t.Skipf("Scraping failed (network issue or website down): %v", err)
		}

		if tracks == 0 {
			t.Skip("No tracks found (website might be down or no data for this date)")
		}

		t.Logf("Successfully scraped %d tracks for date %s", tracks, date)
	})

	t.Run("Invalid date format", func(t *testing.T) {
		_, err := ScrapeSingleDate("invalid-date")
		if err == nil {
			t.Error("Expected error for invalid date format, got nil")
		}
	})

	t.Run("Future date", func(t *testing.T) {
		futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
		_, err := ScrapeSingleDate(futureDate)

		// Should error for future dates
		if err == nil {
			t.Error("Expected error for future date, got nil")
		}
	})
}

func TestFileExists(t *testing.T) {
	t.Run("Existing file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.txt")
		os.WriteFile(tmpFile, []byte("test"), 0644)

		if !fileExists(tmpFile) {
			t.Error("fileExists should return true for existing file")
		}
	})

	t.Run("Non-existing file", func(t *testing.T) {
		if fileExists("/nonexistent/file.txt") {
			t.Error("fileExists should return false for non-existing file")
		}
	})

	t.Run("Directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		if !fileExists(tmpDir) {
			t.Error("fileExists should return true for existing directory")
		}
	})
}

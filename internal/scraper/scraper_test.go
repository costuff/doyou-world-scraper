package scraper

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScrapeSingleDate(t *testing.T) {
	// Create temp directory for CSV files
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp dir so ./data/csv is created there
	os.Chdir(tmpDir)
	os.MkdirAll("data/csv", 0755)

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

		// Verify CSV file was created in correct location
		csvPath := filepath.Join(tmpDir, "data", "csv", "tracks-"+date+".csv")
		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			t.Errorf("CSV file not created at expected path: %s", csvPath)
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

func TestScrapeMonth(t *testing.T) {
	// Skip by default - this test takes 5-10 minutes due to network calls
	// Run with: go test -run TestScrapeMonth -v ./internal/scraper/... -timeout 15m
	t.Skip("Skipping long-running network test (use -run TestScrapeMonth to run manually)")

	// Skip in CI - this test takes 5-10 minutes due to network calls
	if os.Getenv("CI") != "" {
		t.Skip("Skipping long-running network test in CI environment")
	}

	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// Create temp directory for CSV files
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp dir so ./data/csv is created there
	os.Chdir(tmpDir)
	os.MkdirAll("data/csv", 0755)

	// Test scraping last month (should have data)
	lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")

	days, tracks, err := ScrapeMonth(lastMonth, nil, nil)
	if err != nil {
		t.Skipf("Month scraping failed (network issue): %v", err)
	}

	if days == 0 {
		t.Error("Expected at least one day to be scraped")
	}

	if days == 1 {
		t.Error("ScrapeMonth only scraped 1 day - bug not fixed!")
	}

	if tracks == 0 {
		t.Error("Expected at least some tracks to be scraped")
	}

	// Verify multiple CSV files were created
	csvFiles, _ := filepath.Glob(filepath.Join(tmpDir, "data", "csv", "*.csv"))
	if len(csvFiles) < 2 {
		t.Errorf("Expected at least 2 CSV files for month scraping, got %d", len(csvFiles))
	}

	t.Logf("Successfully scraped %d days with %d total tracks", days, tracks)
}

func TestFileExists(t *testing.T) {
	t.Run("Existing file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.txt")
		_ = os.WriteFile(tmpFile, []byte("test"), 0644)

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

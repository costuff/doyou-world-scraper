package scraper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---- fileExists ----

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

// ---- extractBetween ----

func TestExtractBetween(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    string
		end      string
		expected string
	}{
		{
			name:     "Normal extraction",
			input:    `<h1 class="title">Hello World</h1>`,
			start:    `<h1 class="title">`,
			end:      `</h1>`,
			expected: "Hello World",
		},
		{
			name:     "With extra whitespace",
			input:    `<h1>  Spaced Text  </h1>`,
			start:    `<h1>`,
			end:      `</h1>`,
			expected: "Spaced Text",
		},
		{
			name:     "Start not found",
			input:    `<h1>Hello</h1>`,
			start:    `<h2>`,
			end:      `</h2>`,
			expected: "",
		},
		{
			name:     "End not found",
			input:    `<h1>Hello`,
			start:    `<h1>`,
			end:      `</h1>`,
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			start:    `<h1>`,
			end:      `</h1>`,
			expected: "",
		},
		{
			name:     "Multiple occurrences (first match)",
			input:    `<span>First</span><span>Second</span>`,
			start:    `<span>`,
			end:      `</span>`,
			expected: "First",
		},
		{
			name:     "Nested tags",
			input:    `<div><span>Nested</span></div>`,
			start:    `<div>`,
			end:      `</div>`,
			expected: "<span>Nested</span>",
		},
		{
			name:     "Real HTML track title extraction",
			input:    `<h1 class="Track_title__rH4bg">Some Track Name</h1>`,
			start:    `<h1 class="Track_title__rH4bg">`,
			end:      `</h1>`,
			expected: "Some Track Name",
		},
		{
			name:     "Empty content between tags",
			input:    `<h1></h1>`,
			start:    `<h1>`,
			end:      `</h1>`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBetween(tt.input, tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("extractBetween() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ---- extractHref ----

func TestExtractHref(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    string
		end      string
		expected string
	}{
		{
			name:     "Normal href extraction",
			input:    `<a href="https://example.com" target="_blank">`,
			start:    `<a href="`,
			end:      `" target=`,
			expected: "https://example.com",
		},
		{
			name:     "URL with parameters",
			input:    `<a href="https://example.com?id=123&name=test" target="_blank">`,
			start:    `<a href="`,
			end:      `" target=`,
			expected: "https://example.com?id=123&name=test",
		},
		{
			name:     "Start not found",
			input:    `<div>No link here</div>`,
			start:    `<a href="`,
			end:      `" target=`,
			expected: "",
		},
		{
			name:     "End not found",
			input:    `<a href="https://example.com`,
			start:    `<a href="`,
			end:      `" target=`,
			expected: "",
		},
		{
			name:     "Empty href",
			input:    `<a href="" target="_blank">`,
			start:    `<a href="`,
			end:      `" target=`,
			expected: "",
		},
		{
			name:     "Multiple links (first match)",
			input:    `<a href="first.com">Link1</a><a href="second.com">Link2</a>`,
			start:    `<a href="`,
			end:      `">`,
			expected: "first.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHref(tt.input, tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("extractHref() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ---- ScrapeSingleDate date validation (no Chrome needed) ----

func TestScrapeSingleDate_InvalidDateFormat(t *testing.T) {
	_, err := ScrapeSingleDate("not-a-valid-date")
	if err == nil {
		t.Error("expected error for invalid date format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Errorf("expected 'invalid date format' in error, got: %v", err)
	}
}

func TestScrapeSingleDate_FutureDate(t *testing.T) {
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	_, err := ScrapeSingleDate(futureDate)
	if err == nil {
		t.Error("expected error for future date, got nil")
	}
	if !strings.Contains(err.Error(), "future") {
		t.Errorf("expected 'future' in error, got: %v", err)
	}
}

// ---- ScrapeMonth date validation (no Chrome needed) ----

func TestScrapeMonth_InvalidFormat(t *testing.T) {
	_, _, err := ScrapeMonth("not-a-month", nil, nil)
	if err == nil {
		t.Error("expected error for invalid month format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid month format") {
		t.Errorf("expected 'invalid month format' in error, got: %v", err)
	}
}

func TestScrapeMonth_BadFormat(t *testing.T) {
	_, _, err := ScrapeMonth("2026/01", nil, nil) // wrong separator
	if err == nil {
		t.Error("expected error for bad month format, got nil")
	}
}

// ---- ScrapeYear date validation (no Chrome needed) ----

func TestScrapeYear_InvalidFormat(t *testing.T) {
	_, _, err := ScrapeYear("not-a-year", nil, nil)
	if err == nil {
		t.Error("expected error for invalid year format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid year format") {
		t.Errorf("expected 'invalid year format' in error, got: %v", err)
	}
}

func TestScrapeYear_NegativeYear(t *testing.T) {
	_, _, err := ScrapeYear("-1", nil, nil)
	if err == nil {
		t.Error("expected error for negative year, got nil")
	}
}

func TestScrapeYear_ZeroYear(t *testing.T) {
	_, _, err := ScrapeYear("0", nil, nil)
	if err == nil {
		t.Error("expected error for year 0, got nil")
	}
}

// ---- ScrapeSingleDate network (skipped by default) ----

func TestScrapeSingleDate_Network(t *testing.T) {
	t.Skip("Skipping network test - run manually with -run TestScrapeSingleDate_Network if needed")
	date := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	tracks, err := ScrapeSingleDate(date)
	if err != nil {
		t.Skipf("Scraping failed (network/Chrome issue): %v", err)
	}
	t.Logf("Successfully scraped %d tracks for date %s", tracks, date)
}

func TestScrapeMonth_Network(t *testing.T) {
	t.Skip("Skipping long-running network test (use -run TestScrapeMonth_Network to run manually)")
	lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")
	days, tracks, err := ScrapeMonth(lastMonth, nil, nil)
	if err != nil {
		t.Skipf("Month scraping failed: %v", err)
	}
	t.Logf("Scraped %d days with %d total tracks", days, tracks)
}

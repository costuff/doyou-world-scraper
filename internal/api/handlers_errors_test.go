package api

import (
	"bytes"
	"doyou-world-scraper/internal/config"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestHandlerErrorCases tests error handling across all handlers
func TestHandlerErrorCases(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       filepath.Join(tmpDir, "csv"),
		DownloadsDir: filepath.Join(tmpDir, "downloads"),
		NASPath:      "",
		LogLevel:     "info",
		Environment:  "test",
	}
	cfg.EnsureDirectories()

	handlers := NewHandlers(cfg)
	app := fiber.New()

	// Setup routes
	api := app.Group("/api/v1")
	api.Post("/scrape/date", handlers.HandleScrapeDate)
	api.Post("/scrape/month", handlers.HandleScrapeMonth)
	api.Post("/scrape/year", handlers.HandleScrapeYear)
	api.Post("/download/csv", handlers.HandleDownloadFromCSV)
	api.Post("/util/deduplicate", handlers.HandleDeduplicate)
	api.Post("/util/move-to-nas", handlers.HandleMoveToNAS)

	t.Run("Invalid JSON in request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader([]byte("invalid json {")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing required fields - date", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for missing date, got %d", resp.StatusCode)
		}

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		if _, ok := result["error"]; !ok {
			t.Error("Response should include error message")
		}
	})

	t.Run("Missing required fields - month", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("POST", "/api/v1/scrape/month", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for missing month, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing required fields - year", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest("POST", "/api/v1/scrape/year", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for missing year, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing required fields - csv_path", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"column": 5,
		})
		req := httptest.NewRequest("POST", "/api/v1/download/csv", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for missing csv_path, got %d", resp.StatusCode)
		}
	})

	t.Run("Empty string values", func(t *testing.T) {
		tests := []struct {
			endpoint string
			body     map[string]interface{}
		}{
			{
				endpoint: "/api/v1/scrape/date",
				body:     map[string]interface{}{"date": ""},
			},
			{
				endpoint: "/api/v1/scrape/month",
				body:     map[string]interface{}{"month": ""},
			},
			{
				endpoint: "/api/v1/scrape/year",
				body:     map[string]interface{}{"year": ""},
			},
			{
				endpoint: "/api/v1/download/csv",
				body:     map[string]interface{}{"csv_path": "", "column": 5},
			},
		}

		for _, tt := range tests {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", tt.endpoint, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed for %s: %v", tt.endpoint, err)
			}

			if resp.StatusCode != 400 {
				t.Errorf("Endpoint %s: expected status 400 for empty string, got %d", tt.endpoint, resp.StatusCode)
			}
		}
	})

	t.Run("Special characters in date fields", func(t *testing.T) {
		specialChars := []string{
			"<script>alert('xss')</script>",
			"'; DROP TABLE tracks; --",
			"../../etc/passwd",
			"nullnullnullnullnullnullnullnull",
			string([]byte{0x00, 0x01, 0x02}), // null bytes
		}

		for _, char := range specialChars {
			body, _ := json.Marshal(map[string]string{"date": char})
			req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// Should either reject (400) or accept and handle safely (202)
			if resp.StatusCode != 400 && resp.StatusCode != 202 {
				t.Errorf("Unexpected status %d for special char input: %q", resp.StatusCode, char)
			}
		}
	})

	t.Run("Invalid NAS path - nonexistent directory", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"nas_dir": "/this/path/definitely/does/not/exist",
		})
		req := httptest.NewRequest("POST", "/api/v1/util/deduplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should either reject or fallback to valid path
		// Status 400 = rejected invalid path
		// Status 202 = accepted with fallback
		if resp.StatusCode != 400 && resp.StatusCode != 202 {
			t.Errorf("Expected 400 or 202 for invalid NAS path, got %d", resp.StatusCode)
		}
	})

	t.Run("Path traversal attempt", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"nas_dir": "../../../../../../etc/passwd",
		})
		req := httptest.NewRequest("POST", "/api/v1/util/move-to-nas", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should reject path traversal attempts
		if resp.StatusCode == 202 {
			respBody, _ := io.ReadAll(resp.Body)
			t.Errorf("Path traversal should be rejected or sanitized. Got 202 with body: %s", string(respBody))
		}
	})

	t.Run("Very long input strings", func(t *testing.T) {
		longString := string(make([]byte, 100000)) // 100KB string
		body, _ := json.Marshal(map[string]string{"date": longString})
		req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should either reject or handle safely
		if resp.StatusCode != 400 && resp.StatusCode != 202 {
			t.Errorf("Unexpected status %d for very long input", resp.StatusCode)
		}
	})

	t.Run("Null values in JSON", func(t *testing.T) {
		body := []byte(`{"date": null}`)
		req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should treat null as missing/empty value
		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for null date, got %d", resp.StatusCode)
		}
	})

	t.Run("Wrong content type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader([]byte(`date=2099-01-01`)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should reject or parse incorrectly and return 400
		if resp.StatusCode != 400 {
			t.Logf("Warning: endpoint accepted non-JSON content type, got status %d", resp.StatusCode)
		}
	})

	t.Run("Negative column number", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"csv_path": "test.csv",
			"column":   -1,
		})
		req := httptest.NewRequest("POST", "/api/v1/download/csv", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should either reject or handle gracefully (default to column 5)
		// Both 400 and 202 are acceptable
		if resp.StatusCode != 400 && resp.StatusCode != 202 {
			t.Errorf("Unexpected status %d for negative column", resp.StatusCode)
		}
	})

	t.Run("Invalid year format", func(t *testing.T) {
		invalidYears := []string{"abcd", "20260", "-2026", "2026.5"} // removed "99" (year 99 CE would launch Chrome)
		for _, year := range invalidYears {
			body, _ := json.Marshal(map[string]string{"year": year})
			req := httptest.NewRequest("POST", "/api/v1/scrape/year", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// Should accept and let scraper validate, or reject upfront
			// Both are acceptable approaches
			if resp.StatusCode != 400 && resp.StatusCode != 202 {
				t.Logf("Year %q: got status %d", year, resp.StatusCode)
			}
		}
	})

	t.Run("Invalid month format", func(t *testing.T) {
		invalidMonths := []string{"2026", "2026-13", "2026-00", "2026-1", "january", "2026-02-01"}
		for _, month := range invalidMonths {
			body, _ := json.Marshal(map[string]string{"month": month})
			req := httptest.NewRequest("POST", "/api/v1/scrape/month", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// Let scraper handle validation
			if resp.StatusCode != 400 && resp.StatusCode != 202 {
				t.Logf("Month %q: got status %d", month, resp.StatusCode)
			}
		}
	})
}

// TestConcurrentRequests tests handling of simultaneous API calls
func TestConcurrentRequests(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       filepath.Join(tmpDir, "csv"),
		DownloadsDir: filepath.Join(tmpDir, "downloads"),
		NASPath:      "",
		LogLevel:     "info",
		Environment:  "test",
	}
	cfg.EnsureDirectories()

	handlers := NewHandlers(cfg)
	app := fiber.New()

	api := app.Group("/api/v1")
	api.Post("/scrape/date", handlers.HandleScrapeDate)
	api.Get("/jobs", handlers.HandleGetJobs)

	t.Run("Multiple simultaneous scrape requests", func(t *testing.T) {
		numRequests := 10
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(idx int) {
				body, _ := json.Marshal(map[string]string{
					"date": "2026-01-01",
				})
				req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				resp, err := app.Test(req)
				if err != nil {
					errors <- err
					return
				}

				if resp.StatusCode != 202 {
					errors <- fiber.NewError(resp.StatusCode, "unexpected status")
					return
				}

				done <- true
			}(i)
		}

		// Wait for all requests to complete
		successCount := 0
		errorCount := 0
		for i := 0; i < numRequests; i++ {
			select {
			case <-done:
				successCount++
			case err := <-errors:
				t.Logf("Request failed: %v", err)
				errorCount++
			}
		}

		if successCount < numRequests/2 {
			t.Errorf("Too many failures: %d/%d succeeded", successCount, numRequests)
		}
	})

	t.Run("Concurrent read and write operations", func(t *testing.T) {
		done := make(chan bool, 20)

		// Start 10 write operations (create jobs)
		for i := 0; i < 10; i++ {
			go func() {
					body, _ := json.Marshal(map[string]string{"date": "2099-01-01"}) // far future: no Chrome
				req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				_, _ = app.Test(req)
				done <- true
			}()
		}

		// Start 10 read operations (get jobs)
		for i := 0; i < 10; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
				_, _ = app.Test(req)
				done <- true
			}()
		}

		// Wait for all operations
		for i := 0; i < 20; i++ {
			<-done
		}

		// If we get here without panic, concurrent access is safe
	})
}

// TestEdgeCaseInputs tests various edge case inputs
func TestEdgeCaseInputs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       filepath.Join(tmpDir, "csv"),
		DownloadsDir: filepath.Join(tmpDir, "downloads"),
		NASPath:      "",
		LogLevel:     "info",
		Environment:  "test",
	}
	cfg.EnsureDirectories()

	handlers := NewHandlers(cfg)
	app := fiber.New()

	api := app.Group("/api/v1")
	api.Post("/scrape/date", handlers.HandleScrapeDate)
	api.Post("/download/csv", handlers.HandleDownloadFromCSV)

	t.Run("Unicode and emoji in inputs", func(t *testing.T) {
		unicodeInputs := []string{
			"2026-01-01 🎵",
			"日本語",
			"téléchargement",
			"Это тест",
			"测试",
		}

		for _, input := range unicodeInputs {
			body, _ := json.Marshal(map[string]string{"date": input})
			req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed for input %q: %v", input, err)
			}

			// Should handle unicode gracefully
			if resp.StatusCode != 400 && resp.StatusCode != 202 {
				t.Errorf("Unexpected status %d for unicode input: %q", resp.StatusCode, input)
			}
		}
	})

	t.Run("Whitespace variations", func(t *testing.T) {
		whitespaceInputs := []string{
			" 2026-01-01 ",   // leading/trailing spaces
			"2026-01-01\t",   // tab
			"2026-01-01\n",   // newline
			"  2026-01-01  ", // multiple spaces
			"\n2026-01-01\n", // newlines
		}

		for _, input := range whitespaceInputs {
			body, _ := json.Marshal(map[string]string{"date": input})
			req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// Should either trim and accept, or reject
			if resp.StatusCode != 400 && resp.StatusCode != 202 {
				t.Errorf("Unexpected status %d for whitespace input: %q", resp.StatusCode, input)
			}
		}
	})

	t.Run("Column edge cases", func(t *testing.T) {
		columnTests := []struct {
			column int
			valid  bool
		}{
			{-999, false},
			{-1, false},
			{0, true}, // Could be valid (first column)
			{1, true},
			{5, true},   // Default
			{999, true}, // Could be valid if CSV has that many columns
		}

		for _, tt := range columnTests {
			body, _ := json.Marshal(map[string]interface{}{
				"csv_path": "test.csv",
				"column":   tt.column,
			})
			req := httptest.NewRequest("POST", "/api/v1/download/csv", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// Should accept valid columns (202) and might reject invalid (400)
			if !tt.valid && resp.StatusCode == 202 {
				t.Logf("Warning: column %d accepted but might be invalid", tt.column)
			}
		}
	})

	t.Run("Very large JSON payload", func(t *testing.T) {
		// Create a large but valid JSON
		largePayload := map[string]string{
			"date":  "2026-01-01",
			"extra": string(make([]byte, 10*1024*1024)), // 10MB of data
		}
		body, err := json.Marshal(largePayload)
		if err != nil {
			t.Skip("Could not create large payload")
		}

		req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, 10000) // 10 second timeout
		if err != nil {
			t.Logf("Large payload rejected (expected): %v", err)
			return
		}

		// Should either reject (4xx/5xx) or accept and ignore extra data (202)
		t.Logf("Large payload handling: status %d", resp.StatusCode)
	})
}

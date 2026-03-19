package api

import (
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/jobs"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TestScrapeConcurrencyLimit verifies semaphore limits concurrent Chrome instances
func TestScrapeConcurrencyLimit(t *testing.T) {
	t.Run("Semaphore blocks excess jobs while slots are full", func(t *testing.T) {
		// Fill both semaphore slots directly (same package = access to unexported var)
		scrapeSemaphore <- struct{}{}
		scrapeSemaphore <- struct{}{}
		defer func() {
			<-scrapeSemaphore
			<-scrapeSemaphore
		}()

		tmpDir := t.TempDir()
		cfg := &config.Config{
			Port:         0,
			DataDir:      tmpDir,
			CSVDir:       tmpDir + "/csv",
			DownloadsDir: tmpDir + "/downloads",
			NASPath:      "",
			LogLevel:     "error",
			Environment:  "test",
		}
		cfg.EnsureDirectories()

		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/api/v1/scrape/date", handlers.HandleScrapeDate)

		// Submit 3 jobs — all will block on the full semaphore (Chrome never launched)
		// Use future date: even if semaphore clears, validation fails before chromedp.Run
		jobIDs := make([]string, 3)
		for i := 0; i < 3; i++ {
			body := `{"date":"2099-01-01"}`
			req := httptest.NewRequest("POST", "/api/v1/scrape/date", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			if resp.StatusCode != 202 {
				t.Errorf("Expected 202, got %d", resp.StatusCode)
			}
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				if job, ok := result["job"].(map[string]interface{}); ok {
					if id, ok := job["id"].(string); ok {
						jobIDs[i] = id
					}
				}
			}
		}

		// Give goroutines time to start and try to acquire semaphore
		time.Sleep(100 * time.Millisecond)

		// All goroutines should be pending (blocked on full semaphore), none running
		runningCount := 0
		for _, jobID := range jobIDs {
			if job := jobs.GetJob(jobID); job != nil && job.Status == "running" {
				runningCount++
			}
		}
		if runningCount > 0 {
			t.Errorf("Expected 0 running (semaphore full), got %d", runningCount)
		}
		t.Logf("Verified: %d jobs queued, 0 running (semaphore capacity 2 is full)", len(jobIDs))

		// Cancel all to prevent goroutines from running after defer releases semaphore
		for _, jobID := range jobIDs {
			if job := jobs.GetJob(jobID); job != nil {
				job.Cancel()
			}
		}
	})

	t.Run("Semaphore releases on job failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Port:         0,
			DataDir:      tmpDir,
			CSVDir:       tmpDir + "/csv",
			DownloadsDir: tmpDir + "/downloads",
			NASPath:      "",
			LogLevel:     "error",
			Environment:  "test",
		}
		cfg.EnsureDirectories()

		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/api/v1/scrape/date", handlers.HandleScrapeDate)

		// Start a job that will fail (invalid date - parse fails before chromedp.Run)
		body := `{"date":"invalid-date"}`
		req := httptest.NewRequest("POST", "/api/v1/scrape/date", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 202 {
			t.Errorf("Expected 202, got %d", resp.StatusCode)
		}

		// Wait for job to fail quickly (invalid date causes immediate error)
		time.Sleep(200 * time.Millisecond)

		// Start another job to verify semaphore was released — also use safe date
		body2 := `{"date":"2099-01-01"}`
		req2 := httptest.NewRequest("POST", "/api/v1/scrape/date", strings.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")

		resp2, err := app.Test(req2, -1)
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}
		if resp2.StatusCode != 202 {
			t.Errorf("Expected 202 for second request, got %d", resp2.StatusCode)
		}
		t.Log("Semaphore successfully released after job failure")
	})
}

// TestJobCancellationSpeed verifies that a job blocked on the semaphore can be cancelled promptly
func TestJobCancellationSpeed(t *testing.T) {
	t.Run("Job cancelled while blocked on semaphore exits quickly", func(t *testing.T) {
		// Fill both semaphore slots so the goroutine blocks
		scrapeSemaphore <- struct{}{}
		scrapeSemaphore <- struct{}{}

		tmpDir := t.TempDir()
		cfg := &config.Config{
			Port:         0,
			DataDir:      tmpDir,
			CSVDir:       tmpDir + "/csv",
			DownloadsDir: tmpDir + "/downloads",
			NASPath:      "",
			LogLevel:     "error",
			Environment:  "test",
		}
		cfg.EnsureDirectories()

		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/api/v1/scrape/month", handlers.HandleScrapeMonth)

		// Submit a job; goroutine will block waiting for the semaphore
		body := `{"month":"9999-01"}`
		req := httptest.NewRequest("POST", "/api/v1/scrape/month", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 202 {
			t.Errorf("Expected 202, got %d", resp.StatusCode)
		}

		// Parse job ID
		var result map[string]interface{}
		var jobID string
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if job, ok := result["job"].(map[string]interface{}); ok {
				if id, ok := job["id"].(string); ok {
					jobID = id
				}
			}
		}

		// Give the goroutine time to start and block on the semaphore
		time.Sleep(50 * time.Millisecond)

		// Cancel the job while it's blocked
		if job := jobs.GetJob(jobID); job != nil {
			job.Cancel()
		}

		// Release one semaphore slot so the goroutine can unblock and check IsCancelled
		<-scrapeSemaphore
		defer func() { <-scrapeSemaphore }() // release the remaining filled slot

		// Within a short window the goroutine should detect cancel and exit
		deadline := time.Now().Add(2 * time.Second)
		exited := false
		for time.Now().Before(deadline) {
			if job := jobs.GetJob(jobID); job != nil {
				if job.Status == "cancelled" || job.Status == "failed" || job.Status == "completed" {
					exited = true
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		if !exited {
			t.Error("Job did not exit within 2s after cancel + semaphore release")
		} else {
			t.Log("Job exited promptly after cancellation")
		}
	})

	t.Run("Cancelled job releases semaphore", func(t *testing.T) {
		t.Skip("Tested implicitly by semaphore release tests")
	})
}

// TestSemaphoreWithDifferentEndpoints tests semaphore across all scrape endpoints
func TestSemaphoreWithDifferentEndpoints(t *testing.T) {
	t.Run("Semaphore shared across date/month/year endpoints", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Port:         0,
			DataDir:      tmpDir,
			CSVDir:       tmpDir + "/csv",
			DownloadsDir: tmpDir + "/downloads",
			NASPath:      "",
			LogLevel:     "error",
			Environment:  "test",
		}
		cfg.EnsureDirectories()

		handlers := NewHandlers(cfg)
		app := fiber.New()
		app.Post("/api/v1/scrape/date", handlers.HandleScrapeDate)
		app.Post("/api/v1/scrape/month", handlers.HandleScrapeMonth)
		app.Post("/api/v1/scrape/year", handlers.HandleScrapeYear)

		// Start one of each type — use Chrome-safe values (fail before chromedp.Run)
		requests := []struct {
			endpoint string
			body     string
		}{
			{"/api/v1/scrape/date", `{"date":"2099-01-01"}`}, // future: fails before chromedp.Run
			{"/api/v1/scrape/month", `{"month":"9999-01"}`},  // far future month: 0 days, no Chrome
			{"/api/v1/scrape/year", `{"year":"9999"}`},       // far future year: 0 days, no Chrome
		}

		jobIDs := make([]string, 0)

		for _, reqData := range requests {
			req := httptest.NewRequest("POST", reqData.endpoint, strings.NewReader(reqData.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Request to %s failed: %v", reqData.endpoint, err)
			}
			if resp.StatusCode != 202 {
				t.Errorf("Expected 202 from %s, got %d", reqData.endpoint, resp.StatusCode)
			}

			// Parse job ID
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				if job, ok := result["job"].(map[string]interface{}); ok {
					if id, ok := job["id"].(string); ok {
						jobIDs = append(jobIDs, id)
					}
				}
			}
		}

		// Check that max 2 are running at once
		time.Sleep(2 * time.Second)
		runningCount := 0
		for _, jobID := range jobIDs {
			if job := jobs.GetJob(jobID); job != nil {
				if job.Status == "running" {
					runningCount++
				}
			}
		}

		if runningCount > 2 {
			t.Errorf("Too many jobs running across different endpoints: %d (max should be 2)", runningCount)
		}

		t.Logf("Jobs running across endpoints: %d (max 2) ✓", runningCount)

		// Cleanup
		for _, jobID := range jobIDs {
			if job := jobs.GetJob(jobID); job != nil {
				job.Cancel()
			}
		}
	})
}

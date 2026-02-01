package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	app := fiber.New()

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "DoYou World Radio Scraper API is running",
		})
	})

	scrapeGroup := api.Group("/scrape")
	scrapeGroup.Post("/date", handleScrapeDate)
	scrapeGroup.Post("/month", handleScrapeMonth)
	scrapeGroup.Post("/year", handleScrapeYear)

	downloadGroup := api.Group("/download")
	downloadGroup.Post("/csv", handleDownloadFromCSV)
	downloadGroup.Post("/all", handleDownloadAll)

	utilGroup := api.Group("/util")
	utilGroup.Post("/deduplicate", handleDeduplicate)
	utilGroup.Post("/match", handleMatchDownloads)
	utilGroup.Post("/clear-csvs", handleClearCSVs)
	utilGroup.Get("/stats", handleStats)

	api.Get("/jobs", handleGetJobs)
	api.Get("/jobs/:id", handleGetJob)

	return app
}

func TestHealthEndpoint(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing health endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["status"] != "ok" {
		t.Error("Health check should return status 'ok'")
	}
}

func TestScrapeDateEndpoint(t *testing.T) {
	app := setupTestApp()

	payload := map[string]string{
		"date": "2026-01-27",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing scrape date endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(responseBody, &result)

	if result["message"] != "Scraping started" {
		t.Error("Should return 'Scraping started' message")
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "scrape_date" {
			t.Error("Job type should be 'scrape_date'")
		}
		if job["status"] != "running" {
			t.Error("Job status should be 'running'")
		}
	} else {
		t.Error("Response should include job object")
	}
}

func TestScrapeDateEndpointMissingDate(t *testing.T) {
	app := setupTestApp()

	payload := map[string]string{}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/scrape/date", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing scrape date endpoint: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(responseBody, &result)

	if result["error"] != "Date is required" {
		t.Error("Should return error message for missing date")
	}
}

func TestDownloadFromCSVEndpoint(t *testing.T) {
	app := setupTestApp()

	payload := map[string]interface{}{
		"csv_path": "out/tracks-2026-01-27.csv",
		"column":   5,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/download/csv", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing download CSV endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	responseBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(responseBody, &result)

	if result["message"] != "Download started" {
		t.Error("Should return 'Download started' message")
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "download_csv" {
			t.Error("Job type should be 'download_csv'")
		}
	}
}

func TestStatsEndpoint(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/util/stats", nil)
	resp, err := app.Test(req, 5000) // 5 second timeout

	if err != nil {
		t.Fatalf("Error testing stats endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Check that stats contains expected fields
	expectedFields := []string{
		"total_csvs",
		"total_tracks_scraped",
		"total_downloads",
		"downloads_size_gb",
		"nas_downloads",
		"nas_size_gb",
		"earliest_date",
		"latest_date",
	}

	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Stats should include '%s' field", field)
		}
	}
}

func TestGetJobsEndpoint(t *testing.T) {
	app := setupTestApp()

	// Clear jobs and create test jobs
	jobsMu.Lock()
	jobs = make(map[string]*Job)
	jobsMu.Unlock()

	NewJob("test_job_1", fiber.Map{})
	NewJob("test_job_2", fiber.Map{})

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing jobs endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if total, ok := result["total"].(float64); !ok || total != 2 {
		t.Errorf("Expected total 2, got %v", result["total"])
	}
}

func TestGetJobByIDEndpoint(t *testing.T) {
	app := setupTestApp()

	// Clear jobs and create test job
	jobsMu.Lock()
	jobs = make(map[string]*Job)
	jobsMu.Unlock()

	testJob := NewJob("test_job", fiber.Map{"key": "value"})

	req := httptest.NewRequest("GET", "/api/v1/jobs/"+testJob.ID, nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing get job endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["id"] != testJob.ID {
		t.Error("Should return correct job ID")
	}

	if result["type"] != "test_job" {
		t.Error("Should return correct job type")
	}
}

func TestGetJobByIDNotFound(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/jobs/nonexistent-id", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing get job endpoint: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["error"] != "Job not found" {
		t.Error("Should return 'Job not found' error")
	}
}

func TestClearCSVsEndpoint(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/util/clear-csvs", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing clear-csvs endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["message"] != "Clearing CSV files" {
		t.Error("Should return 'Clearing CSV files' message")
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "clear_csvs" {
			t.Error("Job type should be 'clear_csvs'")
		}
	}
}

func TestCancelJobEndpoint(t *testing.T) {
	app := setupTestApp()

	// Clear jobs and create test job
	jobsMu.Lock()
	jobs = make(map[string]*Job)
	jobsMu.Unlock()

	testJob := NewJob("test_cancel_job", fiber.Map{})

	// Add cancel route to test app
	api := app.Group("/api/v1")
	api.Post("/jobs/:id/cancel", handleCancelJob)

	req := httptest.NewRequest("POST", "/api/v1/jobs/"+testJob.ID+"/cancel", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing cancel job endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["message"] != "Job cancelled successfully" {
		t.Error("Should return 'Job cancelled successfully' message")
	}

	// Verify job was actually cancelled
	if !testJob.IsCancelled() {
		t.Error("Job should be marked as cancelled")
	}
}

func TestCancelJobNotFound(t *testing.T) {
	app := setupTestApp()

	// Add cancel route to test app
	api := app.Group("/api/v1")
	api.Post("/jobs/:id/cancel", handleCancelJob)

	req := httptest.NewRequest("POST", "/api/v1/jobs/nonexistent-id/cancel", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing cancel job endpoint: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["error"] != "Job not found" {
		t.Error("Should return 'Job not found' error")
	}
}

func TestScrapeMonthEndpoint(t *testing.T) {
	app := setupTestApp()

	payload := map[string]string{
		"month": "2026-01",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/scrape/month", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)

	if err != nil {
		t.Fatalf("Error testing scrape month endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	// Just check that we got a job response, don't verify exact message
	if job, ok := result["job"].(map[string]interface{}); !ok || job["type"] != "scrape_month" {
		t.Error("Should return scrape_month job")
	}
}

func TestScrapeYearEndpoint(t *testing.T) {
	app := setupTestApp()

	payload := map[string]string{
		"year": "2025",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/scrape/year", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)

	if err != nil {
		t.Fatalf("Error testing scrape year endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	// Just check that we got a job response, don't verify exact message
	if job, ok := result["job"].(map[string]interface{}); !ok || job["type"] != "scrape_year" {
		t.Error("Should return scrape_year job")
	}
}

func TestDownloadAllEndpoint(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/download/all", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing download all endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["message"] != "Downloading from all CSVs" {
		t.Error("Should return 'Downloading from all CSVs' message")
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "download_all" {
			t.Error("Job type should be 'download_all'")
		}
	}
}

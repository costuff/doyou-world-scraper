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
	resp, err := app.Test(req)

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
		"total_csv_files",
		"total_downloads",
		"downloads_dir_size_mb",
		"nas_dir_size_mb",
		"last_scrape",
		"last_download",
	}

	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
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

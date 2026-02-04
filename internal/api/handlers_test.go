package api

import (
	"bytes"
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/jobs"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	// Create test config
	cfg := &config.Config{
		Port:         3001,
		DataDir:      "./test_data",
		CSVDir:       "./test_data/csv",
		DownloadsDir: "./test_data/downloads",
		NASPath:      "/tmp/test_nas",
		LogLevel:     "info",
		Environment:  "test",
	}

	handlers := NewHandlers(cfg)

	app := fiber.New()

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "DoYou World Radio Scraper API is running",
		})
	})

	scrapeGroup := api.Group("/scrape")
	scrapeGroup.Post("/date", handlers.HandleScrapeDate)
	scrapeGroup.Post("/month", handlers.HandleScrapeMonth)
	scrapeGroup.Post("/year", handlers.HandleScrapeYear)

	downloadGroup := api.Group("/download")
	downloadGroup.Post("/csv", handlers.HandleDownloadFromCSV)
	downloadGroup.Post("/all", handlers.HandleDownloadAll)

	utilGroup := api.Group("/util")
	utilGroup.Post("/deduplicate", handlers.HandleDeduplicate)
	utilGroup.Post("/match", handlers.HandleMatchDownloads)
	utilGroup.Post("/clear-csvs", handlers.HandleClearCSVs)
	utilGroup.Post("/clear-downloads", handlers.HandleClearDownloads)
	utilGroup.Get("/stats", handlers.HandleStats)

	api.Get("/jobs", handlers.HandleGetJobs)
	api.Get("/jobs/:id", handlers.HandleGetJob)
	api.Post("/jobs/:id/cancel", handlers.HandleCancelJob)

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
	_ = json.Unmarshal(body, &result)

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
	_ = json.Unmarshal(responseBody, &result)

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
	_ = json.Unmarshal(responseBody, &result)

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
	_ = json.Unmarshal(responseBody, &result)

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
	_ = json.Unmarshal(body, &result)

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
	jobs.ClearAllJobs()

	jobs.NewJob("test_job_1", fiber.Map{})
	jobs.NewJob("test_job_2", fiber.Map{})

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
	_ = json.Unmarshal(body, &result)

	if total, ok := result["total"].(float64); !ok || total != 2 {
		t.Errorf("Expected total 2, got %v", result["total"])
	}
}

func TestGetJobByIDEndpoint(t *testing.T) {
	app := setupTestApp()

	// Clear jobs and create test job
	jobs.ClearAllJobs()

	testJob := jobs.NewJob("test_job", fiber.Map{"key": "value"})

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

func TestClearDownloadsEndpoint(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/util/clear-downloads", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing clear-downloads endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["message"] != "Clearing downloaded files" {
		t.Errorf("Expected 'Clearing downloaded files', got '%v'", result["message"])
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "clear_downloads" {
			t.Errorf("Job type should be 'clear_downloads', got '%v'", job["type"])
		}
		if job["status"] != "running" {
			t.Errorf("Job status should be 'running', got '%v'", job["status"])
		}
	} else {
		t.Error("Response should contain job object")
	}
}

func TestCancelJobEndpoint(t *testing.T) {
	app := setupTestApp()

	// Clear jobs and create test job
	jobs.ClearAllJobs()

	testJob := jobs.NewJob("test_cancel_job", fiber.Map{})

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

func TestHandleDeduplicate(t *testing.T) {
	app := setupTestApp()

	// Use "skip" to bypass NAS validation for this test
	payload := map[string]string{
		"nas_dir": "skip",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/util/deduplicate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing deduplicate endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if result["message"] != "Deduplication started" {
		t.Errorf("Expected 'Deduplication started', got %v", result["message"])
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "deduplicate" {
			t.Error("Job type should be 'deduplicate'")
		}
	}
}

func TestHandleMoveToNAS(t *testing.T) {
	// Create temporary NAS directory
	tmpNASDir := t.TempDir()

	cfg := &config.Config{
		Port:         3001,
		DataDir:      "./test_data",
		CSVDir:       "./test_data/csv",
		DownloadsDir: "./test_data/downloads",
		NASPath:      tmpNASDir,
		LogLevel:     "info",
		Environment:  "test",
	}

	handlers := NewHandlers(cfg)
	app := fiber.New()

	api := app.Group("/api/v1")
	utilGroup := api.Group("/util")
	utilGroup.Post("/move-to-nas", handlers.HandleMoveToNAS)

	// Send NAS directory in payload
	payload := map[string]string{
		"nas_dir": tmpNASDir,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/util/move-to-nas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing move-to-nas endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if result["message"] != "Moving files to NAS" {
		t.Errorf("Expected 'Moving files to NAS', got %v", result["message"])
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "move_to_nas" {
			t.Error("Job type should be 'move_to_nas'")
		}
	}
}

func TestHandleMatchDownloads(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/util/match", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Error testing match endpoint: %v", err)
	}

	if resp.StatusCode != 202 {
		t.Errorf("Expected status 202, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["message"] != "Matching downloads to CSVs" {
		t.Errorf("Expected 'Matching downloads to CSVs', got %v", result["message"])
	}

	if job, ok := result["job"].(map[string]interface{}); ok {
		if job["type"] != "match_downloads" {
			t.Error("Job type should be 'match_downloads'")
		}
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

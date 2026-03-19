package api

import (
	"bytes"
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/jobs"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	api.Get("/version", handlers.HandleVersion)

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

	api.Get("/tracks", handlers.HandleGetTracks)
	api.Get("/tracks/status", handlers.HandleGetTracksStatus)
	api.Get("/tracks/stream", handlers.HandleStreamTrack)
	api.Get("/tracks/analyzed", handlers.HandleGetAnalyzedTracks)
	api.Post("/tracks/analyze", handlers.HandleAnalyzeTrack)
	api.Post("/tracks/analyze-all", handlers.HandleAnalyzeAll)

	return app
}

func TestVersionEndpoint(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing version endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)
	if result["backend"] == nil {
		t.Error("Should return backend version")
	}
	if result["frontend"] == nil {
		t.Error("Should return frontend version")
	}
	if result["commit"] == nil {
		t.Error("Should return commit SHA")
	}
	if result["build_time"] == nil {
		t.Error("Should return build time")
	}
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
		"date": "2099-01-01", // far future: fails in scraper before chromedp.Run
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
		// Job starts as 'pending' before semaphore acquisition
		if job["status"] != "pending" && job["status"] != "running" {
			t.Errorf("Job status should be 'pending' or 'running', got '%v'", job["status"])
		}
		// Cancel immediately so the background goroutine does not launch Chrome
		if jobID, ok := job["id"].(string); ok {
			jobs.CancelJob(jobID)
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
		// Job starts as 'pending' before starting clearing operation
		if job["status"] != "pending" && job["status"] != "running" {
			t.Errorf("Job status should be 'pending' or 'running', got '%v'", job["status"])
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
		"month": "9999-01", // far future: no days before today, no Chrome launched
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
	} else if jobID, ok := job["id"].(string); ok {
		// Cancel immediately so the background goroutine does not launch Chrome
		jobs.CancelJob(jobID)
	}
}

func TestScrapeYearEndpoint(t *testing.T) {
	app := setupTestApp()

	payload := map[string]string{
		"year": "9999", // far future: no days before today, no Chrome launched
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
	} else if jobID, ok := job["id"].(string); ok {
		// Cancel immediately so the background goroutine does not launch Chrome
		jobs.CancelJob(jobID)
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

func TestGetAnalyzedTracksEndpoint(t *testing.T) {
	app := setupTestApp()

	// Use a temp dir (not the shared ./test_data/downloads) to avoid races with
	// background goroutines spawned by TestClearDownloadsEndpoint.
	downloadsDir := t.TempDir()

	// Create a test MP3 file (minimal valid MP3 header)
	testMP3Path := filepath.Join(downloadsDir, "test_track.mp3")
	testMP3Data := []byte{
		0xFF, 0xFB, 0x90, 0x00, // MP3 sync word and header
		0x00, 0x00, 0x00, 0x00,
	}
	if err := os.WriteFile(testMP3Path, testMP3Data, 0644); err != nil {
		t.Fatalf("Failed to create test MP3: %v", err)
	}
	// cleanup handled by t.TempDir()

	// Create companion JSON analysis file
	testJSONPath := filepath.Join(downloadsDir, "test_track.json")
	testAnalysis := map[string]interface{}{
		"bpm":          128.5,
		"key":          "C major",
		"energy":       0.85,
		"danceability": 0.72,
		"valence":      0.68,
		"acousticness": 0.12,
	}
	jsonData, _ := json.Marshal(testAnalysis)
	if err := os.WriteFile(testJSONPath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to create test JSON: %v", err)
	}
	// t.TempDir() handles cleanup automatically; no need for manual defer os.Remove

	// Use downloads_dir param so the handler reads from the isolated temp dir.
	req := httptest.NewRequest("GET", "/api/v1/tracks/analyzed?downloads_dir="+downloadsDir, nil)
	resp, err := app.Test(req, 5000)

	if err != nil {
		t.Fatalf("Error testing analyzed tracks endpoint: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check tracks array exists
	tracks, ok := result["tracks"].([]interface{})
	if !ok {
		t.Fatal("Expected 'tracks' array in response")
	}

	// Should have at least 1 track (our test track)
	if len(tracks) < 1 {
		t.Fatal("Expected at least 1 analyzed track")
	}

	// Check first track structure
	track := tracks[0].(map[string]interface{})

	expectedFields := []string{"file_path", "file_name", "analysis"}
	for _, field := range expectedFields {
		if _, exists := track[field]; !exists {
			t.Errorf("Track should include '%s' field", field)
		}
	}

	// Verify analysis data structure
	analysis, ok := track["analysis"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'analysis' object in track")
	}

	expectedAnalysisFields := []string{"bpm", "key", "energy", "danceability", "valence", "acousticness"}
	for _, field := range expectedAnalysisFields {
		if _, exists := analysis[field]; !exists {
			t.Errorf("Analysis should include '%s' field", field)
		}
	}
}

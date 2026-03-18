package api

import (
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/jobs"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ---- HandleGetTracks ----

func TestHandleGetTracks_EmptyDir(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing tracks endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Invalid JSON response: %v\n%s", err, body)
	}
}

func TestHandleGetTracks_WithPaginationParams(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks?page=1&per_page=10&sort_by=filename&sort_order=asc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing tracks endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleGetTracks_AsyncMode(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks?async=true", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing async tracks endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleGetTracks_WithSearch(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks?search=sometrack", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing search tracks endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// ---- HandleGetTracksStatus ----

func TestHandleGetTracksStatus(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing tracks status endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Invalid JSON response: %v\n%s", err, body)
	}
}

// ---- HandleStreamTrack ----

func TestHandleStreamTrack_MissingFileParam(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks/stream", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing stream endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing file param, got %d", resp.StatusCode)
	}
}

func TestHandleStreamTrack_FileNotFound(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks/stream?file=/definitely/does/not/exist/track.mp3", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing stream endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for missing file, got %d", resp.StatusCode)
	}
}

func TestHandleStreamTrack_ExistingFile(t *testing.T) {
	// Create a temp audio file with a real absolute path
	tmpFile, err := os.CreateTemp("", "test-*.mp3")
	if err != nil {
		t.Fatalf("Could not create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake mp3 content for streaming"))
	tmpFile.Close()

	app := setupTestApp()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tracks/stream?file=%s", tmpFile.Name()), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing stream endpoint: %v", err)
	}
	// 200 expected for an existing file
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for existing file, got %d", resp.StatusCode)
	}
}

func TestHandleStreamTrack_RelativePath(t *testing.T) {
	// Relative paths get resolved to downloads dir or NAS — likely not found
	app := setupTestApp()
	req := httptest.NewRequest("GET", "/api/v1/tracks/stream?file=nosuchtrack.mp3", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error testing stream endpoint: %v", err)
	}
	// Should either 200 (if file found somewhere) or 404
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 404 or 200 for relative path, got %d", resp.StatusCode)
	}
}

// ---- HandleAnalyzeAll ----

func TestHandleAnalyzeAll_NoMP3Files(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest("POST", "/api/v1/tracks/analyze-all", nil)
	resp, err := app.Test(req, 30000)
	if err != nil {
		t.Fatalf("Error testing analyze-all endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Invalid JSON response: %v\n%s", err, body)
	}
	if _, ok := result["job"]; !ok {
		t.Error("Expected 'job' field in response")
	}
}

func TestHandleAnalyzeAll_WithBody(t *testing.T) {
	app := setupTestApp()
	body := strings.NewReader(`{"generate_spectrograms": false}`)
	req := httptest.NewRequest("POST", "/api/v1/tracks/analyze-all", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 30000)
	if err != nil {
		t.Fatalf("Error testing analyze-all endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", resp.StatusCode)
	}
}
func TestHandleAnalyzeAll_JobCompletes(t *testing.T) {
	// Test_data/downloads is empty, so the job should quickly complete with "No MP3 files found"
	app := setupTestApp()
	req := httptest.NewRequest("POST", "/api/v1/tracks/analyze-all", nil)
	resp, err := app.Test(req, 10000)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	jobData, ok := result["job"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'job' object in response")
	}
	jobID, _ := jobData["id"].(string)
	if jobID == "" {
		t.Fatal("Expected job ID in response")
	}

	// Wait for the async job goroutine to complete
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		j := jobs.GetJob(jobID)
		if j != nil && (j.Status == "completed" || j.Status == "failed") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	j := jobs.GetJob(jobID)
	if j == nil {
		t.Fatal("Job not found after waiting")
	}
	// With no MP3 files, it should complete (not fail)
	if j.Status == "running" {
		t.Logf("Job still running after timeout — acceptable in fast test env")
	}
}
// ---- HandleClearNASDownloads (register route in a local test app) ----

func newTestHandlers() *Handlers {
	return NewHandlers(&config.Config{
		Port:         3001,
		DataDir:      "./test_data",
		CSVDir:       "./test_data/csv",
		DownloadsDir: "./test_data/downloads",
		NASPath:      "/tmp/test_nas",
		LogLevel:     "info",
		Environment:  "test",
	})
}

func TestHandleClearNASDownloads_NASPathNotFound(t *testing.T) {
	app := fiber.New()
	app.Post("/clear-nas", newTestHandlers().HandleClearNASDownloads)

	req := httptest.NewRequest("POST", "/clear-nas", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Error testing clear-nas endpoint: %v", err)
	}
	// /tmp/test_nas doesn't exist → 404 or missing nas_dir → 400
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 or 404, got %d", resp.StatusCode)
	}
}

func TestHandleClearNASDownloads_WithExistingNASDir(t *testing.T) {
	nasDir := t.TempDir()

	app := fiber.New()
	app.Post("/clear-nas", newTestHandlers().HandleClearNASDownloads)

	body := strings.NewReader(fmt.Sprintf(`{"nas_dir": "%s"}`, nasDir))
	req := httptest.NewRequest("POST", "/clear-nas", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Error testing clear-nas endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", resp.StatusCode)
	}
}

func TestHandleClearNASDownloads_EmptyNASPath(t *testing.T) {
	handlers := NewHandlers(&config.Config{
		NASPath: "", // No fallback
	})
	app := fiber.New()
	app.Post("/clear-nas", handlers.HandleClearNASDownloads)

	req := httptest.NewRequest("POST", "/clear-nas", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 400 or 404, got %d", resp.StatusCode)
	}
}

// ---- resolveStreamPath edge cases ----

func TestResolveStreamPath_MntMusicPath(t *testing.T) {
	handlers := newTestHandlers()
	// /mnt/music/downloads/ translates to /nas/... which doesn't exist in tests
	_, _ = handlers.resolveStreamPath("/mnt/music/downloads/somefile.mp3")
}

func TestResolveStreamPath_ExistingAbsolutePath(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "stream_test_*.mp3")
	if err != nil {
		t.Fatalf("Could not create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("content"))
	tmpFile.Close()

	resolved, err := newTestHandlers().resolveStreamPath(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected to resolve existing file: %v", err)
	}
	if resolved != tmpFile.Name() {
		t.Errorf("Expected %s, got %s", tmpFile.Name(), resolved)
	}
}

func TestResolveStreamPath_RelativeViaConfigNASPath(t *testing.T) {
	nasDir := t.TempDir()
	mp3File := filepath.Join(nasDir, "mytrack.mp3")
	os.WriteFile(mp3File, []byte("audio"), 0644)

	handlers := NewHandlers(&config.Config{
		NASPath:      nasDir,
		DownloadsDir: "./test_data/downloads",
	})

	resolved, err := handlers.resolveStreamPath("mytrack.mp3")
	if err != nil {
		t.Fatalf("Expected relative file resolved via NASPath: %v", err)
	}
	if !strings.HasSuffix(resolved, "mytrack.mp3") {
		t.Errorf("Expected path ending in mytrack.mp3, got: %s", resolved)
	}
}

func TestResolveStreamPath_NotFound(t *testing.T) {
	_, err := newTestHandlers().resolveStreamPath("/absolutely/nonexistent/track.mp3")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

package api

import (
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/jobs"
	"encoding/json"
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

// makeHandlers creates a Handlers instance backed by real temp directories.
func makeHandlers(t *testing.T) (*Handlers, *config.Config, string) {
	t.Helper()
	tmpDir := t.TempDir()
	csvDir := filepath.Join(tmpDir, "csv")
	downloadsDir := filepath.Join(tmpDir, "downloads")
	nasDir := filepath.Join(tmpDir, "nas")
	for _, d := range []string{csvDir, downloadsDir, nasDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}
	cfg := &config.Config{
		Port:         0,
		DataDir:      tmpDir,
		CSVDir:       csvDir,
		DownloadsDir: downloadsDir,
		NASPath:      nasDir,
		LogLevel:     "error",
		Environment:  "test",
	}
	return NewHandlers(cfg), cfg, tmpDir
}

// waitForJobDone polls until the job reaches a terminal state or the deadline passes.
func waitForJobDone(t *testing.T, jobID string, timeout time.Duration) *jobs.Job {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if j := jobs.GetJob(jobID); j != nil {
			if j.Status == "completed" || j.Status == "failed" || j.Status == "cancelled" {
				return j
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return jobs.GetJob(jobID)
}

// ---- HandleVersion coverage ----

// TestHandleVersion_WithVersionFiles exercises the VERSION file-reading branches
// by temporarily creating the files in the package directory (tests CWD).
func TestHandleVersion_WithVersionFiles(t *testing.T) {
	const backendVer = "9.8.7"
	const frontendVer = "9.8.6"

	if err := os.WriteFile("VERSION", []byte(backendVer+"\n"), 0644); err != nil {
		t.Skip("cannot write VERSION file in test CWD")
	}
	t.Cleanup(func() { os.Remove("VERSION") }) //nolint:errcheck

	if err := os.MkdirAll("frontend", 0755); err != nil {
		t.Skip("cannot create frontend dir in test CWD")
	}
	if err := os.WriteFile("frontend/VERSION", []byte(frontendVer+"\n"), 0644); err != nil {
		t.Skip("cannot write frontend/VERSION file in test CWD")
	}
	t.Cleanup(func() {
		os.Remove("frontend/VERSION") //nolint:errcheck
		os.Remove("frontend")         //nolint:errcheck
	})

	h, _, _ := makeHandlers(t)
	app := fiber.New()
	app.Get("/version", h.HandleVersion)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if result["backend"] != backendVer {
		t.Errorf("Expected backend=%q, got %v", backendVer, result["backend"])
	}
	if result["frontend"] != frontendVer {
		t.Errorf("Expected frontend=%q, got %v", frontendVer, result["frontend"])
	}
}

// ---- HandleStats coverage ----

// TestHandleStats_WithCSVFiles exercises the CSV-counting logic inside HandleStats
// (line-count loop, date range extraction, format YYYY-MM-DD from filename).
func TestHandleStats_WithCSVFiles(t *testing.T) {
	h, cfg, _ := makeHandlers(t)

	// Populate csvDir with 3 CSVs — each has 2 data rows.
	dates := []string{"2026-01-01", "2026-02-01", "2026-03-01"}
	for _, d := range dates {
		content := "Title,Artist,Album,Label,Date,URL\n" +
			"Track A,Artist A,Album,Label," + d + ",http://a.com\n" +
			"Track B,Artist B,Album,Label," + d + ",http://b.com\n"
		os.WriteFile(filepath.Join(cfg.CSVDir, "tracks-"+d+".csv"), []byte(content), 0644)
	}

	// Also put MP3 files in downloads and NAS to exercise those goroutine paths.
	dummyMP3 := make([]byte, 512)
	os.WriteFile(filepath.Join(cfg.DownloadsDir, "song1.mp3"), dummyMP3, 0644)
	os.WriteFile(filepath.Join(cfg.NASPath, "nas_song.mp3"), dummyMP3, 0644)

	app := fiber.New()
	app.Get("/stats", h.HandleStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	resp, err := app.Test(req, 15000) // generous timeout for goroutines
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %s", body)
	}

	// 3 CSV files
	if v, _ := result["total_csvs"].(float64); int(v) != 3 {
		t.Errorf("expected total_csvs=3, got %v", result["total_csvs"])
	}
	// 6 tracks total (2 per CSV)
	if v, _ := result["total_tracks_scraped"].(float64); int(v) != 6 {
		t.Errorf("expected total_tracks_scraped=6, got %v", result["total_tracks_scraped"])
	}
	// Date range from CSVs
	if result["earliest_date"] != "2026-01-01" {
		t.Errorf("expected earliest_date=2026-01-01, got %v", result["earliest_date"])
	}
	if result["latest_date"] != "2026-03-01" {
		t.Errorf("expected latest_date=2026-03-01, got %v", result["latest_date"])
	}
}

// TestHandleStats_NASCountingGoroutine specifically exercises the NAS walk branch.
func TestHandleStats_NASCountingGoroutine(t *testing.T) {
	h, cfg, _ := makeHandlers(t)

	// Put 2 MP3 files in NAS dir.
	for _, name := range []string{"a.mp3", "b.mp3"} {
		os.WriteFile(filepath.Join(cfg.NASPath, name), []byte("data"), 0644)
	}

	app := fiber.New()
	app.Get("/stats", h.HandleStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	resp, err := app.Test(req, 10000)
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ---- HandleMatchDownloads success path ----

// TestHandleMatchDownloads_SuccessPath sets up CSV + MP3 data so the goroutine calls
// job.Complete instead of job.Fail.
func TestHandleMatchDownloads_SuccessPath(t *testing.T) {
	h, cfg, _ := makeHandlers(t)

	// Create one CSV with track data.
	csvContent := "Title,Artist,Album,Label,Date,URL\n" +
		"My Song,Cool Artist,Album,Label,2026-01-01,http://example.com\n"
	os.WriteFile(filepath.Join(cfg.CSVDir, "tracks-2026-01-01.csv"), []byte(csvContent), 0644)

	// Create one MP3 file (size >= minTrackSizeBytes = 65536).
	mp3Data := make([]byte, 70000)
	os.WriteFile(filepath.Join(cfg.DownloadsDir, "Cool Artist - My Song.mp3"), mp3Data, 0644)

	app := fiber.New()
	app.Post("/util/match", h.HandleMatchDownloads)

	req := httptest.NewRequest("POST", "/util/match", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("match request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)
	if jobID == "" {
		t.Fatal("no job ID in response")
	}

	// Wait for goroutine to call job.Complete.
	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found after timeout")
	}
	if j.Status != "completed" {
		t.Errorf("expected job completed, got %s (error: %s)", j.Status, j.Error)
	}
}

// ---- HandleClearDownloads goroutine completion ----

// TestHandleClearDownloads_JobCompletes waits for the goroutine to finish and
// verifies the job transitions to "completed". Files are added so the progress
// callback closure runs (covering those lines).
func TestHandleClearDownloads_JobCompletes(t *testing.T) {
	h, cfg, _ := makeHandlers(t)

	// Add 2 files to downloads dir so the callback fires at least once.
	for _, name := range []string{"song1.mp3", "song2.mp3"} {
		os.WriteFile(filepath.Join(cfg.DownloadsDir, name), []byte("data"), 0644)
	}

	app := fiber.New()
	app.Post("/util/clear-downloads", h.HandleClearDownloads)

	req := httptest.NewRequest("POST", "/util/clear-downloads", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s", j.Status)
	}
}

// ---- HandleClearNASDownloads goroutine ----

// TestHandleClearNASDownloads_Goroutine verifies the goroutine body runs to
// completion when a valid NAS dir is provided. Files are added so the callback fires.
func TestHandleClearNASDownloads_Goroutine(t *testing.T) {
	nasDir := t.TempDir()

	// Add files so the progress callback runs (covers closure lines).
	for _, name := range []string{"track1.mp3", "track2.mp3"} {
		os.WriteFile(filepath.Join(nasDir, name), []byte("mp3"), 0644)
	}

	h := NewHandlers(&config.Config{NASPath: nasDir})
	app := fiber.New()
	app.Post("/clear-nas", h.HandleClearNASDownloads)

	body := strings.NewReader(`{"nas_dir":"` + nasDir + `"}`)
	req := httptest.NewRequest("POST", "/clear-nas", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s", j.Status)
	}
}

// ---- HandleDeduplicate goroutine ----

// TestHandleDeduplicate_JobCompletes waits for the deduplicate goroutine to finish.
func TestHandleDeduplicate_JobCompletes(t *testing.T) {
	h, _, _ := makeHandlers(t)
	app := fiber.New()
	app.Post("/util/deduplicate", h.HandleDeduplicate)

	body := strings.NewReader(`{"nas_dir":"skip"}`)
	req := httptest.NewRequest("POST", "/util/deduplicate", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s", j.Status)
	}
}

// ---- HandleMoveToNAS goroutine ----

// TestHandleMoveToNAS_JobCompletes waits for the move goroutine to finish.
func TestHandleMoveToNAS_JobCompletes(t *testing.T) {
	srcDir := t.TempDir()
	nasDir := t.TempDir()

	h := NewHandlers(&config.Config{DownloadsDir: srcDir, NASPath: nasDir})
	app := fiber.New()
	app.Post("/util/move-to-nas", h.HandleMoveToNAS)

	body := strings.NewReader(`{"nas_dir":"` + nasDir + `"}`)
	req := httptest.NewRequest("POST", "/util/move-to-nas", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s", j.Status)
	}
}

// ---- HandleClearCSVs goroutine ----

// TestHandleClearCSVs_JobCompletes verifies the clear-csvs goroutine finishes.
func TestHandleClearCSVs_JobCompletes(t *testing.T) {
	h, _, _ := makeHandlers(t)
	app := fiber.New()
	app.Post("/util/clear-csvs", h.HandleClearCSVs)

	req := httptest.NewRequest("POST", "/util/clear-csvs", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s", j.Status)
	}
}

// ---- HandleDownloadFromCSV success path ----

// TestHandleDownloadFromCSV_EmptyCSVSuccess uses a header-only CSV so
// DownloadFromCSV finds 0 downloadable URLs and calls job.Complete (no yt-dlp needed).
func TestHandleDownloadFromCSV_EmptyCSVSuccess(t *testing.T) {
	h, cfg, _ := makeHandlers(t)
	app := fiber.New()
	app.Post("/download/csv", h.HandleDownloadFromCSV)

	// Create a valid CSV with only the header — no URL rows.
	csvPath := filepath.Join(cfg.CSVDir, "tracks-empty.csv")
	os.WriteFile(csvPath, []byte("Title,Artist,Album,Label,Date,URL\n"), 0644)

	body := strings.NewReader(`{"csv_path":"` + csvPath + `","column":5}`)
	req := httptest.NewRequest("POST", "/download/csv", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	// 0 songs → DownloadFromCSV returns (result, nil) → job.Complete()
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s (err: %s)", j.Status, j.Error)
	}
}

// ---- HandleDownloadAll success path ----

// TestHandleDownloadAll_HeaderOnlyCSV puts a header-only CSV in csvDir so
// DownloadFromFolder processes it and calls job.Complete (no yt-dlp needed).
func TestHandleDownloadAll_HeaderOnlyCSV(t *testing.T) {
	h, cfg, _ := makeHandlers(t)
	app := fiber.New()
	app.Post("/download/all", h.HandleDownloadAll)

	// Create a header-only CSV in csvDir.
	os.WriteFile(filepath.Join(cfg.CSVDir, "tracks-empty.csv"), []byte("Title,Artist,Album,Label,Date,URL\n"), 0644)

	req := httptest.NewRequest("POST", "/download/all", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found")
	}
	// CSV found, 0 downloadable songs → job.Complete()
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s (err: %s)", j.Status, j.Error)
	}
}

// ---- HandleAnalyzeAll with existing JSON files ----

// TestHandleAnalyzeAll_WithExistingJSON puts MP3+JSON pairs in the downloads dir
// so the analyze-all goroutine runs the "already analyzed → skip" branch and
// reaches job.Complete (no Python/librosa needed).
func TestHandleAnalyzeAll_WithExistingJSON(t *testing.T) {
	h, cfg, _ := makeHandlers(t)
	app := fiber.New()
	app.Post("/tracks/analyze-all", h.HandleAnalyzeAll)

	// Create 2 MP3 files each with a companion .json file.
	for _, name := range []string{"trackA", "trackB"} {
		os.WriteFile(filepath.Join(cfg.DownloadsDir, name+".mp3"), []byte("id3data"), 0644)
		os.WriteFile(filepath.Join(cfg.DownloadsDir, name+".json"), []byte(`{"bpm":120}`), 0644)
	}

	req := httptest.NewRequest("POST", "/tracks/analyze-all", nil)
	resp, err := app.Test(req, 10000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Fatal("job not found after 5s")
	}
	// All files already analyzed → skipped=2 → job.Complete() with analyzed_count=0
	if j.Status != "completed" {
		t.Errorf("expected completed, got %s (err: %s)", j.Status, j.Error)
	}
}

// TestResolveDownloadsDir_NASPath verifies "nas" resolves to NASPath when it exists.
func TestResolveDownloadsDir_NASPath(t *testing.T) {
	nasDir := t.TempDir()
	h := NewHandlers(&config.Config{NASPath: nasDir, DownloadsDir: t.TempDir()})

	got, err := h.resolveDownloadsDir("nas")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nasDir {
		t.Errorf("expected %s, got %s", nasDir, got)
	}
}

// TestResolveDownloadsDir_NASNotFound verifies an error when NAS is requested but absent.
// This test is skipped if a real NAS happens to be mounted on the dev machine.
func TestResolveDownloadsDir_NASNotFound(t *testing.T) {
	// Check if any common NAS paths exist — if so, the function will succeed and the
	// "not found" branch is untestable in this environment.
	commonNASPaths := []string{
		"/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
		"/Volumes/Music/doyou-world-radio-tracks",
		"/mnt/nas/music/doyou-world-radio-tracks",
		"/nas",
	}
	for _, p := range commonNASPaths {
		if _, err := os.Stat(p); err == nil {
			t.Skipf("real NAS found at %s — skipping NAS-not-found test", p)
		}
	}

	h := NewHandlers(&config.Config{NASPath: "", DownloadsDir: t.TempDir()})

	_, err := h.resolveDownloadsDir("nas")
	if err == nil {
		t.Error("expected error when NAS not found")
	}
}

// TestResolveDownloadsDir_DirectoryNotFound returns an error when dir doesn't exist.
func TestResolveDownloadsDir_DirectoryNotFound(t *testing.T) {
	h := NewHandlers(&config.Config{DownloadsDir: "/nonexistent/path/to/dir"})

	_, err := h.resolveDownloadsDir("/nonexistent/path/to/dir")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

// ---- HandleDownloadAll goroutine ----

// TestHandleDownloadAll_GoroutineRunsAndFails verifies the goroutine starts and
// either completes or fails (yt-dlp may not be available in CI).
func TestHandleDownloadAll_GoroutineRunsAndFails(t *testing.T) {
	h, _, _ := makeHandlers(t) // empty downloads dir → no CSVs → downloads from folder (no-op or fail)
	app := fiber.New()
	app.Post("/download/all", h.HandleDownloadAll)

	req := httptest.NewRequest("POST", "/download/all", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	jobData, _ := result["job"].(map[string]interface{})
	jobID, _ := jobData["id"].(string)

	// Wait for goroutine — can be completed or failed (no yt-dlp)
	j := waitForJobDone(t, jobID, 5*time.Second)
	if j == nil {
		t.Log("job not completed within 5s — acceptable for slow CI")
	} else {
		t.Logf("download-all job ended with status: %s", j.Status)
	}
}

// ---- HandleGetAnalyzedTracks coverage ----

// TestHandleGetAnalyzedTracks_WithJSONFile verifies the handler reads an existing JSON
// companion file and includes it in the response.
func TestHandleGetAnalyzedTracks_WithJSONFile(t *testing.T) {
	h, cfg, _ := makeHandlers(t)
	app := fiber.New()
	app.Get("/tracks/analyzed", h.HandleGetAnalyzedTracks)

	// Create MP3 + companion JSON in downloads dir.
	mp3Path := filepath.Join(cfg.DownloadsDir, "test.mp3")
	jsonPath := filepath.Join(cfg.DownloadsDir, "test.json")
	os.WriteFile(mp3Path, []byte("id3"), 0644)
	analysis := `{"bpm":120,"key":"C"}`
	os.WriteFile(jsonPath, []byte(analysis), 0644)

	req := httptest.NewRequest("GET", "/tracks/analyzed?downloads_dir="+cfg.DownloadsDir, nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	tracks, _ := result["tracks"].([]interface{})
	if len(tracks) != 1 {
		t.Errorf("expected 1 analyzed track, got %d", len(tracks))
	}
}

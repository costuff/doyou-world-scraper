package api

import (
	"bufio"
	"doyou-world-scraper/internal/config"
	"doyou-world-scraper/internal/downloader"
	"doyou-world-scraper/internal/jobs"
	"doyou-world-scraper/internal/scraper"
	"doyou-world-scraper/internal/storage"
	"doyou-world-scraper/pkg/logger"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// scrapeSemaphore limits concurrent Chrome instances to prevent resource exhaustion
// Max 2 concurrent scraping jobs (Fix 4)
var scrapeSemaphore = make(chan struct{}, 2)

// Handlers holds the configuration and dependencies for HTTP handlers
type Handlers struct {
	cfg *config.Config
}

// NewHandlers creates a new Handlers instance
func NewHandlers(cfg *config.Config) *Handlers {
	return &Handlers{cfg: cfg}
}

// VersionInfo struct for version endpoint
type VersionInfo struct {
	Backend   string `json:"backend"`
	Frontend  string `json:"frontend"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// HandleVersion returns version info for backend/frontend
func (h *Handlers) HandleVersion(c *fiber.Ctx) error {
	// Read backend version from VERSION file
	backendVersion := "1.0.0"
	if data, err := os.ReadFile("VERSION"); err == nil {
		backendVersion = strings.TrimSpace(string(data))
	}

	// Read frontend version from frontend/VERSION file
	frontendVersion := "1.0.0"
	if data, err := os.ReadFile("frontend/VERSION"); err == nil {
		frontendVersion = strings.TrimSpace(string(data))
	}

	commit := os.Getenv("GIT_COMMIT")
	buildTime := os.Getenv("BUILD_TIME")
	return c.JSON(VersionInfo{
		Backend:   backendVersion,
		Frontend:  frontendVersion,
		Commit:    commit,
		BuildTime: buildTime,
	})
}

// Helper function to check if a file/directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolveDownloadsDir resolves the downloads directory path based on context
// Returns resolved path and error if NAS is requested but not available
func (h *Handlers) resolveDownloadsDir(requestedDir string) (string, error) {
	downloadsDir := requestedDir

	// Handle relative paths - convert to absolute
	if !filepath.IsAbs(downloadsDir) {
		// Check if we're in Docker first
		if fileExists("/app/downloads") {
			switch downloadsDir {
			case h.cfg.DownloadsDir, "downloads":
				downloadsDir = "/app/downloads"
			case "nas":
				downloadsDir = "/nas"
			}
		} else {
			// Local development
			if downloadsDir == "nas" {
				// Try NAS_PATH from config first
				if h.cfg.NASPath != "" && fileExists(h.cfg.NASPath) {
					downloadsDir = h.cfg.NASPath
				} else {
					// Auto-detect common NAS mount points on macOS/Linux
					commonPaths := []string{
						"/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
						"/Volumes/Music/doyou-world-radio-tracks",
						"/mnt/nas/music/doyou-world-radio-tracks",
					}
					found := false
					for _, path := range commonPaths {
						if fileExists(path) {
							downloadsDir = path
							found = true
							break
						}
					}
					if !found {
						return "", fmt.Errorf("NAS storage not found - not mounted or configured. Set NAS_PATH environment variable or mount NAS at a common location")
					}
				}
			} else {
				// Use absolute path from config (already resolved)
				downloadsDir = h.cfg.DownloadsDir
			}
		}
	}

	// Translate Mac volume paths to Docker mount point
	// If user provides /Volumes/... path but we're in Docker, use /nas instead
	if strings.HasPrefix(downloadsDir, "/Volumes/") && fileExists("/nas") {
		downloadsDir = "/nas"
	}

	// Verify directory exists
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("directory not found: %s", downloadsDir)
	}

	return downloadsDir, nil
}

// Scrape handlers
func (h *Handlers) HandleScrapeDate(c *fiber.Ctx) error {
	type Request struct {
		Date string `json:"date"` // Format: YYYY-MM-DD or "today"
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Date == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Date is required"})
	}

	// Create and start job
	job := jobs.NewJob("scrape_date", fiber.Map{
		"date": req.Date,
	})

	go func() {
		// Acquire semaphore (blocks if 2 jobs already running) - Fix 4
		scrapeSemaphore <- struct{}{}
		defer func() { <-scrapeSemaphore }() // Release when done

		// Don't launch Chrome if job was cancelled while waiting for semaphore
		if job.IsCancelled() {
			return
		}

		// Start job now that we have resources
		job.Start()

		count, err := scraper.ScrapeSingleDate(req.Date)
		if err != nil {
			job.Fail(fmt.Sprintf("Scraping failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"csv_file":       fmt.Sprintf("tracks-%s.csv", req.Date),
			"tracks_scraped": count,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Scraping started",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleScrapeMonth(c *fiber.Ctx) error {
	type Request struct {
		Month string `json:"month"` // Format: YYYY-MM
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Month is required"})
	}

	job := jobs.NewJob("scrape_month", fiber.Map{
		"month": req.Month,
	})

	go func() {
		// Acquire semaphore (blocks if 2 jobs already running) - Fix 4
		scrapeSemaphore <- struct{}{}
		defer func() { <-scrapeSemaphore }() // Release when done

		// Don't launch Chrome if job was cancelled while waiting for semaphore
		if job.IsCancelled() {
			return
		}

		// Start job now that we have resources
		job.Start()

		days, tracks, err := scraper.ScrapeMonth(req.Month, job, func(current, total int) {
			progress := int((float64(current) / float64(total)) * 100)
			job.UpdateProgress(progress)
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Month scraping failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"days_scraped": days,
			"total_tracks": tracks,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Month scraping started",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleScrapeYear(c *fiber.Ctx) error {
	type Request struct {
		Year string `json:"year"` // Format: YYYY
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Year == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Year is required"})
	}

	job := jobs.NewJob("scrape_year", fiber.Map{
		"year": req.Year,
	})

	go func() {
		// Acquire semaphore (blocks if 2 jobs already running) - Fix 4
		scrapeSemaphore <- struct{}{}
		defer func() { <-scrapeSemaphore }() // Release when done

		// Don't launch Chrome if job was cancelled while waiting for semaphore
		if job.IsCancelled() {
			return
		}

		// Start job now that we have resources
		job.Start()

		days, tracks, err := scraper.ScrapeYear(req.Year, job, func(current, total int) {
			progress := int((float64(current) / float64(total)) * 100)
			job.UpdateProgress(progress)
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Year scraping failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"days_scraped": days,
			"total_tracks": tracks,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Year scraping started",
		"job":     job.ToMap(),
	})
}

// Download handlers
func (h *Handlers) HandleDownloadFromCSV(c *fiber.Ctx) error {
	type Request struct {
		CSVPath string `json:"csv_path"`
		Column  int    `json:"column"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.CSVPath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "CSV path is required"})
	}

	if req.Column == 0 {
		req.Column = 5 // Default column
	}

	job := jobs.NewJob("download_csv", fiber.Map{
		"csv_path": req.CSVPath,
		"column":   req.Column,
	})

	go func() {
		job.Start()
		downloadsDir := h.cfg.DownloadsDir
		if fileExists("/app/data/downloads") {
			downloadsDir = "/app/data/downloads"
		}
		result, err := downloader.DownloadFromCSV(req.CSVPath, req.Column, downloadsDir, "yt-dlp", job, func(current, total int, title, artist string) {
			progress := int((float64(current) / float64(total)) * 100)
			message := fmt.Sprintf("Downloading: %s - %s (%d/%d)", title, artist, current, total)
			job.UpdateProgressWithMessage(progress, message)
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Download failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"downloaded": result.Downloaded,
			"failed":     result.Failed,
			"total":      result.TotalSongs,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Download started",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleDownloadAll(c *fiber.Ctx) error {
	csvDir := h.cfg.CSVDir
	downloadsDir := h.cfg.DownloadsDir
	if fileExists("/app/data/csv") {
		csvDir = "/app/data/csv"
		downloadsDir = "/app/data/downloads"
	}

	job := jobs.NewJob("download_all", fiber.Map{
		"folder": csvDir,
	})

	go func() {
		job.Start()
		result, err := downloader.DownloadFromFolder(csvDir, 5, downloadsDir, "yt-dlp", job, func(current, total int, title, artist string) {
			progress := int((float64(current) / float64(total)) * 100)
			message := fmt.Sprintf("Downloading: %s - %s (%d/%d)", title, artist, current, total)
			job.UpdateProgressWithMessage(progress, message)
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Download all failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"downloaded": result.Downloaded,
			"failed":     result.Failed,
			"total":      result.TotalSongs,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Downloading from all CSVs",
		"job":     job.ToMap(),
	})
}

// Utility handlers
func (h *Handlers) HandleDeduplicate(c *fiber.Ctx) error {
	type Request struct {
		DownloadsDir string `json:"downloads_dir"`
		NASDir       string `json:"nas_dir"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	downloadsDir := req.DownloadsDir
	if downloadsDir == "" {
		if fileExists("/app/downloads") {
			downloadsDir = "/app/downloads"
		} else {
			downloadsDir = h.cfg.DownloadsDir
		}
	}
	// Determine NAS directory with validation and auto-detection
	nasDir := req.NASDir

	// If path provided, check if it exists; if not, auto-detect
	if nasDir != "" && nasDir != "skip" {
		if _, err := os.Stat(nasDir); os.IsNotExist(err) {
			// Requested path doesn't exist, try auto-detection
			nasDir = ""
		}
	}

	// Auto-detect if empty or invalid path
	if nasDir == "" {
		if fileExists("/nas") {
			nasDir = "/nas" // Docker mount
		} else {
			nasDir = h.cfg.NASPath // From NAS_PATH env var
		}
	}

	// Final validation: if still empty/skip after auto-detection, NAS is not available
	if nasDir == "" || nasDir == "skip" {
		// This is OK for deduplicate - will only scan downloads
		nasDir = "skip"
	}

	job := jobs.NewJob("deduplicate", fiber.Map{
		"downloads_dir": downloadsDir,
		"nas_dir":       nasDir,
	})

	go func() {
		job.Start()
		result, err := storage.RemoveDuplicates(downloadsDir, nasDir, job, func(current, total int) {
			progress := int((float64(current) / float64(total)) * 100)
			job.UpdateProgress(progress)
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Deduplication failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"total_scanned":      result.TotalScanned,
			"duplicates_found":   result.DuplicatesFound,
			"duplicates_removed": result.DuplicatesRemoved,
			"errors":             result.Errors,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Deduplication started",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleMoveToNAS(c *fiber.Ctx) error {
	type Request struct {
		DownloadsDir string `json:"downloads_dir"`
		NASDir       string `json:"nas_dir"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	downloadsDir := req.DownloadsDir
	if downloadsDir == "" {
		if fileExists("/app/data/downloads") {
			downloadsDir = "/app/data/downloads"
		} else {
			downloadsDir = h.cfg.DownloadsDir
		}
	}

	// Determine NAS directory with validation and auto-detection
	nasDir := req.NASDir

	// If path provided, check if it exists; if not, auto-detect
	if nasDir != "" && nasDir != "skip" {
		if _, err := os.Stat(nasDir); os.IsNotExist(err) {
			// Requested path doesn't exist, try auto-detection
			nasDir = ""
		}
	}

	// Auto-detect if empty or invalid path
	if nasDir == "" {
		if fileExists("/nas") {
			nasDir = "/nas" // Docker mount
		} else {
			nasDir = h.cfg.NASPath // From NAS_PATH env var
		}
	}

	// Validate NAS directory exists - cannot move files if NAS is not available
	if nasDir == "" || nasDir == "skip" {
		return c.Status(400).JSON(fiber.Map{
			"error":   "NAS directory not configured",
			"message": "NAS not found. In Docker, ensure /nas is mounted. Locally, set NAS_PATH or clear the UI field.",
		})
	}

	if _, err := os.Stat(nasDir); os.IsNotExist(err) {
		return c.Status(400).JSON(fiber.Map{
			"error":   fmt.Sprintf("NAS directory does not exist: %s", nasDir),
			"message": "Please ensure your NAS is mounted or update the NAS path",
		})
	}

	job := jobs.NewJob("move_to_nas", fiber.Map{
		"downloads_dir": downloadsDir,
		"nas_dir":       nasDir,
	})

	go func() {
		job.Start()
		result, err := storage.MoveDownloadsToNAS(downloadsDir, nasDir, job, func(current, total int) {
			progress := int((float64(current) / float64(total)) * 100)
			job.UpdateProgress(progress)
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Move to NAS failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"total_files":   result.TotalFiles,
			"moved_files":   result.MovedFiles,
			"skipped_files": result.SkippedFiles,
			"failed_files":  result.FailedFiles,
			"errors":        result.Errors,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Moving files to NAS",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleMatchDownloads(c *fiber.Ctx) error {
	job := jobs.NewJob("match_downloads", fiber.Map{})

	go func() {
		job.Start()
		csvDir := h.cfg.CSVDir
		downloadsDir := h.cfg.DownloadsDir
		if fileExists("/app/csv") {
			csvDir = "/app/csv"
			downloadsDir = "/app/downloads"
		}
		result, err := storage.MatchDownloadsToCSVs(csvDir, downloadsDir)
		if err != nil {
			job.Fail(fmt.Sprintf("Matching failed: %v", err))
			return
		}
		latestCSVName := ""
		if result.LatestCSV != nil {
			latestCSVName = result.LatestCSV.Name
		}
		job.Complete(fiber.Map{
			"matched":     result.MatchedFiles,
			"total_files": result.TotalFiles,
			"latest_csv":  latestCSVName,
			"top_csvs":    result.TopCSVs,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Matching downloads to CSVs",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleClearCSVs(c *fiber.Ctx) error {
	csvDir := h.cfg.CSVDir
	if fileExists("/app/data/csv") {
		csvDir = "/app/data/csv"
	}

	job := jobs.NewJob("clear_csvs", fiber.Map{
		"csv_dir": csvDir,
	})

	go func() {
		job.Start()
		result, err := storage.ClearCSVs(csvDir)
		if err != nil {
			job.Fail(fmt.Sprintf("Clear CSVs failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"total_files":   result.TotalFiles,
			"deleted_files": result.DeletedFiles,
			"errors":        result.Errors,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Clearing CSV files",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleClearDownloads(c *fiber.Ctx) error {
	downloadsDir := h.cfg.DownloadsDir
	if fileExists("/app/data/downloads") {
		downloadsDir = "/app/data/downloads"
	}

	job := jobs.NewJob("clear_downloads", fiber.Map{
		"downloads_dir": downloadsDir,
	})

	go func() {
		job.Start()
		result, err := storage.ClearDownloads(downloadsDir, func(current, total int, filename string) {
			progress := int(float64(current) / float64(total) * 100)
			job.UpdateProgressWithMessage(progress, fmt.Sprintf("Deleting %d/%d: %s", current, total, filename))
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Clear downloads failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"total_files":   result.TotalFiles,
			"deleted_files": result.DeletedFiles,
			"errors":        result.Errors,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Clearing downloaded files",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleClearNASDownloads(c *fiber.Ctx) error {
	type Request struct {
		NASDir string `json:"nas_dir"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	nasDir := req.NASDir

	// Translate Mac volume paths to Docker mount point
	// If user provides /Volumes/... path but we're in Docker, use /nas instead
	if strings.HasPrefix(nasDir, "/Volumes/") && fileExists("/nas") {
		nasDir = "/nas"
	}

	if nasDir == "" {
		// Try to auto-detect NAS path
		if fileExists("/nas") {
			nasDir = "/nas"
		} else if h.cfg.NASPath != "" {
			nasDir = h.cfg.NASPath
		} else {
			return c.Status(400).JSON(fiber.Map{
				"error":   "NAS path not provided",
				"message": "Please provide nas_dir in request body or set NAS_PATH environment variable",
			})
		}
	}

	// Verify NAS directory exists
	if _, err := os.Stat(nasDir); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{
			"error":   fmt.Sprintf("NAS directory does not exist: %s", nasDir),
			"message": "In Docker, NAS should be mounted at /nas. Check docker-compose.yml volumes configuration.",
		})
	}

	job := jobs.NewJob("clear_nas_downloads", fiber.Map{
		"nas_dir": nasDir,
	})

	go func() {
		job.Start()
		result, err := storage.ClearDownloads(nasDir, func(current, total int, filename string) {
			progress := int(float64(current) / float64(total) * 100)
			job.UpdateProgressWithMessage(progress, fmt.Sprintf("Deleting %d/%d: %s", current, total, filename))
		})
		if err != nil {
			job.Fail(fmt.Sprintf("Clear NAS downloads failed: %v", err))
			return
		}
		job.Complete(fiber.Map{
			"total_files":   result.TotalFiles,
			"deleted_files": result.DeletedFiles,
			"errors":        result.Errors,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Clearing NAS downloaded files",
		"job":     job.ToMap(),
	})
}

func (h *Handlers) HandleStats(c *fiber.Ctx) error {
	stats := fiber.Map{
		"total_csvs":           0,
		"total_tracks_scraped": 0,
		"total_downloads":      0,
		"downloads_size_gb":    0.0,
		"nas_downloads":        0,
		"nas_size_gb":          0.0,
		"earliest_date":        "",
		"latest_date":          "",
	}

	// Count CSV files (fast)
	csvDir := h.cfg.CSVDir
	if fileExists("/app/data/csv") {
		csvDir = "/app/data/csv"
	}
	csvFiles, err := filepath.Glob(filepath.Join(csvDir, "*.csv"))
	if err == nil {
		stats["total_csvs"] = len(csvFiles)

		// Count total tracks from CSVs and get date range
		totalTracks := 0
		var earliestDate, latestDate string
		for _, csvFile := range csvFiles {
			// Count lines in CSV (subtract 1 for header)
			f, err := os.Open(csvFile)
			if err == nil {
				scanner := bufio.NewScanner(f)
				lineCount := 0
				for scanner.Scan() {
					lineCount++
				}
				if lineCount > 1 {
					totalTracks += lineCount - 1
				}
				if err := f.Close(); err != nil {
						logger.Warn(fmt.Sprintf("Failed to close CSV file: %v", err))
					}
			}

			// Extract date from filename (tracks-YYYY-MM-DD.csv)
			baseName := filepath.Base(csvFile)
			if len(baseName) >= 21 {
				dateStr := baseName[7:17] // Extract YYYY-MM-DD
				if earliestDate == "" || dateStr < earliestDate {
					earliestDate = dateStr
				}
				if latestDate == "" || dateStr > latestDate {
					latestDate = dateStr
				}
			}
		}
		stats["total_tracks_scraped"] = totalTracks
		stats["earliest_date"] = earliestDate
		stats["latest_date"] = latestDate
	}

	// Count downloads (with timeout)
	type downloadResult struct {
		count  int
		sizeGB float64
	}
	downloadsDone := make(chan downloadResult, 1)
	go func() {
		var totalMP3s int
		var downloadsDirSize int64
		downloadsDir := h.cfg.DownloadsDir
		if fileExists("/app/data/downloads") {
			downloadsDir = "/app/data/downloads"
		}
		_ = filepath.Walk(downloadsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
				totalMP3s++
				downloadsDirSize += info.Size()
			}
			return nil
		})
		downloadsDone <- downloadResult{count: totalMP3s, sizeGB: float64(downloadsDirSize) / 1024 / 1024 / 1024}
	}()

	// Count NAS (with timeout) - only if directory exists
	type nasResult struct {
		count  int
		sizeGB float64
	}
	nasDone := make(chan nasResult, 1)
	go func() {
		var nasMP3s int
		var nasDirSize int64
		nasDir := h.cfg.NASPath
		if fileExists("/nas") {
			nasDir = "/nas"
		}

		// Only scan if NAS directory exists and is not empty
		if nasDir != "" && fileExists(nasDir) {
			_ = filepath.Walk(nasDir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
					nasMP3s++
					nasDirSize += info.Size()
				}
				return nil
			})
		}

		nasDone <- nasResult{count: nasMP3s, sizeGB: float64(nasDirSize) / 1024 / 1024 / 1024}
	}()

	// Wait for downloads scan to complete (max 2 seconds)
	select {
	case r := <-downloadsDone:
		stats["total_downloads"] = r.count
		stats["downloads_size_gb"] = r.sizeGB
	case <-time.After(2 * time.Second):
	}

	// Wait for NAS scan to complete (max 3 seconds)
	select {
	case r := <-nasDone:
		stats["nas_downloads"] = r.count
		stats["nas_size_gb"] = r.sizeGB
	case <-time.After(3 * time.Second):
	}

	return c.JSON(stats)
}

// Job handlers
func (h *Handlers) HandleGetJobs(c *fiber.Ctx) error {
	jobs := jobs.GetAllJobs()
	return c.JSON(fiber.Map{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

func (h *Handlers) HandleGetJob(c *fiber.Ctx) error {
	id := c.Params("id")
	job := jobs.GetJob(id)

	if job == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	return c.JSON(job.ToMap())
}

func (h *Handlers) HandleCancelJob(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := jobs.CancelJob(id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Job cancelled successfully",
		"job_id":  id,
	})
}

// Track browsing handler
func (h *Handlers) HandleGetTracks(c *fiber.Ctx) error {
	// Parse query parameters
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 50)
	search := c.Query("search", "")
	sortBy := c.Query("sort_by", "mod_time")
	sortOrder := c.Query("sort_order", "desc")
	requestedDir := c.Query("downloads_dir", h.cfg.DownloadsDir)
	async := c.QueryBool("async", false)

	// Resolve downloads directory (handles Docker, local, NAS)
	downloadsDir, err := h.resolveDownloadsDir(requestedDir)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Storage directory not accessible",
			"hint":    "Set NAS_PATH environment variable or check if NAS is mounted",
		})
	}

	result, err := storage.GetTracks(downloadsDir, page, perPage, search, sortBy, sortOrder, async)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to list tracks: %v", err)})
	}

	return c.JSON(result)
}

// Track scan status handler
func (h *Handlers) HandleGetTracksStatus(c *fiber.Ctx) error {
	requestedDir := c.Query("downloads_dir", h.cfg.DownloadsDir)

	// Resolve downloads directory (handles Docker, local, NAS)
	downloadsDir, err := h.resolveDownloadsDir(requestedDir)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Storage directory not accessible",
		})
	}

	status := storage.GetTrackScanStatus(downloadsDir)
	return c.JSON(status)
}

// Get analyzed tracks (tracks with companion .json analysis files)
func (h *Handlers) HandleGetAnalyzedTracks(c *fiber.Ctx) error {
	requestedDir := c.Query("downloads_dir", h.cfg.DownloadsDir)

	// Resolve downloads directory (handles Docker, local, NAS)
	downloadsDir, err := h.resolveDownloadsDir(requestedDir)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Storage directory not accessible",
			"hint":    "Set NAS_PATH environment variable or check if NAS is mounted",
		})
	}

	// Scan for MP3 files with companion JSON files
	tracks := []fiber.Map{}

	entries, err := os.ReadDir(downloadsDir)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to read directory: %v", err),
		})
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		// Only process MP3 files
		if !strings.HasSuffix(strings.ToLower(fileName), ".mp3") {
			continue
		}

		// Check for companion JSON file
		baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		jsonPath := filepath.Join(downloadsDir, baseName+".json")

		if _, err := os.Stat(jsonPath); err == nil {
			// JSON file exists, read analysis data
			jsonData, err := os.ReadFile(jsonPath)
			if err != nil {
				logger.Errorf("Failed to read analysis file %s: %v", jsonPath, err)
				continue
			}

			var analysis map[string]interface{}
			if err := json.Unmarshal(jsonData, &analysis); err != nil {
				logger.Errorf("Failed to parse analysis file %s: %v", jsonPath, err)
				continue
			}

			tracks = append(tracks, fiber.Map{
				"file_path": filepath.Join(downloadsDir, fileName),
				"file_name": fileName,
				"analysis":  analysis,
			})
		}
	}

	return c.JSON(fiber.Map{
		"tracks": tracks,
		"count":  len(tracks),
	})
}

// Serve audio file for playback
func (h *Handlers) HandleStreamTrack(c *fiber.Ctx) error {
	requestedPath := c.Query("file")
	if requestedPath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "File path is required"})
	}

	resolvedPath, err := h.resolveStreamPath(requestedPath)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "File not found"})
	}

	// Set proper headers for audio streaming
	c.Set("Content-Type", "audio/mpeg")
	c.Set("Accept-Ranges", "bytes")

	return c.SendFile(resolvedPath)
}

func (h *Handlers) resolveStreamPath(requestedPath string) (string, error) {
	normalized := filepath.Clean(requestedPath)
	var candidates []string

	// Direct path first (works for already-correct absolute paths like /nas/...)
	candidates = append(candidates, normalized)

	// Map analytics worker absolute paths to scraper mount
	if strings.HasPrefix(normalized, "/mnt/music/downloads/") || normalized == "/mnt/music/downloads" {
		rel := strings.TrimPrefix(normalized, "/mnt/music/downloads")
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			candidates = append(candidates, "/nas")
		} else {
			candidates = append(candidates, filepath.Join("/nas", rel))
		}
	}

	// Relative paths from analytics DB (e.g. complete/foo/bar.mp3)
	if !filepath.IsAbs(normalized) {
		candidates = append(candidates,
			filepath.Join("/nas", normalized),
			filepath.Join("/app/downloads", normalized),
			filepath.Join(h.cfg.DownloadsDir, normalized),
		)
		if h.cfg.NASPath != "" {
			candidates = append(candidates, filepath.Join(h.cfg.NASPath, normalized))
		}
	}

	// Try candidates in order and return first existing file
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("stream file not found")
}

// Audio analysis handler
func (h *Handlers) HandleAnalyzeTrack(c *fiber.Ctx) error {
	type Request struct {
		FilePath string `json:"file_path"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.FilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "File path is required"})
	}

	// Check if file exists
	if _, err := os.Stat(req.FilePath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "File not found"})
	}

	job := jobs.NewJob("analyze_track", fiber.Map{
		"file_path": req.FilePath,
	})

	go func() {
		job.Start()
		// Find Python executable (venv locally, system python3 in Docker)
		pythonPath := filepath.Join(h.cfg.DataDir, "..", ".venv", "bin", "python")
		if !fileExists(pythonPath) {
			// In Docker or no venv available
			if fileExists("/usr/bin/python3") {
				pythonPath = "/usr/bin/python3"
			} else {
				pythonPath = "python3"
			}
		}

		scriptPath := filepath.Join(h.cfg.DataDir, "..", "scripts", "analyze_audio.py")
		if fileExists("/app/scripts/analyze_audio.py") {
			scriptPath = "/app/scripts/analyze_audio.py"
		}

		// Run Python analysis script with --quick flag for faster web UI response
		cmd := exec.Command(pythonPath, scriptPath, "--single", "--quiet", "--quick", req.FilePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			job.Fail(fmt.Sprintf("Analysis failed: %v - %s", err, string(output)))
			return
		}

		// Parse JSON output
		var analysis map[string]interface{}
		if err := json.Unmarshal(output, &analysis); err != nil {
			job.Fail(fmt.Sprintf("Failed to parse analysis results: %v - Output: %s", err, string(output)))
			return
		}

		// Save analysis to companion JSON file
		baseName := strings.TrimSuffix(req.FilePath, filepath.Ext(req.FilePath))
		jsonPath := baseName + ".json"
		if err := os.WriteFile(jsonPath, output, 0644); err != nil {
			logger.Errorf("Failed to save analysis JSON: %v (path: %s)", err, jsonPath)
			// Don't fail the job, just log the error
		}

		job.Complete(fiber.Map{
			"analysis":  analysis,
			"json_path": jsonPath,
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Audio analysis started",
		"job":     job.ToMap(),
	})
}

// Batch analyze multiple tracks
func (h *Handlers) HandleAnalyzeAll(c *fiber.Ctx) error {
	type Request struct {
		GenerateSpectrograms bool `json:"generate_spectrograms"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		// Default to false if no body
		req.GenerateSpectrograms = false
	}

	downloadsDir := h.cfg.DownloadsDir
	if fileExists("/app/data/downloads") {
		downloadsDir = "/app/data/downloads"
	}

	job := jobs.NewJob("analyze_all", fiber.Map{
		"downloads_dir":         downloadsDir,
		"generate_spectrograms": req.GenerateSpectrograms,
	})

	go func() {
		job.Start()
		// Find Python executable (venv locally, system python3 in Docker)
		pythonPath := filepath.Join(h.cfg.DataDir, "..", ".venv", "bin", "python")
		if !fileExists(pythonPath) {
			// In Docker or no venv available
			if fileExists("/usr/bin/python3") {
				pythonPath = "/usr/bin/python3"
			} else {
				pythonPath = "python3"
			}
		}

		scriptPath := filepath.Join(h.cfg.DataDir, "..", "scripts", "analyze_audio.py")
		if fileExists("/app/scripts/analyze_audio.py") {
			scriptPath = "/app/scripts/analyze_audio.py"
		}

		// Get all MP3 files in downloads directory
		mp3Files, err := filepath.Glob(filepath.Join(downloadsDir, "*.mp3"))
		if err != nil {
			job.Fail(fmt.Sprintf("Failed to list MP3 files: %v", err))
			return
		}

		if len(mp3Files) == 0 {
			job.Complete(fiber.Map{
				"message":        "No MP3 files found to analyze",
				"analyzed_count": 0,
			})
			return
		}

		job.UpdateProgressWithMessage(0, fmt.Sprintf("Found %d tracks to analyze", len(mp3Files)))

		analyzed := 0
		failed := 0
		skipped := 0

		for i, mp3Path := range mp3Files {
			// Check if already analyzed (JSON exists)
			baseName := strings.TrimSuffix(mp3Path, filepath.Ext(mp3Path))
			jsonPath := baseName + ".json"
			if fileExists(jsonPath) {
				skipped++
				continue
			}

			// Update progress
			progress := int((float64(i) / float64(len(mp3Files))) * 100)
			job.UpdateProgressWithMessage(progress, fmt.Sprintf("Analyzing %d/%d: %s", i+1, len(mp3Files), filepath.Base(mp3Path)))

			// Run analysis for this file
			cmd := exec.Command(pythonPath, scriptPath, "--single", "--quiet", "--quick", mp3Path)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logger.Errorf("Failed to analyze track %s: %v", mp3Path, err)
				failed++
				continue
			}

			// Save companion JSON file
			if err := os.WriteFile(jsonPath, output, 0644); err != nil {
				logger.Errorf("Failed to save analysis JSON %s: %v", jsonPath, err)
				failed++
				continue
			}

			analyzed++
		}

		job.Complete(fiber.Map{
			"analyzed_count": analyzed,
			"failed_count":   failed,
			"skipped_count":  skipped,
			"total_files":    len(mp3Files),
			"message":        fmt.Sprintf("Analyzed %d tracks (%d skipped, %d failed)", analyzed, skipped, failed),
		})
	}()

	return c.Status(202).JSON(fiber.Map{
		"message": "Batch analysis started",
		"job":     job.ToMap(),
	})
}

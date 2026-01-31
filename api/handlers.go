package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Helper function to check if a file/directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Scrape handlers
func handleScrapeDate(c *fiber.Ctx) error {
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
	job := NewJob("scrape_date", fiber.Map{
		"date": req.Date,
	})

	go func() {
		count, err := ScrapeSingleDate(req.Date)
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

func handleScrapeMonth(c *fiber.Ctx) error {
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

	job := NewJob("scrape_month", fiber.Map{
		"month": req.Month,
	})

	go func() {
		days, tracks, err := ScrapeMonth(req.Month, job, func(current, total int) {
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

func handleScrapeYear(c *fiber.Ctx) error {
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

	job := NewJob("scrape_year", fiber.Map{
		"year": req.Year,
	})

	go func() {
		days, tracks, err := ScrapeYear(req.Year, job, func(current, total int) {
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
func handleDownloadFromCSV(c *fiber.Ctx) error {
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

	job := NewJob("download_csv", fiber.Map{
		"csv_path": req.CSVPath,
		"column":   req.Column,
	})

	go func() {
		downloadsDir := "./downloads"
		if filepath.IsAbs("downloads") || fileExists("/app/downloads") {
			downloadsDir = "/app/downloads"
		}
		result, err := DownloadFromCSV(req.CSVPath, req.Column, downloadsDir, "yt-dlp", job, func(current, total int) {
			progress := int((float64(current) / float64(total)) * 100)
			job.UpdateProgress(progress)
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

func handleDownloadAll(c *fiber.Ctx) error {
	csvDir := "./csv"
	downloadsDir := "./downloads"
	if fileExists("/app/csv") {
		csvDir = "/app/csv"
		downloadsDir = "/app/downloads"
	}

	job := NewJob("download_all", fiber.Map{
		"folder": csvDir,
	})

	go func() {
		result, err := DownloadFromFolder(csvDir, 5, downloadsDir, "yt-dlp", job, func(current, total int) {
			progress := int((float64(current) / float64(total)) * 100)
			job.UpdateProgress(progress)
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
func handleDeduplicate(c *fiber.Ctx) error {
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
			downloadsDir = "./downloads"
		}
	}
	nasDir := req.NASDir
	if nasDir == "" {
		if fileExists("/nas") {
			nasDir = "/nas"
		} else {
			nasDir = "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"
		}
	}

	job := NewJob("deduplicate", fiber.Map{
		"downloads_dir": downloadsDir,
		"nas_dir":       nasDir,
	})

	go func() {
		result, err := RemoveDuplicates(downloadsDir, nasDir, job, func(current, total int) {
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

func handleMoveToNAS(c *fiber.Ctx) error {
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
			downloadsDir = "./downloads"
		}
	}

	nasDir := req.NASDir
	if nasDir == "" {
		if fileExists("/nas") {
			nasDir = "/nas"
		} else {
			nasDir = "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"
		}
	}

	job := NewJob("move_to_nas", fiber.Map{
		"downloads_dir": downloadsDir,
		"nas_dir":       nasDir,
	})

	go func() {
		result, err := MoveDownloadsToNAS(downloadsDir, nasDir, job, func(current, total int) {
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

func handleMatchDownloads(c *fiber.Ctx) error {
	job := NewJob("match_downloads", fiber.Map{})

	go func() {
		csvDir := "./csv"
		downloadsDir := "./downloads"
		if fileExists("/app/csv") {
			csvDir = "/app/csv"
			downloadsDir = "/app/downloads"
		}
		result, err := MatchDownloadsToCSVs(csvDir, downloadsDir)
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

func handleStats(c *fiber.Ctx) error {
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
	csvDir := "./csv"
	if fileExists("/app/csv") {
		csvDir = "/app/csv"
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
				f.Close()
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
	done := make(chan bool, 1)
	go func() {
		var totalMP3s int
		var downloadsDirSize int64
		downloadsDir := "./downloads"
		if fileExists("/app/downloads") {
			downloadsDir = "/app/downloads"
		}
		filepath.Walk(downloadsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
				totalMP3s++
				downloadsDirSize += info.Size()
			}
			return nil
		})
		stats["total_downloads"] = totalMP3s
		stats["downloads_size_gb"] = float64(downloadsDirSize) / 1024 / 1024 / 1024
		done <- true
	}()

	// Count NAS (with timeout)
	go func() {
		var nasMP3s int
		var nasDirSize int64
		nasDir := "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"
		if fileExists("/nas") {
			nasDir = "/nas"
		}
		filepath.Walk(nasDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
				nasMP3s++
				nasDirSize += info.Size()
			}
			return nil
		})
		stats["nas_downloads"] = nasMP3s
		stats["nas_size_gb"] = float64(nasDirSize) / 1024 / 1024 / 1024
		done <- true
	}()

	// Wait for downloads scan to complete (max 2 seconds)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	// Wait for NAS scan to complete (max 3 seconds)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}

	return c.JSON(stats)
}

// Job handlers
func handleGetJobs(c *fiber.Ctx) error {
	jobs := GetAllJobs()
	return c.JSON(fiber.Map{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

func handleGetJob(c *fiber.Ctx) error {
	id := c.Params("id")
	job := GetJob(id)

	if job == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	return c.JSON(job.ToMap())
}

func handleCancelJob(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := CancelJob(id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Job cancelled successfully",
		"job_id":  id,
	})
}

// Track browsing handler
func handleGetTracks(c *fiber.Ctx) error {
	// Parse query parameters
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 50)
	search := c.Query("search", "")
	sortBy := c.Query("sort_by", "mod_time")
	sortOrder := c.Query("sort_order", "desc")
	downloadsDir := c.Query("downloads_dir", "downloads")
	async := c.QueryBool("async", false)

	// Handle relative paths - convert to absolute
	if !filepath.IsAbs(downloadsDir) {
		// Check if we're in Docker first
		if fileExists("/app/downloads") {
			if downloadsDir == "downloads" {
				downloadsDir = "/app/downloads"
			} else if downloadsDir == "nas" {
				downloadsDir = "/nas"
			}
		} else {
			// Local development - use relative path from project root
			cwd, err := os.Getwd()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to get working directory: %v", err)})
			}
			downloadsDir = filepath.Join(filepath.Dir(cwd), downloadsDir)
		}
	}

	// Check if directory exists
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": fmt.Sprintf("Directory not found: %s", downloadsDir)})
	}

	result, err := GetTracks(downloadsDir, page, perPage, search, sortBy, sortOrder, async)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to list tracks: %v", err)})
	}

	return c.JSON(result)
}

// Track scan status handler
func handleGetTracksStatus(c *fiber.Ctx) error {
	downloadsDir := c.Query("downloads_dir", "downloads")

	// Handle relative paths - convert to absolute
	if !filepath.IsAbs(downloadsDir) {
		// Check if we're in Docker first
		if fileExists("/app/downloads") {
			if downloadsDir == "downloads" {
				downloadsDir = "/app/downloads"
			} else if downloadsDir == "nas" {
				downloadsDir = "/nas"
			}
		} else {
			// Local development - use relative path from project root
			cwd, err := os.Getwd()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to get working directory: %v", err)})
			}
			downloadsDir = filepath.Join(filepath.Dir(cwd), downloadsDir)
		}
	}

	// Check if directory exists
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": fmt.Sprintf("Directory not found: %s", downloadsDir)})
	}

	status := GetTrackScanStatus(downloadsDir)
	return c.JSON(status)
}

// Serve audio file for playback
func handleStreamTrack(c *fiber.Ctx) error {
	filepath := c.Query("file")
	if filepath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "File path is required"})
	}

	// Security: Check if file exists and is within downloads directory
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"error": "File not found"})
	}

	// Set proper headers for audio streaming
	c.Set("Content-Type", "audio/mpeg")
	c.Set("Accept-Ranges", "bytes")

	return c.SendFile(filepath)
}

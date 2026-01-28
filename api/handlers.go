package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

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
		days, tracks, err := ScrapeMonth(req.Month, func(current, total int) {
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
		days, tracks, err := ScrapeYear(req.Year, func(current, total int) {
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
		result, err := DownloadFromCSV(req.CSVPath, req.Column, "downloads", "yt-dlp", func(current, total int) {
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
	job := NewJob("download_all", fiber.Map{
		"folder": "csv/",
	})

	go func() {
		result, err := DownloadFromFolder("csv", 5, "downloads", "yt-dlp", func(current, total int) {
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
		downloadsDir = "downloads"
	}
	nasDir := req.NASDir
	if nasDir == "" {
		nasDir = "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"
	}

	job := NewJob("deduplicate", fiber.Map{
		"downloads_dir": downloadsDir,
		"nas_dir":       nasDir,
	})

	go func() {
		result, err := RemoveDuplicates(downloadsDir, nasDir)
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

func handleMatchDownloads(c *fiber.Ctx) error {
	job := NewJob("match_downloads", fiber.Map{})

	go func() {
		result, err := MatchDownloadsToCSVs("csv", "downloads")
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
	// Count CSV files
	csvFiles, _ := filepath.Glob("csv/*.csv")
	totalCSVs := len(csvFiles)

	// Count downloads
	var totalMP3s int
	filepath.Walk("downloads", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			totalMP3s++
		}
		return nil
	})

	// Get directory sizes
	var downloadsDirSize, nasDirSize int64
	filepath.Walk("downloads", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			downloadsDirSize += info.Size()
		}
		return nil
	})
	filepath.Walk("/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			nasDirSize += info.Size()
		}
		return nil
	})

	return c.JSON(fiber.Map{
		"total_csv_files":       totalCSVs,
		"total_downloads":       totalMP3s,
		"downloads_dir_size_mb": downloadsDirSize / 1024 / 1024,
		"nas_dir_size_mb":       nasDirSize / 1024 / 1024,
	})
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

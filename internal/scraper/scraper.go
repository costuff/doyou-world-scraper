package scraper

import (
	"context"
	"doyou-world-scraper/internal/jobs"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

type Track struct {
	Title, Artist, Album, Label, ReleaseDate, Listen string
}

// createChromeAllocator creates a Chrome allocator with memory-optimized flags
func createChromeAllocator() (context.Context, context.CancelFunc) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			// Memory and resource limits (Fix 3)
			chromedp.Flag("disable-dev-tools", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-plugins", true),
			chromedp.Flag("disable-images", true), // Don't load images (faster + less memory)
			chromedp.Flag("disk-cache-size", "1"), // Minimal cache
			chromedp.Flag("media-cache-size", "1"),
			chromedp.Flag("aggressive-cache-discard", true),
			chromedp.Flag("disable-cache", true),
			chromedp.Flag("disable-application-cache", true),
			chromedp.Flag("disable-offline-load-stale-cache", true),
		)...)
	return allocCtx, allocCancel
}

// scrapeSingleDateWithContext scrapes tracks for a single date using an existing context
func scrapeSingleDateWithContext(ctx context.Context, dateStr string) (int, error) {
	// Validate date is not in the future
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid date format (use YYYY-MM-DD): %v", err)
	}
	if date.After(time.Now()) {
		return 0, fmt.Errorf("cannot scrape future date: %s", dateStr)
	}

	url := fmt.Sprintf("https://doyoutrackid.com/tracks/%s", dateStr)
	fmt.Println("Loading:", url)

	// Add timeout for page operations (Fix 2)
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()

	var trackNodes []*cdp.Node
	err = chromedp.Run(timeoutCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
		chromedp.Nodes(`div.Track_root__fkoK4`, &trackNodes, chromedp.ByQueryAll),
	)
	if err != nil {
		if err == context.DeadlineExceeded {
			return 0, fmt.Errorf("page load timeout after 30s for %s", dateStr)
		}
		return 0, fmt.Errorf("error loading page or finding tracks: %v", err)
	}

	var tracks []Track

	for _, node := range trackNodes {
		var html string
		err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			h, err := dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			if err != nil {
				return err
			}
			html = h
			return nil
		}))
		if err != nil {
			fmt.Println("Error extracting track HTML:", err)
			continue
		}
		title := extractBetween(html, `<h1 class="Track_title__rH4bg">`, `</h1>`)
		artist := extractBetween(html, `<h2 class="Track_artist__MQzWB">`, `</h2>`)
		album := extractBetween(html, `<p class="Track_album__xqOrO">Album: <span class="Track_value__KiAxm">`, `</span>`)
		label := extractBetween(html, `<p class="Track_label__HS9Oh">Label: <span class="Track_value__KiAxm">`, `</span>`)
		releaseDate := extractBetween(html, `<p class="Track_releaseDate__rpmm5">Release date: <span class="Track_value__KiAxm">`, `</span>`)
		listen := extractHref(html, `<a href="`, `" target=`)
		tracks = append(tracks, Track{title, artist, album, label, releaseDate, listen})
	}

	// Write to CSV in data/csv/tracks-[date].csv
	csvDir := "./data/csv"
	if fileExists("/app/data/csv") {
		csvDir = "/app/data/csv"
	}
	if err := os.MkdirAll(csvDir, 0755); err != nil {
		return 0, fmt.Errorf("error creating csv directory: %v", err)
	}
	outFile := filepath.Join(csvDir, fmt.Sprintf("tracks-%s.csv", dateStr))
	f, err := os.Create(outFile)
	if err != nil {
		return 0, fmt.Errorf("error creating CSV file: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	// Write header
	_ = w.Write([]string{"Title", "Artist", "Album", "Label", "ReleaseDate", "Listen"})
	for _, t := range tracks {
		_ = w.Write([]string{t.Title, t.Artist, t.Album, t.Label, t.ReleaseDate, t.Listen})
	}
	fmt.Printf("Saved %d tracks to %s\n", len(tracks), outFile)
	return len(tracks), nil
}

// ScrapeSingleDate scrapes tracks for a single date and returns the count
// This is the public API that creates and cleans up its own Chrome context
func ScrapeSingleDate(dateStr string) (int, error) {
	// Create Chrome allocator with memory limits (Fix 3)
	allocCtx, allocCancel := createChromeAllocator()
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	return scrapeSingleDateWithContext(ctx, dateStr)
}

// ScrapeMonth scrapes all days in a given month (YYYY-MM format)
func ScrapeMonth(monthStr string, job *jobs.Job, progressCallback func(current, total int)) (int, int, error) {
	t, err := time.Parse("2006-01", monthStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month format (use YYYY-MM): %v", err)
	}
	year, mon := t.Year(), t.Month()
	today := time.Now()

	// Count total days first (cap at today)
	totalDaysInMonth := 0
	for day := 1; day <= 31; day++ {
		d := time.Date(year, mon, day, 0, 0, 0, 0, time.UTC)
		if d.Month() != mon || d.After(today) {
			break
		}
		totalDaysInMonth++
	}

	// Create allocator once, but create new context per page (Fix 1)
	allocCtx, allocCancel := createChromeAllocator()
	defer allocCancel()

	totalDays := 0
	totalTracks := 0

	for day := 1; day <= 31; day++ {
		// Check if job is cancelled BEFORE starting each page (Fix 5)
		if job != nil && job.IsCancelled() {
			return totalDays, totalTracks, fmt.Errorf("job cancelled")
		}

		d := time.Date(year, mon, day, 0, 0, 0, 0, time.UTC)
		if d.Month() != mon || d.After(today) {
			break
		}
		dateStr := d.Format("2006-01-02")

		// Create new context for each date (Fix 1 - prevents memory leaks)
		ctx, cancel := chromedp.NewContext(allocCtx)

		// Use select pattern to support cancellation during scraping (Fix 5)
		done := make(chan struct {
			count int
			err   error
		}, 1)

		go func() {
			count, err := scrapeSingleDateWithContext(ctx, dateStr)
			done <- struct {
				count int
				err   error
			}{count, err}
		}()

		// Wait for scraping to complete or job to be cancelled
		var count int
		var err error
		select {
		case result := <-done:
			count = result.count
			err = result.err
		case <-time.After(35 * time.Second): // Slightly longer than individual page timeout
			cancel() // Force cleanup on timeout
			err = fmt.Errorf("scraping timeout after 35s")
		}

		cancel() // Always cleanup context

		// Check again if cancelled during the operation (Fix 5)
		if job != nil && job.IsCancelled() {
			return totalDays, totalTracks, fmt.Errorf("job cancelled during scraping")
		}

		if err != nil {
			fmt.Printf("Error scraping %s: %v\n", dateStr, err)
			continue
		}
		totalDays++
		totalTracks += count

		// Report progress
		if progressCallback != nil {
			progressCallback(totalDays, totalDaysInMonth)
		}
	}

	return totalDays, totalTracks, nil
}

// ScrapeYear scrapes all days in a given year (YYYY format)
func ScrapeYear(yearStr string, job *jobs.Job, progressCallback func(current, total int)) (int, int, error) {
	y, err := strconv.Atoi(yearStr)
	if err != nil || y < 1 {
		return 0, 0, fmt.Errorf("invalid year format (use YYYY): %v", err)
	}

	today := time.Now()

	// Count total days in year first (cap at today)
	totalDaysInYear := 0
	for m := 1; m <= 12; m++ {
		t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		mon := t.Month()
		for day := 1; day <= 31; day++ {
			d := time.Date(y, mon, day, 0, 0, 0, 0, time.UTC)
			if d.Month() != mon || d.After(today) {
				break
			}
			totalDaysInYear++
		}
	}

	// Create allocator once, but create new context per page (Fix 1)
	allocCtx, allocCancel := createChromeAllocator()
	defer allocCancel()

	totalDays := 0
	totalTracks := 0

	for m := 1; m <= 12; m++ {
		// Check if job is cancelled BEFORE starting each month (Fix 5)
		if job != nil && job.IsCancelled() {
			return totalDays, totalTracks, fmt.Errorf("job cancelled")
		}

		t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		mon := t.Month()
		for day := 1; day <= 31; day++ {
			// Check if job is cancelled BEFORE starting each page (Fix 5)
			if job != nil && job.IsCancelled() {
				return totalDays, totalTracks, fmt.Errorf("job cancelled")
			}

			d := time.Date(y, mon, day, 0, 0, 0, 0, time.UTC)
			if d.Month() != mon || d.After(today) {
				break
			}
			dateStr := d.Format("2006-01-02")

			// Create new context for each date (Fix 1 - prevents memory leaks)
			ctx, cancel := chromedp.NewContext(allocCtx)

			// Use select pattern to support cancellation during scraping (Fix 5)
			done := make(chan struct {
				count int
				err   error
			}, 1)

			go func() {
				count, err := scrapeSingleDateWithContext(ctx, dateStr)
				done <- struct {
					count int
					err   error
				}{count, err}
			}()

			// Wait for scraping to complete or job to be cancelled
			var count int
			var err error
			select {
			case result := <-done:
				count = result.count
				err = result.err
			case <-time.After(35 * time.Second): // Slightly longer than individual page timeout
				cancel() // Force cleanup on timeout
				err = fmt.Errorf("scraping timeout after 35s")
			}

			cancel() // Always cleanup context

			// Check again if cancelled during the operation (Fix 5)
			if job != nil && job.IsCancelled() {
				return totalDays, totalTracks, fmt.Errorf("job cancelled during scraping")
			}

			if err != nil {
				fmt.Printf("Error scraping %s: %v\n", dateStr, err)
				continue
			}
			totalDays++
			totalTracks += count

			// Report progress
			if progressCallback != nil {
				progressCallback(totalDays, totalDaysInYear)
			}
		}
	}

	return totalDays, totalTracks, nil
}

// Helper functions
func extractBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i == -1 {
		return ""
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j == -1 {
		return ""
	}
	return strings.TrimSpace(s[i : i+j])
}

func extractHref(s, start, end string) string {
	i := strings.Index(s, start)
	if i == -1 {
		return ""
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j == -1 {
		return ""
	}
	return s[i : i+j]
}

// fileExists checks if a file/directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

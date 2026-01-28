package main

import (
	"context"
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

// ScrapeSingleDate scrapes tracks for a single date and returns the count
func ScrapeSingleDate(dateStr string) (int, error) {
	url := fmt.Sprintf("https://doyoutrackid.com/tracks/%s", dateStr)
	fmt.Println("Loading:", url)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var trackNodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
		chromedp.Nodes(`div.Track_root__fkoK4`, &trackNodes, chromedp.ByQueryAll),
	)
	if err != nil {
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

	// Write to CSV in csv/tracks-[date].csv
	csvDir := "csv"
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
	w.Write([]string{"Title", "Artist", "Album", "Label", "ReleaseDate", "Listen"})
	for _, t := range tracks {
		w.Write([]string{t.Title, t.Artist, t.Album, t.Label, t.ReleaseDate, t.Listen})
	}
	fmt.Printf("Saved %d tracks to %s\n", len(tracks), outFile)
	return len(tracks), nil
}

// ScrapeMonth scrapes all days in a given month (YYYY-MM format)
func ScrapeMonth(monthStr string, progressCallback func(current, total int)) (int, int, error) {
	t, err := time.Parse("2006-01", monthStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month format (use YYYY-MM): %v", err)
	}
	year, mon := t.Year(), t.Month()

	// Count total days first
	totalDaysInMonth := 0
	for day := 1; day <= 31; day++ {
		d := time.Date(year, mon, day, 0, 0, 0, 0, time.UTC)
		if d.Month() != mon {
			break
		}
		totalDaysInMonth++
	}

	totalDays := 0
	totalTracks := 0

	for day := 1; day <= 31; day++ {
		d := time.Date(year, mon, day, 0, 0, 0, 0, time.UTC)
		if d.Month() != mon {
			break
		}
		dateStr := d.Format("2006-01-02")
		count, err := ScrapeSingleDate(dateStr)
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
func ScrapeYear(yearStr string, progressCallback func(current, total int)) (int, int, error) {
	y, err := strconv.Atoi(yearStr)
	if err != nil || y < 1 {
		return 0, 0, fmt.Errorf("invalid year format (use YYYY): %v", err)
	}

	// Count total days in year first
	totalDaysInYear := 0
	for m := 1; m <= 12; m++ {
		t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		mon := t.Month()
		for day := 1; day <= 31; day++ {
			d := time.Date(y, mon, day, 0, 0, 0, 0, time.UTC)
			if d.Month() != mon {
				break
			}
			totalDaysInYear++
		}
	}

	totalDays := 0
	totalTracks := 0

	for m := 1; m <= 12; m++ {
		t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		mon := t.Month()
		for day := 1; day <= 31; day++ {
			d := time.Date(y, mon, day, 0, 0, 0, 0, time.UTC)
			if d.Month() != mon {
				break
			}
			dateStr := d.Format("2006-01-02")
			count, err := ScrapeSingleDate(dateStr)
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

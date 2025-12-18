package main

import (
	"context"
	"encoding/csv"
	"flag"
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

func main() {
	date := flag.String("date", "today", "Date to scrape (e.g. today or 2025-12-01)")
	month := flag.String("month", "", "Month to scrape all days (e.g. 2025-11)")
	year := flag.String("year", "", "Year to scrape all days (e.g. 2025)")
	flag.Parse()

	// Helper to scrape a single date
	scrapeDate := func(dateStr string) {
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
			fmt.Println("Error loading page or finding tracks:", err)
			return
		}

		type Track struct {
			Title, Artist, Album, Label, ReleaseDate, Listen string
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
			// Use unescaped quotes for matching
			title := extractBetween(html, `<h1 class="Track_title__rH4bg">`, `</h1>`)
			artist := extractBetween(html, `<h2 class="Track_artist__MQzWB">`, `</h2>`)
			album := extractBetween(html, `<p class="Track_album__xqOrO">Album: <span class="Track_value__KiAxm">`, `</span>`)
			label := extractBetween(html, `<p class="Track_label__HS9Oh">Label: <span class="Track_value__KiAxm">`, `</span>`)
			releaseDate := extractBetween(html, `<p class="Track_releaseDate__rpmm5">Release date: <span class="Track_value__KiAxm">`, `</span>`)
			listen := extractHref(html, `<a href="`, `" target=`)
			fmt.Printf("Track: %s\nArtist: %s\nAlbum: %s\nLabel: %s\nRelease Date: %s\nListen: %s\n---\n",
				title, artist, album, label, releaseDate, listen)
			tracks = append(tracks, Track{title, artist, album, label, releaseDate, listen})
		}

		// Write to CSV in out/tracks-[date].csv
		outDir := "out"
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fmt.Println("Error creating out directory:", err)
			return
		}
		outFile := filepath.Join(outDir, fmt.Sprintf("tracks-%s.csv", dateStr))
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Println("Error creating CSV file:", err)
			return
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
	}

	if *year != "" {
		y, err := strconv.Atoi(*year)
		if err != nil || y < 1 {
			fmt.Println("Invalid year format. Use YYYY (e.g. 2025)")
			return
		}
		for m := 1; m <= 12; m++ {
			t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
			mon := t.Month()
			for day := 1; day <= 31; day++ {
				d := time.Date(y, mon, day, 0, 0, 0, 0, time.UTC)
				if d.Month() != mon {
					break
				}
				dateStr := d.Format("2006-01-02")
				scrapeDate(dateStr)
			}
		}
		return
	}

	if *month != "" {
		// Parse year and month
		t, err := time.Parse("2006-01", *month)
		if err != nil {
			fmt.Println("Invalid month format. Use YYYY-MM (e.g. 2025-11)")
			return
		}
		year, mon := t.Year(), t.Month()
		// Iterate through all days in the month
		for day := 1; day <= 31; day++ {
			d := time.Date(year, mon, day, 0, 0, 0, 0, time.UTC)
			if d.Month() != mon {
				break
			}
			dateStr := d.Format("2006-01-02")
			scrapeDate(dateStr)
		}
		return
	}

	// Default: single date
	scrapeDate(*date)
}

// extractBetween returns the string between start and end, or "" if not found
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

// extractHref returns the first href value between start and end
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

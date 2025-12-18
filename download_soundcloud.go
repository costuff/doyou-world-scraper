package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

func extractPlayableFromAnchors(htmlStr string) string {
	// Parse HTML and extract all <a href=...>
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}
	var urls []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					urls = append(urls, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	// Priority: SoundCloud > YouTube > YouTube Music > Spotify > Apple Music
	for _, u := range urls {
		if strings.Contains(u, "soundcloud.com/") {
			return u
		}
	}
	for _, u := range urls {
		if strings.Contains(u, "youtube.com/") || strings.Contains(u, "youtu.be/") {
			return u
		}
	}
	for _, u := range urls {
		if strings.Contains(u, "music.youtube.com/") {
			return u
		}
	}
	for _, u := range urls {
		if strings.Contains(u, "spotify.com/") {
			return u
		}
	}
	for _, u := range urls {
		if strings.Contains(u, "music.apple.com/") {
			return u
		}
	}
	return ""
}

func resolvePlayableURL(link string) (string, error) {
	if !strings.Contains(link, "lis.tn/") {
		return link, nil
	}
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var htmlStr string
	var iframeSrcs []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(link),
		chromedp.OuterHTML("html", &htmlStr, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('iframe')).map(f => f.src)`, &iframeSrcs),
	)
	if err != nil {
		return "", fmt.Errorf("chromedp error: %v", err)
	}
	// Try anchors in main HTML
	playable := extractPlayableFromAnchors(htmlStr)
	if playable != "" {
		return playable, nil
	}
	// Try SoundCloud/YouTube regex in main HTML
	reSC := regexp.MustCompile(`https://soundcloud\\.com/[^"]+`)
	matchSC := reSC.FindString(htmlStr)
	if matchSC != "" {
		return matchSC, nil
	}
	reYT := regexp.MustCompile(`https://(www\\.)?(youtube\\.com|youtu\\.be)/[^"]+`)
	matchYT := reYT.FindString(htmlStr)
	if matchYT != "" {
		return matchYT, nil
	}
	// Search all iframes for playable URLs
	for _, src := range iframeSrcs {
		if src == "" {
			continue
		}
		var iframeHTML string
		err := chromedp.Run(ctx,
			chromedp.Navigate(src),
			chromedp.OuterHTML("html", &iframeHTML, chromedp.ByQuery),
		)
		if err != nil {
			fmt.Printf("[DEBUG] Error loading iframe %s: %v\n", src, err)
			continue
		}
		playable := extractPlayableFromAnchors(iframeHTML)
		if playable != "" {
			return playable, nil
		}
		matchSC := reSC.FindString(iframeHTML)
		if matchSC != "" {
			return matchSC, nil
		}
		matchYT := reYT.FindString(iframeHTML)
		if matchYT != "" {
			return matchYT, nil
		}
	}
	// Print HTML for debugging
	fmt.Println("[DEBUG] Main HTML:\n", htmlStr)
	for _, src := range iframeSrcs {
		if src == "" {
			continue
		}
		var iframeHTML string
		err := chromedp.Run(ctx,
			chromedp.Navigate(src),
			chromedp.OuterHTML("html", &iframeHTML, chromedp.ByQuery),
		)
		if err == nil {
			fmt.Printf("[DEBUG] Iframe %s HTML:\n%s\n", src, iframeHTML)
		}
	}
	return "", fmt.Errorf("No SoundCloud, YouTube, Spotify, or Apple Music URL found in %s", link)
}

func main() {
	csvPath := flag.String("csv", "", "CSV file or folder with SoundCloud links (required)")
	col := flag.Int("col", 5, "Column index (0-based) for SoundCloud link in CSV")
	outDir := flag.String("out", "downloads", "Output directory for downloads")
	ytdlpPath := flag.String("ytdlp", "yt-dlp", "Path to yt-dlp executable")
	flag.Parse()

	if *csvPath == "" {
		fmt.Println("--csv is required")
		return
	}
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Println("Error creating output dir:", err)
		return
	}

	var files []string
	fi, err := os.Stat(*csvPath)
	if err != nil {
		fmt.Println("Error accessing path:", err)
		return
	}
	if fi.IsDir() {
		files, err = filepath.Glob(filepath.Join(*csvPath, "*.csv"))
		if err != nil {
			fmt.Println("Error listing CSV files:", err)
			return
		}
		if len(files) == 0 {
			fmt.Println("No CSV files found in folder.")
			return
		}
	} else {
		files = []string{*csvPath}
	}

	var failedSongs [][]string
	var totalSongs, failedCount int
	for _, csvFile := range files {
		f, err := os.Open(csvFile)
		if err != nil {
			fmt.Println("Error opening CSV:", csvFile, err)
			continue
		}
		defer f.Close()
		r := csv.NewReader(f)
		records, err := r.ReadAll()
		if err != nil {
			fmt.Println("Error reading CSV:", csvFile, err)
			continue
		}
		for i, rec := range records {
			if i == 0 {
				continue // skip header
			}
			if *col >= len(rec) {
				continue
			}
			link := rec[*col]
			if link == "" {
				continue
			}
			totalSongs++
			resolved, err := resolvePlayableURL(link)
			if err != nil {
				fmt.Printf("Warning: %v\n", err)
				failedSongs = append(failedSongs, rec)
				failedCount++
				continue
			}
			fmt.Printf("Downloading: %s\n", resolved)
			// Prepare metadata from CSV columns
			title := ""
			artist := ""
			album := ""
			label := ""
			release := ""
			if len(rec) > 0 {
				title = rec[0]
			}
			if len(rec) > 1 {
				artist = rec[1]
			}
			if len(rec) > 2 {
				album = rec[2]
			}
			if len(rec) > 3 {
				label = rec[3]
			}
			if len(rec) > 4 {
				release = rec[4]
			}
			// yt-dlp args for mp3 and metadata
			args := []string{"-P", *outDir, "--extract-audio", "--audio-format", "mp3", "--add-metadata"}
			if title != "" {
				args = append(args, "--metadata", fmt.Sprintf("title:%s", title))
			}
			if artist != "" {
				args = append(args, "--metadata", fmt.Sprintf("artist:%s", artist))
			}
			if album != "" {
				args = append(args, "--metadata", fmt.Sprintf("album:%s", album))
			}
			if label != "" {
				args = append(args, "--metadata", fmt.Sprintf("label:%s", label))
			}
			if release != "" {
				args = append(args, "--metadata", fmt.Sprintf("date:%s", release))
			}
			args = append(args, resolved)

			// Try to determine output filename (yt-dlp default: %(title)s.%(ext)s)
			// Use title if available, else fallback to URL id
			outFile := ""
			if title != "" {
				outFile = fmt.Sprintf("%s/%s.mp3", *outDir, sanitizeFilename(title))
			}
			if outFile == "" {
				// fallback: try to extract id from resolved url
				parts := strings.Split(resolved, "/")
				if len(parts) > 0 {
					outFile = fmt.Sprintf("%s/%s.mp3", *outDir, parts[len(parts)-1])
				}
			}
			if outFile != "" {
				if _, err := os.Stat(outFile); err == nil {
					fmt.Printf("Already downloaded: %s\n", outFile)
					continue
				}
			}

			cmd := exec.Command(*ytdlpPath, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				fmt.Printf("Failed to download %s: %v\n", resolved, err)
				failedSongs = append(failedSongs, rec)
				failedCount++
			}
		}
	}

	// Write failed songs to CSV in downloads folder
	failedPath := ""
	if len(failedSongs) > 0 {
		failedPath = filepath.Join(*outDir, "failed_songs.csv")
		failedFile, err := os.Create(failedPath)
		if err != nil {
			fmt.Println("Error creating failed_songs.csv:", err)
		} else {
			defer failedFile.Close()
			w := csv.NewWriter(failedFile)
			defer w.Flush()
			for _, row := range failedSongs {
				w.Write(row)
			}
		}
	}
	fmt.Printf("\nSummary: %d out of %d songs failed to download.\n", failedCount, totalSongs)
	if failedCount > 0 && failedPath != "" {
		absPath, err := filepath.Abs(failedPath)
		if err == nil {
			fmt.Printf("Failed songs written to: %s\n", absPath)
		} else {
			fmt.Printf("Failed songs written to: %s\n", failedPath)
		}
	}
}

func sanitizeFilename(name string) string {
	// Remove or replace characters not allowed in filenames
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range forbidden {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

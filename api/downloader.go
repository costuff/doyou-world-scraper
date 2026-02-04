package main

import (
	"context"
	"encoding/csv"
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

type DownloadResult struct {
	TotalSongs  int
	Failed      int
	Downloaded  int
	FailedSongs [][]string
}

func extractPlayableFromAnchors(htmlStr string) string {
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
	playable := extractPlayableFromAnchors(htmlStr)
	if playable != "" {
		return playable, nil
	}
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
	return "", fmt.Errorf("no playable URL found in %s", link)
}

// DownloadFromCSV downloads tracks from a single CSV file
func DownloadFromCSV(csvPath string, col int, outDir string, ytdlpPath string, job *Job, progressCallback func(current, total int)) (*DownloadResult, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating output dir: %v", err)
	}

	f, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %v", err)
	}

	result := &DownloadResult{}

	for i, rec := range records {
		// Check if job is cancelled
		if job != nil && job.IsCancelled() {
			return result, fmt.Errorf("job cancelled")
		}

		if i == 0 {
			continue // skip header
		}
		if col >= len(rec) {
			continue
		}
		link := rec[col]
		if link == "" {
			continue
		}

		result.TotalSongs++

		if progressCallback != nil {
			progressCallback(i, len(records))
		}

		resolved, err := resolvePlayableURL(link)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			result.FailedSongs = append(result.FailedSongs, rec)
			result.Failed++
			continue
		}

		fmt.Printf("Downloading: %s\n", resolved)

		// Prepare metadata
		title, artist, album, label, release := "", "", "", "", ""
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

		// Check if already downloaded
		outFile := ""
		if title != "" {
			outFile = fmt.Sprintf("%s/%s.mp3", outDir, sanitizeFilename(title))
		}
		if outFile != "" {
			if _, err := os.Stat(outFile); err == nil {
				fmt.Printf("Already downloaded: %s\n", outFile)
				result.Downloaded++
				continue
			}
		}

		// yt-dlp args
		args := []string{"-P", outDir, "--extract-audio", "--audio-format", "mp3", "--add-metadata"}
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

		cmd := exec.Command(ytdlpPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Printf("Failed to download %s: %v\n", resolved, err)
			result.FailedSongs = append(result.FailedSongs, rec)
			result.Failed++
		} else {
			result.Downloaded++
		}
	}

	// Write failed songs if any
	if len(result.FailedSongs) > 0 {
		failedPath := filepath.Join(outDir, "failed_songs.csv")
		failedFile, err := os.Create(failedPath)
		if err == nil {
			defer failedFile.Close()
			w := csv.NewWriter(failedFile)
			defer w.Flush()
			for _, row := range result.FailedSongs {
				_ = w.Write(row) // Error is logged elsewhere
			}
		}
	}

	return result, nil
}

// DownloadFromFolder downloads from all CSV files in a folder
func DownloadFromFolder(folderPath string, col int, outDir string, ytdlpPath string, job *Job, progressCallback func(current, total int)) (*DownloadResult, error) {
	csvFiles, err := filepath.Glob(filepath.Join(folderPath, "*.csv"))
	if err != nil || len(csvFiles) == 0 {
		return nil, fmt.Errorf("no CSV files found in %s", folderPath)
	}

	totalResult := &DownloadResult{}

	for i, csvFile := range csvFiles {
		// Check if job is cancelled
		if job != nil && job.IsCancelled() {
			return totalResult, fmt.Errorf("job cancelled")
		}

		fmt.Printf("\n=== Processing %s (%d/%d) ===\n", filepath.Base(csvFile), i+1, len(csvFiles))

		if progressCallback != nil {
			progressCallback(i, len(csvFiles))
		}

		result, err := DownloadFromCSV(csvFile, col, outDir, ytdlpPath, job, nil)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", csvFile, err)
			continue
		}

		totalResult.TotalSongs += result.TotalSongs
		totalResult.Downloaded += result.Downloaded
		totalResult.Failed += result.Failed
		totalResult.FailedSongs = append(totalResult.FailedSongs, result.FailedSongs...)
	}

	return totalResult, nil
}

func sanitizeFilename(name string) string {
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range forbidden {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

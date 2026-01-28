package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// normalize returns a lowercased, alphanumeric-only version of a string for fuzzy matching
func normalize(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9 ]+`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}

// wordSet returns a set of words from a normalized string
func wordSet(s string) map[string]struct{} {
	ws := make(map[string]struct{})
	for _, w := range strings.Fields(s) {
		ws[w] = struct{}{}
	}
	return ws
}

// overlapCount returns the number of words in common between two sets
func overlapCount(a, b map[string]struct{}) int {
	count := 0
	for w := range a {
		if _, ok := b[w]; ok {
			count++
		}
	}
	return count
}

type trackRef struct {
	csvFile string
	row     int
}

func main() {
	csvDir := flag.String("csv", "csv", "Path to the folder with CSV files")
	downloadsDir := flag.String("downloads", "downloads", "Path to the downloads folder")
	flag.Parse()

	// Find all CSV files in csvDir
	csvFiles, err := filepath.Glob(filepath.Join(*csvDir, "*.csv"))
	if err != nil || len(csvFiles) == 0 {
		fmt.Println("No CSV files found in", *csvDir)
		os.Exit(1)
	}

	// Build a slice of CSV track info for fuzzy matching
	type csvTrack struct {
		title, artist       string
		titleSet, artistSet map[string]struct{}
		csvFile             string
		row                 int
	}
	var csvTracks []csvTrack
	for _, csvPath := range csvFiles {
		csvFile, err := os.Open(csvPath)
		if err != nil {
			fmt.Println("Error opening CSV:", csvPath, err)
			continue
		}
		r := csv.NewReader(csvFile)
		records, err := r.ReadAll()
		csvFile.Close()
		if err != nil {
			fmt.Println("Error reading CSV:", csvPath, err)
			continue
		}
		for i, rec := range records {
			if i == 0 || len(rec) < 2 {
				continue // skip header or incomplete
			}
			title := normalize(rec[0])
			artist := normalize(rec[1])
			csvTracks = append(csvTracks, csvTrack{
				title:     title,
				artist:    artist,
				titleSet:  wordSet(title),
				artistSet: wordSet(artist),
				csvFile:   filepath.Base(csvPath),
				row:       i,
			})
		}
	}

	// List all mp3 files in downloads (recursively)
	var files []string
	err = filepath.Walk(*downloadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking downloads folder:", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d mp3 files in %s\n", len(files), *downloadsDir)

	// Try to match each file to a CSV entry using word overlap
	matched := 0
	csvMatchCount := make(map[string]int)
	for _, f := range files {
		base := filepath.Base(f)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		parts := strings.Split(name, " - ")
		var title, artist string
		if len(parts) == 2 {
			artist = normalize(parts[0])
			title = normalize(parts[1])
		} else {
			title = normalize(name)
			artist = ""
		}
		fileTitleSet := wordSet(title)
		fileArtistSet := wordSet(artist)

		// Collect all matches with their scores
		type matchScore struct {
			track csvTrack
			score int
		}
		var allMatches []matchScore
		for _, ct := range csvTracks {
			score := overlapCount(fileTitleSet, ct.titleSet) + overlapCount(fileArtistSet, ct.artistSet)
			if score > 0 {
				allMatches = append(allMatches, matchScore{ct, score})
			}
		}
		// Sort by score descending
		sort.Slice(allMatches, func(i, j int) bool {
			return allMatches[i].score > allMatches[j].score
		})
		if len(allMatches) > 0 {
			// Use only the best match for determining which CSV to count
			bestMatch := allMatches[0]
			csvMatchCount[bestMatch.track.csvFile]++

			topN := 3
			if len(allMatches) < topN {
				topN = len(allMatches)
			}
			fmt.Printf("%s => Top %d matches: ", base, topN)
			for i := 0; i < topN; i++ {
				m := allMatches[i]
				fmt.Printf("%s[row %d] (score: %d) ", m.track.csvFile, m.track.row, m.score)
			}
			fmt.Println()
			matched++
		} else {
			fmt.Printf("%s => No match\n", base)
		}
	}
	fmt.Printf("\nMatched %d out of %d files.\n", matched, len(files))

	// Show match counts for all CSVs sorted by match count
	type csvCount struct {
		name  string
		date  string
		count int
	}
	dateRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	var csvCounts []csvCount
	for csv, count := range csvMatchCount {
		date := "0000-00-00"
		match := dateRe.FindStringSubmatch(csv)
		if len(match) > 1 {
			date = match[1]
		}
		csvCounts = append(csvCounts, csvCount{csv, date, count})
	}

	// Sort by match count descending (most matches first)
	sort.Slice(csvCounts, func(i, j int) bool {
		return csvCounts[i].count > csvCounts[j].count
	})

	fmt.Println("\nTop 20 CSVs by match count:")
	displayCount := 20
	if len(csvCounts) < displayCount {
		displayCount = len(csvCounts)
	}
	for i := 0; i < displayCount; i++ {
		cc := csvCounts[i]
		fmt.Printf("  %d. %s: %d matches\n", i+1, cc.name, cc.count)
	}

	// Among the top 20, find the one with the latest date
	if len(csvCounts) > 0 {
		topCount := displayCount
		if len(csvCounts) < topCount {
			topCount = len(csvCounts)
		}

		latestCSV := csvCounts[0]
		latestDate := csvCounts[0].date
		for i := 0; i < topCount; i++ {
			cc := csvCounts[i]
			if cc.date > latestDate {
				latestDate = cc.date
				latestCSV = cc
			}
		}

		fmt.Printf("\nLatest date among top %d CSVs: %s (%d matches, date: %s)\n", topCount, latestCSV.name, latestCSV.count, latestCSV.date)
	} else {
		fmt.Println("\nNo likely CSV file found.")
	}
}

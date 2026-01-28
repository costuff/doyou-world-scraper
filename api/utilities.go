package main

import (
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type DeduplicateResult struct {
	TotalScanned      int
	DuplicatesFound   int
	DuplicatesRemoved int
	Errors            []string
}

// RemoveDuplicates scans directories and removes duplicate MP3 files by MD5 hash
func RemoveDuplicates(downloadsDir, nasDir string) (*DeduplicateResult, error) {
	result := &DeduplicateResult{}
	fileHashes := make(map[[16]byte]string)
	var filesToRemove []string

	scanDirs := []string{downloadsDir}
	if nasDir != "" {
		if _, err := os.Stat(nasDir); err == nil {
			scanDirs = append(scanDirs, nasDir)
		}
	}

	for _, dir := range scanDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && filepath.Ext(info.Name()) == ".mp3" {
				result.TotalScanned++
				hash, err := hashFileMD5(path)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Error hashing %s: %v", path, err))
					return nil
				}
				if orig, exists := fileHashes[hash]; exists {
					fmt.Printf("Duplicate found: %s (original: %s)\n", path, orig)
					filesToRemove = append(filesToRemove, path)
					result.DuplicatesFound++
				} else {
					fileHashes[hash] = path
				}
			}
			return nil
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error walking %s: %v", dir, err))
		}
	}

	// Remove duplicates
	for _, f := range filesToRemove {
		err := os.Remove(f)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to remove %s: %v", f, err))
		} else {
			fmt.Printf("Removed duplicate: %s\n", f)
			result.DuplicatesRemoved++
		}
	}

	return result, nil
}

func hashFileMD5(filePath string) ([16]byte, error) {
	var result [16]byte
	f, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return result, err
	}
	copy(result[:], h.Sum(nil))
	return result, nil
}

// MatchDownloadsResult contains the results of matching downloads to CSVs
type MatchDownloadsResult struct {
	TotalFiles   int
	MatchedFiles int
	TopCSVs      []CSVMatchInfo
	LatestCSV    *CSVMatchInfo
}

type CSVMatchInfo struct {
	Name       string
	Date       string
	MatchCount int
}

// MatchDownloadsToCSVs performs fuzzy matching of downloaded files to CSV entries
func MatchDownloadsToCSVs(outDir, downloadsDir string) (*MatchDownloadsResult, error) {
	result := &MatchDownloadsResult{}

	// Find all CSV files
	csvFiles, err := filepath.Glob(filepath.Join(outDir, "*.csv"))
	if err != nil || len(csvFiles) == 0 {
		return nil, fmt.Errorf("no CSV files found in %s", outDir)
	}

	// Build CSV track database
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
			continue
		}
		r := csv.NewReader(csvFile)
		records, err := r.ReadAll()
		csvFile.Close()
		if err != nil {
			continue
		}
		for i, rec := range records {
			if i == 0 || len(rec) < 2 {
				continue
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

	// List all mp3 files
	var files []string
	err = filepath.Walk(downloadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking downloads: %v", err)
	}

	result.TotalFiles = len(files)
	csvMatchCount := make(map[string]int)

	// Match each file
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

		// Find best match
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
		sort.Slice(allMatches, func(i, j int) bool {
			return allMatches[i].score > allMatches[j].score
		})

		if len(allMatches) > 0 {
			bestMatch := allMatches[0]
			csvMatchCount[bestMatch.track.csvFile]++
			result.MatchedFiles++
		}
	}

	// Build results
	dateRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	for csv, count := range csvMatchCount {
		date := "0000-00-00"
		match := dateRe.FindStringSubmatch(csv)
		if len(match) > 1 {
			date = match[1]
		}
		result.TopCSVs = append(result.TopCSVs, CSVMatchInfo{
			Name:       csv,
			Date:       date,
			MatchCount: count,
		})
	}

	// Sort by match count
	sort.Slice(result.TopCSVs, func(i, j int) bool {
		return result.TopCSVs[i].MatchCount > result.TopCSVs[j].MatchCount
	})

	// Find latest date among top 20
	topCount := 20
	if len(result.TopCSVs) < topCount {
		topCount = len(result.TopCSVs)
	}

	if topCount > 0 {
		latestCSV := result.TopCSVs[0]
		latestDate := result.TopCSVs[0].Date
		for i := 0; i < topCount; i++ {
			if result.TopCSVs[i].Date > latestDate {
				latestDate = result.TopCSVs[i].Date
				latestCSV = result.TopCSVs[i]
			}
		}
		result.LatestCSV = &latestCSV
	}

	// Trim to top 20
	if len(result.TopCSVs) > 20 {
		result.TopCSVs = result.TopCSVs[:20]
	}

	return result, nil
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9 ]+`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}

func wordSet(s string) map[string]struct{} {
	ws := make(map[string]struct{})
	for _, w := range strings.Fields(s) {
		ws[w] = struct{}{}
	}
	return ws
}

func overlapCount(a, b map[string]struct{}) int {
	count := 0
	for w := range a {
		if _, ok := b[w]; ok {
			count++
		}
	}
	return count
}

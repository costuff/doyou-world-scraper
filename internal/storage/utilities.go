package storage

import (
	"crypto/md5"
	"doyou-world-scraper/internal/jobs"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
)

type DeduplicateResult struct {
	TotalScanned      int
	DuplicatesFound   int
	DuplicatesRemoved int
	Errors            []string
}

// RemoveDuplicates scans directories and removes duplicate MP3 files by MD5 hash
func RemoveDuplicates(downloadsDir, nasDir string, job *jobs.Job, progressCallback func(current, total int)) (*DeduplicateResult, error) {
	result := &DeduplicateResult{}
	fileHashes := make(map[[16]byte]string)
	var filesToRemove []string

	scanDirs := []string{downloadsDir}
	if nasDir != "" {
		if _, err := os.Stat(nasDir); err == nil {
			scanDirs = append(scanDirs, nasDir)
		}
	}

	// First pass: count total files for progress reporting
	totalFiles := 0
	for _, dir := range scanDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && filepath.Ext(info.Name()) == ".mp3" {
				totalFiles++
			}
			return nil
		})
	}

	if progressCallback != nil && totalFiles > 0 {
		progressCallback(0, totalFiles)
	}

	// Second pass: scan and hash files
	for _, dir := range scanDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			// Check if job is cancelled
			if job != nil && job.IsCancelled() {
				return fmt.Errorf("job cancelled")
			}

			if err != nil {
				return nil
			}
			if !info.IsDir() && filepath.Ext(info.Name()) == ".mp3" {
				result.TotalScanned++

				// Report progress every 10 files
				if progressCallback != nil && totalFiles > 0 && result.TotalScanned%10 == 0 {
					progressCallback(result.TotalScanned, totalFiles)
				}

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

// TrackInfo represents metadata for a single track
type TrackInfo struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Year     int    `json:"year"`
	Genre    string `json:"genre"`
	Duration int    `json:"duration"`  // in seconds
	FileSize int64  `json:"file_size"` // in bytes
	FilePath string `json:"file_path"`
	ModTime  string `json:"mod_time"`
}

// TrackListResult represents paginated track listing
type TrackListResult struct {
	Tracks     []TrackInfo `json:"tracks"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
	InProgress bool        `json:"in_progress,omitempty"`
	Scanned    int         `json:"scanned,omitempty"`
}

type TrackScanStatus struct {
	InProgress  bool   `json:"in_progress"`
	Scanned     int    `json:"scanned"`
	Total       int    `json:"total"`
	LastUpdated string `json:"last_updated"`
	Error       string `json:"error,omitempty"`
}

type trackCacheEntry struct {
	Tracks   []TrackInfo
	CachedAt time.Time
}

var (
	trackCache   = map[string]trackCacheEntry{}
	trackCacheMu sync.RWMutex

	trackScanStatus = map[string]*TrackScanStatus{}
	trackScanMu     sync.RWMutex
)

const trackCacheTTL = 10 * time.Minute
const minTrackSizeBytes int64 = 64 * 1024

func getTrackScanStatus(dir string) TrackScanStatus {
	trackScanMu.RLock()
	defer trackScanMu.RUnlock()
	if status, ok := trackScanStatus[dir]; ok && status != nil {
		return *status
	}
	return TrackScanStatus{InProgress: false, Scanned: 0, Total: 0, LastUpdated: ""}
}

func GetTrackScanStatus(dir string) TrackScanStatus {
	return getTrackScanStatus(dir)
}

func setTrackScanStatus(dir string, status TrackScanStatus) {
	trackScanMu.Lock()
	defer trackScanMu.Unlock()
	trackScanStatus[dir] = &status
}

func updateTrackScanProgress(dir string, scanned int, inProgress bool, errMsg string) {
	trackScanMu.Lock()
	defer trackScanMu.Unlock()
	status, ok := trackScanStatus[dir]
	if !ok || status == nil {
		status = &TrackScanStatus{}
		trackScanStatus[dir] = status
	}
	status.InProgress = inProgress
	status.Scanned = scanned
	status.LastUpdated = time.Now().Format(time.RFC3339)
	if errMsg != "" {
		status.Error = errMsg
	}
}

func getCachedTracks(dir string) (tracks []TrackInfo, ok bool, stale bool) {
	trackCacheMu.RLock()
	defer trackCacheMu.RUnlock()
	entry, ok := trackCache[dir]
	if !ok {
		return nil, false, true
	}
	return entry.Tracks, true, time.Since(entry.CachedAt) > trackCacheTTL
}

func setCachedTracks(dir string, tracks []TrackInfo) {
	trackCacheMu.Lock()
	defer trackCacheMu.Unlock()
	trackCache[dir] = trackCacheEntry{Tracks: tracks, CachedAt: time.Now()}
}

func scanAllTracks(downloadsDir string) ([]TrackInfo, error) {
	setTrackScanStatus(downloadsDir, TrackScanStatus{
		InProgress:  true,
		Scanned:     0,
		Total:       0,
		LastUpdated: time.Now().Format(time.RFC3339),
	})

	var allTracks []TrackInfo
	scanned := 0

	err := filepath.Walk(downloadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() || filepath.Ext(info.Name()) != ".mp3" {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") || strings.HasPrefix(info.Name(), "._") {
			return nil
		}
		if info.Size() < minTrackSizeBytes {
			return nil
		}

		track := TrackInfo{
			Filename: info.Name(),
			FilePath: path,
			FileSize: info.Size(),
			ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
		}

		// Try to read ID3 tags
		if file, err := os.Open(path); err == nil {
			defer file.Close()
			if metadata, err := readID3Tags(file); err == nil {
				track.Title = metadata.Title
				track.Artist = metadata.Artist
				track.Album = metadata.Album
				track.Year = metadata.Year
				track.Genre = metadata.Genre
			}
		}

		allTracks = append(allTracks, track)
		scanned++
		if scanned%25 == 0 {
			updateTrackScanProgress(downloadsDir, scanned, true, "")
		}

		return nil
	})

	if err != nil {
		updateTrackScanProgress(downloadsDir, scanned, false, fmt.Sprintf("scan error: %v", err))
		return nil, fmt.Errorf("error scanning downloads directory: %v", err)
	}

	updateTrackScanProgress(downloadsDir, scanned, false, "")
	return allTracks, nil
}

func applySearchFilter(tracks []TrackInfo, search string) []TrackInfo {
	if search == "" {
		return tracks
	}
	searchLower := strings.ToLower(search)
	filtered := make([]TrackInfo, 0, len(tracks))
	for _, track := range tracks {
		if !strings.Contains(strings.ToLower(track.Filename), searchLower) &&
			!strings.Contains(strings.ToLower(track.Title), searchLower) &&
			!strings.Contains(strings.ToLower(track.Artist), searchLower) &&
			!strings.Contains(strings.ToLower(track.Album), searchLower) {
			continue
		}
		filtered = append(filtered, track)
	}
	return filtered
}

// GetTracks lists tracks from downloads folder with pagination and filtering
func GetTracks(downloadsDir string, page, perPage int, search, sortBy, sortOrder string, async bool) (*TrackListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	// Normalize path to absolute for consistent cache keys
	if absPath, err := filepath.Abs(downloadsDir); err == nil {
		downloadsDir = absPath
	}

	var allTracks []TrackInfo
	var inProgress bool
	var scanned int

	if cachedTracks, ok, stale := getCachedTracks(downloadsDir); ok && !stale {
		allTracks = cachedTracks
	} else if async {
		if cachedTracks, ok, _ := getCachedTracks(downloadsDir); ok {
			allTracks = cachedTracks
		}
		status := getTrackScanStatus(downloadsDir)
		if !status.InProgress {
			go func() {
				tracks, err := scanAllTracks(downloadsDir)
				if err == nil {
					setCachedTracks(downloadsDir, tracks)
				}
			}()
			status = getTrackScanStatus(downloadsDir)
		}
		inProgress = true
		scanned = status.Scanned
	} else {
		tracks, err := scanAllTracks(downloadsDir)
		if err != nil {
			return nil, err
		}
		setCachedTracks(downloadsDir, tracks)
		allTracks = tracks
	}

	allTracks = applySearchFilter(allTracks, search)

	// Sort tracks
	sortTracks(allTracks, sortBy, sortOrder)

	// Calculate pagination
	total := len(allTracks)
	totalPages := (total + perPage - 1) / perPage
	start := (page - 1) * perPage
	end := start + perPage

	if start >= total {
		allTracks = []TrackInfo{}
	} else {
		if end > total {
			end = total
		}
		allTracks = allTracks[start:end]
	}

	return &TrackListResult{
		Tracks:     allTracks,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		InProgress: inProgress,
		Scanned:    scanned,
	}, nil
}

func sortTracks(tracks []TrackInfo, sortBy, sortOrder string) {
	less := func(i, j int) bool {
		var result bool
		switch sortBy {
		case "title":
			result = strings.ToLower(tracks[i].Title) < strings.ToLower(tracks[j].Title)
		case "artist":
			result = strings.ToLower(tracks[i].Artist) < strings.ToLower(tracks[j].Artist)
		case "date", "mod_time":
			result = tracks[i].ModTime < tracks[j].ModTime
		case "size":
			result = tracks[i].FileSize < tracks[j].FileSize
		case "duration":
			result = tracks[i].Duration < tracks[j].Duration
		default: // filename
			result = strings.ToLower(tracks[i].Filename) < strings.ToLower(tracks[j].Filename)
		}
		if sortOrder == "desc" {
			return !result
		}
		return result
	}
	sort.Slice(tracks, less)
}

// Metadata represents ID3 tag information
type Metadata struct {
	Title  string
	Artist string
	Album  string
	Year   int
	Genre  string
}

// readID3Tags reads ID3 metadata from an MP3 file
func readID3Tags(file *os.File) (*Metadata, error) {
	m, err := tag.ReadFrom(file)
	if err != nil {
		return &Metadata{}, err
	}

	return &Metadata{
		Title:  m.Title(),
		Artist: m.Artist(),
		Album:  m.Album(),
		Year:   m.Year(),
		Genre:  m.Genre(),
	}, nil
}

// MoveToNASResult represents the result of moving files to NAS
type MoveToNASResult struct {
	TotalFiles   int      `json:"total_files"`
	MovedFiles   int      `json:"moved_files"`
	SkippedFiles int      `json:"skipped_files"`
	FailedFiles  int      `json:"failed_files"`
	Errors       []string `json:"errors"`
}

// MoveDownloadsToNAS moves MP3 files from downloads directory to NAS
func MoveDownloadsToNAS(downloadsDir, nasDir string, job *jobs.Job, progressCallback func(current, total int)) (*MoveToNASResult, error) {
	result := &MoveToNASResult{}

	// Check if directories exist
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("downloads directory does not exist: %s", downloadsDir)
	}
	if _, err := os.Stat(nasDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("NAS directory does not exist: %s", nasDir)
	}

	// Get list of MP3 files in downloads
	var files []string
	err := filepath.Walk(downloadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			// Skip hidden files
			if strings.HasPrefix(info.Name(), ".") || strings.HasPrefix(info.Name(), "._") {
				return nil
			}
			// Skip small files
			if info.Size() < minTrackSizeBytes {
				return nil
			}
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning downloads directory: %v", err)
	}

	result.TotalFiles = len(files)

	// Move each file
	for i, srcPath := range files {
		// Check if job is cancelled
		if job != nil && job.IsCancelled() {
			return result, fmt.Errorf("job cancelled")
		}

		if progressCallback != nil {
			progressCallback(i+1, result.TotalFiles)
		}

		filename := filepath.Base(srcPath)
		dstPath := filepath.Join(nasDir, filename)

		// Check if file already exists on NAS
		if _, err := os.Stat(dstPath); err == nil {
			result.SkippedFiles++
			continue
		}

		// Copy file to NAS
		if err := copyFile(srcPath, dstPath); err != nil {
			result.FailedFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to copy %s: %v", filename, err))
			continue
		}

		// Verify file was copied successfully
		srcInfo, _ := os.Stat(srcPath)
		dstInfo, err := os.Stat(dstPath)
		if err != nil || srcInfo.Size() != dstInfo.Size() {
			result.FailedFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("Verification failed for %s", filename))
			os.Remove(dstPath) // Clean up failed copy
			continue
		}

		// Delete original file
		if err := os.Remove(srcPath); err != nil {
			result.FailedFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete %s: %v", filename, err))
			continue
		}

		result.MovedFiles++
	}

	return result, nil
}

// ClearCSVsResult contains statistics about CSV clearing operation
type ClearCSVsResult struct {
	TotalFiles   int      `json:"total_files"`
	DeletedFiles int      `json:"deleted_files"`
	Errors       []string `json:"errors"`
}

// ClearCSVs removes all CSV files from the csv directory
func ClearCSVs(csvDir string) (*ClearCSVsResult, error) {
	result := &ClearCSVsResult{}

	// Check if directory exists
	if _, err := os.Stat(csvDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("CSV directory does not exist: %s", csvDir)
	}

	// Get list of CSV files
	files, err := filepath.Glob(filepath.Join(csvDir, "*.csv"))
	if err != nil {
		return nil, fmt.Errorf("error scanning CSV directory: %v", err)
	}

	result.TotalFiles = len(files)

	// Delete each CSV file
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete %s: %v", filepath.Base(file), err))
			continue
		}
		result.DeletedFiles++
	}

	return result, nil
}

// ClearDownloadsResult contains statistics about downloads clearing operation
type ClearDownloadsResult struct {
	TotalFiles   int      `json:"total_files"`
	DeletedFiles int      `json:"deleted_files"`
	Errors       []string `json:"errors"`
}

// ClearDownloads removes all MP3 files from the downloads directory
func ClearDownloads(downloadsDir string) (*ClearDownloadsResult, error) {
	result := &ClearDownloadsResult{}

	// Check if directory exists
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("downloads directory does not exist: %s", downloadsDir)
	}

	// Get list of MP3 files
	var files []string
	err := filepath.Walk(downloadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning downloads directory: %v", err)
	}

	result.TotalFiles = len(files)

	// Delete each MP3 file
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete %s: %v", filepath.Base(file), err))
			continue
		}
		result.DeletedFiles++
	}

	return result, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

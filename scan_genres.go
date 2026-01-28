package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

func main() {
	dir := "downloads"
	files, err := filepath.Glob(filepath.Join(dir, "*.mp3"))
	if err != nil {
		fmt.Println("Error finding mp3 files:", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Println("No mp3 files found in", dir)
		return
	}

	fmt.Printf("Scanning %d mp3 files for genre metadata...\n", len(files))
	genreCount := make(map[string]int)
	categorized := 0
	       for _, f := range files {
		       file, err := os.Open(f)
		       if err != nil {
			       fmt.Printf("Error opening %s: %v\n", f, err)
			       continue
		       }
		       tags, err := tag.ReadFrom(file)
		       file.Close()
		       if err != nil {
			       fmt.Printf("Error reading tags from %s: %v\n", f, err)
			       continue
		       }
		       genre := tags.Genre()
		       if genre == "" {
			       genre = "Unknown"
		       } else {
			       categorized++
			       // Move only categorized songs
			       genreFolder := filepath.Join(dir, sanitizeFolderName(genre))
			       if err := os.MkdirAll(genreFolder, 0755); err != nil {
				       fmt.Printf("Error creating folder %s: %v\n", genreFolder, err)
				       continue
			       }
			       dstPath := filepath.Join(genreFolder, filepath.Base(f))
			       if f != dstPath {
				       if err := os.Rename(f, dstPath); err != nil {
					       fmt.Printf("Error moving %s to %s: %v\n", f, dstPath, err)
				       }
			       }
		       }
		       fmt.Printf("%s: %s\n", filepath.Base(f), genre)
		       genreCount[genre]++
	       }

	fmt.Println("\nSummary by genre:")
	for g, c := range genreCount {
		fmt.Printf("%s: %d\n", g, c)
	}
	fmt.Printf("\nCategorized songs: %d out of %d\n", categorized, len(files))
}

// sanitizeFolderName replaces invalid folder characters
func sanitizeFolderName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

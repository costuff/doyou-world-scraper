package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	srcDir := flag.String("src", "downloads", "Source directory to copy songs from")
	dstDir := flag.String("dst", "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks", "Destination directory to copy songs to")
	flag.Parse()

	fmt.Printf("Recursively copying folders and files from %s to %s\n", *srcDir, *dstDir)
	var totalFiles, copiedFiles, skippedFiles int

	err := filepath.Walk(*srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
			return nil
		}
		relPath, _ := filepath.Rel(*srcDir, path)
		dstPath := filepath.Join(*dstDir, relPath)

		if info.IsDir() {
			if relPath == "." {
				return nil // skip root
			}
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				fmt.Printf("Error creating directory %s: %v\n", dstPath, err)
			}
			return nil
		}

		totalFiles++
		// Skip if already exists
		if _, err := os.Stat(dstPath); err == nil {
			fmt.Printf("Skipping (already exists): %s\n", relPath)
			skippedFiles++
			return nil
		}

		if err := copyFile(path, dstPath); err != nil {
			fmt.Printf("Failed to copy %s: %v\n", relPath, err)
		} else {
			fmt.Printf("Copied: %s\n", relPath)
			copiedFiles++
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking source directory:", err)
		os.Exit(1)
	}

	fmt.Printf("\nSummary: %d files processed, %d copied, %d skipped.\n", totalFiles, copiedFiles, skippedFiles)
	fmt.Println("Done.")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

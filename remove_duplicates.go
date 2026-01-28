package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	downloadsDir := flag.String("downloads", "downloads", "Path to the downloads folder")
	nasDir := flag.String("nas", "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks", "Path to the NAS folder (optional)")
	flag.Parse()

	fileHashes := make(map[[16]byte]string)
	var filesToRemove []string

	scanDirs := []string{*downloadsDir}
	if *nasDir != "" {
		scanDirs = append(scanDirs, *nasDir)
	}

	for _, dir := range scanDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && filepath.Ext(info.Name()) == ".mp3" {
				hash, err := hashFileMD5(path)
				if err != nil {
					fmt.Printf("Error hashing %s: %v\n", path, err)
					return nil
				}
				if orig, exists := fileHashes[hash]; exists {
					fmt.Printf("Duplicate found: %s (original: %s)\n", path, orig)
					filesToRemove = append(filesToRemove, path)
				} else {
					fileHashes[hash] = path
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error walking folder %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Remove duplicates
	for _, f := range filesToRemove {
		err := os.Remove(f)
		if err != nil {
			fmt.Printf("Failed to remove %s: %v\n", f, err)
		} else {
			fmt.Printf("Removed duplicate: %s\n", f)
		}
	}
	fmt.Printf("\nRemoved %d duplicate files.\n", len(filesToRemove))
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

# doyou-world-scraper

## Project Overview
This project is a Go-based web scraper for the website doyoutrackid.com. It scrapes track information for a given date, month, or year and saves the results as CSV files in the `out/` directory.

## Usage
- Run with Go: `go run main.go --date=YYYY-MM-DD` (default: today)
- You can also use `--month=YYYY-MM` or `--year=YYYY` to scrape all days in a month or year.
- Output is saved in `out/tracks-<date>.csv`.

## Dependencies
- Uses `chromedp` for headless Chrome browser automation.
- Requires Chrome/Chromium installed on your system.

## Status
- As of January 27, 2026, the project is operational and can be run with the above commands.
- No persistent server or background process is running by default; it is a CLI tool.
- No prior README or documentation was found; this file was generated to summarize the current state.

## Script Summaries

- **main.go**: Scrapes track data from doyoutrackid.com for a given date, month, or year and saves results as CSV files in the `out/` directory.
- **copy_songs.go**: Recursively copies all files and folders from the `downloads/` directory to a specified destination, skipping files that already exist.
- **download_soundcloud.go**: Downloads tracks from SoundCloud, YouTube, Spotify, or Apple Music links found in CSV files using yt-dlp, with metadata tagging and error handling.
- **match_downloads_to_csv.go**: Matches downloaded files to CSV track entries using fuzzy string matching, helping to organize or verify downloads.
- **remove_duplicates.go**: Scans the `downloads/` directory for duplicate mp3 files (by MD5 hash) and removes duplicates, keeping only one copy.
- **scan_genres.go**: Scans mp3 files in `downloads/`, reads their genre metadata, and moves them into genre-based subfolders.

## Notes
- The `downloads/` directory contains a large collection of music files and genre folders, likely for post-processing or archival.
- For troubleshooting, check for errors in the terminal output when running the scraper or scripts.

---
_Last updated: 2026-01-27_

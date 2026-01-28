# DoYou World Radio Scraper - Copilot Instructions

## Project Overview
This is a full-stack music scraper and management system for DoYou World Radio. It scrapes daily playlists from doyoutrackid.com, downloads tracks from streaming platforms, manages duplicates, and provides a REST API for automation.

## Tech Stack
- **Backend**: Go 1.x with Fiber v2 web framework
- **Scraping**: chromedp (headless Chrome) for web scraping
- **Downloading**: yt-dlp for audio extraction from SoundCloud/YouTube/Spotify/Apple Music
- **Storage**: Local filesystem + NAS backup at `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- **API**: REST API on port 3000 with async job tracking
- **Frontend**: (Planned) React dashboard
- **Deployment**: (Planned) Docker + Raspberry Pi/CasaOS

## Project Structure
```
doyou-world-scraper/
├── api/                    # REST API server
│   ├── server.go          # Main API entry point (port 3000)
│   ├── handlers.go        # HTTP request handlers
│   ├── jobs.go            # Job tracking system (async operations)
│   ├── scraper.go         # Scraping logic (ScrapeSingleDate, ScrapeMonth, ScrapeYear)
│   ├── downloader.go      # Download logic (DownloadFromCSV, DownloadFromFolder)
│   ├── utilities.go       # Utilities (RemoveDuplicates, MatchDownloadsToCSVs)
│   ├── websocket.go       # WebSocket hub for real-time updates
│   ├── *_test.go          # Unit tests (16 tests, 53.3% coverage)
│   └── README.md          # API documentation
├── main.go                # CLI scraper (standalone)
├── download_soundcloud.go # CLI downloader (standalone)
├── remove_duplicates.go   # CLI deduplicator (standalone)
├── match_downloads_to_csv.go # CLI matcher (standalone)
├── csv/                   # Scraped CSV files (tracks-YYYY-MM-DD.csv)
├── downloads/             # Downloaded MP3 files (~10,000 files, 40GB)
├── README.md              # Project documentation
└── TODO.md                # Project roadmap

```

## Key Functionality

### 1. Scraping
- Scrapes track listings from `https://doyoutrackid.com/tracks/YYYY-MM-DD`
- Extracts: Title, Artist, Album, Label, ReleaseDate, Listen URL
- Saves to `csv/tracks-YYYY-MM-DD.csv`
- Supports single date, month, or full year scraping
- API: `POST /api/v1/scrape/date` `POST /api/v1/scrape/month` `POST /api/v1/scrape/year`

### 2. Downloading
- Resolves `lis.tn` short URLs to playable platform URLs
- Priority order: SoundCloud > YouTube > YouTube Music > Spotify > Apple Music
- Downloads as MP3 with embedded metadata (title, artist, album, label, date)
- Skips already downloaded files
- Tracks failed downloads in `downloads/failed_songs.csv`
- API: `POST /api/v1/download/csv` `POST /api/v1/download/all`

### 3. Deduplication
- Scans both `downloads/` and NAS folders
- Uses MD5 hashing to detect duplicate MP3 files
- Removes duplicates automatically
- API: `POST /api/v1/util/deduplicate`

### 4. Matching
- Fuzzy matches downloaded MP3 files to CSV entries
- Uses word overlap scoring on title and artist
- Identifies which CSV has the most matches (likely source)
- Shows top 20 CSVs and latest date among top matches
- API: `POST /api/v1/util/match`

### 5. Job Tracking
- All long-running operations use async jobs with UUIDs
- Job statuses: `running`, `completed`, `failed`
- Progress tracking with percentage updates
- Jobs auto-cleanup after 24 hours
- API: `GET /api/v1/jobs` `GET /api/v1/jobs/:id`

## API Endpoints
- `GET /api/v1/health` - Health check
- `POST /api/v1/scrape/date` - Scrape single date (body: `{"date":"2026-01-28"}`)
- `POST /api/v1/scrape/month` - Scrape month (body: `{"month":"2026-01"}`)
- `POST /api/v1/scrape/year` - Scrape year (body: `{"year":"2026"}`)
- `POST /api/v1/download/csv` - Download from CSV (body: `{"csv_path":"csv/tracks-2026-01-28.csv", "column":5}`)
- `POST /api/v1/download/all` - Download from all CSVs
- `POST /api/v1/util/deduplicate` - Remove duplicates (body: `{"downloads_dir":"downloads", "nas_dir":"/Volumes/..."}`)
- `POST /api/v1/util/match` - Match downloads to CSVs
- `GET /api/v1/util/stats` - Get statistics
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get job by ID
- `GET /api/v1/ws` - WebSocket endpoint for real-time job updates

## WebSocket Support
- **CSV Files**: Stored in `csv/` directory
- **Downloads**: 9,875 MP3 files (~40GB) in `downloads/` folder
- **NAS Backup**: 4.3GB at `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- **Match Rate**: ~68% of files match to latest scraped CSV

## Development Guidelines

### Running the API
```bash
cd /Users/skoutelis/Desktop/workspace/doyou-world-scraper
go run api/*.go
# Or build and run:
go build -o doyou-api api/*.go
./doyou-api
```

### Running Tests
```bash
go test ./api -v          # Run all tests
go test ./api -cover      # With coverage
```

### Code Style
- Use Go standard formatting (`gofmt`)
- Handle errors explicitly (return error values)
- Use context with timeouts for chromedp operations
- Clean up resources with `defer` statements
- Add progress callbacks for long operations

### Common Patterns
```go
// Job pattern for async operations
job := NewJob("job_type", fiber.Map{"param": "value"})
go func() {
    result, err := SomeOperation(params)
    if err != nil {
        job.Fail(fmt.Sprintf("Operation failed: %v", err))
        return
    }
    job.Complete(fiber.Map{"result": result})
}()
```

## Next Steps (from TODO.md)
1. **Add WebSocket support** for real-time progress updates
2. **Build React frontend** with dashboard for triggering operations
3. **Music metadata extraction** using audio libraries
4. **Docker deployment** for Raspberry Pi/CasaOS
5. **Authentication** and user management

## Important Notes
- The API server runs on port 3000
- All API operations return 202 Accepted immediately with job ID
- Check job status with `GET /api/v1/jobs/:id`
- CSV column 5 (index 5) contains the Listen URL by default
- NAS path: `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- chromedp needs Chrome/Chromium installed
- yt-dlp must be available in PATH

## Dependencies
```go
github.com/gofiber/fiber/v2
github.com/chromedp/chromedp
github.com/chromedp/cdproto
github.com/google/uuid
golang.org/x/net/html
```

Install with: `go get <package>`

# DoYou World Radio Scraper - Copilot Instructions

## Project Overview
This is a full-stack music scraper and management system for DoYou World Radio. It scrapes daily playlists from doyoutrackid.com, downloads tracks from streaming platforms, manages duplicates, and provides a REST API for automation.

## Tech Stack
- **Backend**: Go 1.x with Fiber v2 web framework
- **Scraping**: chromedp (headless Chrome) for web scraping
- **Downloading**: yt-dlp for audio extraction from SoundCloud/YouTube/Spotify/Apple Music
- **Storage**: Local filesystem + NAS backup at `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- **API**: REST API on port 3000 with async job tracking + WebSocket support
- **Frontend**: React + TypeScript + Vite + Tailwind CSS (port 5173)
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
│   ├── *_test.go          # Unit tests (99 tests, 55.7% coverage)
│   └── README.md          # API documentation
├── frontend/              # React frontend dashboard
│   ├── src/
│   │   ├── components/   # React components (JobCard, ScrapeControls, etc.)
│   │   ├── hooks/        # Custom hooks (useWebSocket)
│   │   ├── services/     # API client (axios)
│   │   ├── types/        # TypeScript types
│   │   ├── App.tsx       # Main app component
│   │   └── main.tsx      # Entry point
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── README.md
├── main.go                # CLI scraper (standalone)
├── download_soundcloud.go # CLI downloader (standalone)
├── remove_duplicates.go   # CLI deduplicator (standalone)
├── match_downloads_to_csv.go # CLI matcher (standalone)
├── csv/                   # Scraped CSV files (tracks-YYYY-MM-DD.csv)
├── downloads/             # Downloaded MP3 files (~10,000 files, 40GB)
├── package.json           # Root package.json for dev workflow
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
- `PEndpoint**: `ws://localhost:3000/api/v1/ws`
- **Message Format**: `{"type": "job_update", "data": {...job}}`
- **Auto-reconnect**: Frontend automatically reconnects on disconnect
- **Real-time Updates**: Job progress broadcasts to all connected clients
- **Ping/Pong**: 30-second heartbeat to keep connections alive

## Frontend Architecture
- **Framework**: React 18 + TypeScript + Vite
- **Styling**: Tailwind CSS with dark slate theme
- **State Management**: React hooks (useState, useEffect, useCallback)
- **API Client**: Axios with proxy to backend
- **WebSocket**: Custom useWebSocket hook for real-time job updates
- **Components**:
  - `ScrapeControls`: Date/month/year pickers with scrape buttons
  - `DownloadControls`: Buttons for downloads and utilities
  - `StatsPanel`: Real-time statistics dashboard (auto-refresh every 30s)
  - `JobsList`: Live job cards with progress bars
  - `JobCard`: Individual job display with status colors
  - `ConnectionStatus`: WebSocket connection indicator

## Data & Statistics
- **CSV Files**: Stored in `csv/` directory
- **Downloads**: 9,875 MP3 files (~40GB) in `downloads/` folder
- **NAS Backup**: 4.3GB at `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- **Match Rate**: ~68% of files match to latest scraped CSV

## Development Guidelines

### Running Both Servers
```bash
# Option 1: Single command (recommended)
npm run dev

# O

### Frontend Development
```bash
cd frontend
npm install               # Install dependencies
npm run dev              # Start dev server (port 5173)
npm run build            # Build for production
npm run preview          # Preview production build
```ption 2: Separate terminals
# Terminal 1:
cd api && go run $(ls *.go | grep -v _test.go)
# Terminal 2:
cd frontend && npm run dev

# Option 3: VS Code tasks
# Cmd+Shift+P → Tasks: Run Task → Start All Servers
```plicate` - Remove duplicates (body: `{"downloads_dir":"downloads", "nas_dir":"/Volumes/..."}`)
- `POST /api/v1/util/match` - Match downloads to CSVs
- `GET /api/v1/util/stats` - Get statistics
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get job by ID
- `GET /api/v1/ws` - WebSocket endpoint for real-time job updates

## ~~**Add WebSocket support** for real-time progress updates~~ ✅ Completed
2. ~~**Build React frontend** with dashboard for triggering operations~~ ✅ Completed
3. **Music metadata extraction** using audio libraries
4. **Docker deployment** for Raspberry Pi/CasaOS
5. **Authentication** and user management
6. **Browse/search tracks** UI in frontendlatest scraped CSV

## Development Guidelines

### Running Both Servers
```bash
# Option 1: Single command (recommended)
npm run dev

# Option 2: Separate terminals
# Terminal 1:
cd api && go run $(ls *.go | grep -v _test.go)
# Terminal 2:
cd frontend && npm run dev

# Option 3: VS Code tasks
# Cmd+Shift+P → Tasks: Run Task → Start All Servers
```

### Testing Requirements
**CRITICAL**: This project follows Test-Driven Development (TDD). All features MUST have tests written first.

**Test-Driven Development Workflow (MANDATORY):**
1. **Write failing test first** - Before writing any production code
2. **Run test and verify it fails** - `cd api && go test -v ./...`
3. **Write minimal code** to make the test pass
4. **Run test and verify it passes**
5. **Refactor** if needed while keeping tests green
6. **Update test stats** in line 29 of this file (test count and coverage %)
7. **Commit** with test and implementation together

**Test Coverage Goals:**
- Maintain **80%+ test coverage** for all Go code (currently at 55.7%)
- Write tests for **every new function** - no exceptions
- Update existing tests when modifying functionality
- Run tests before every commit: `cd api && go test -v ./...`
- Never commit production code without corresponding tests

**Testing Standards:**
- Unit tests: Test individual functions in isolation
- Integration tests: Test API endpoints end-to-end
- Use table-driven tests for multiple scenarios
- Mock external dependencies (network calls, file system when appropriate)
- Use `t.TempDir()` for file system tests
- Add `t.Skip()` for tests that require network/external resources
- Test both happy path and error cases
- Use meaningful test names: `TestFunctionName_Scenario_ExpectedBehavior`

**Test Files Location:**
- `api/*_test.go` - Go test files (handlers_test.go, jobs_test.go, utilities_test.go, etc.)
- Run: `cd api && go test -v ./...`
- Coverage: `cd api && go test -cover ./...`
- Count tests: `go test -v ./... 2>&1 | grep -c "^=== RUN"`

**When Adding Features:**
1. Write tests first (TDD approach preferred)
2. Implement feature
3. Ensure tests pass
4. Check coverage: `go test -cover`
5. **Update test stats in this file** - Update line 29 with: `go test -v ./... 2>&1 | grep -c "^=== RUN"` and `go test -cover`
6. Commit with meaningful message

**Documentation Maintenance:**
- **ALWAYS update test coverage statistics** in line 29 of this file after adding/modifying tests
- Run `cd api && go test -cover` to get coverage percentage
- Run `cd api && go test -v ./... 2>&1 | grep -c "^=== RUN"` to count total tests
- Update format: `# Unit tests (X tests, Y% coverage)`
- Keep API documentation in `api/README.md` synchronized with actual endpoints
- Update `docs/TODO.md` when completing tasks or adding new ones

**Remember:** Tests are not optional. They are part of the feature. A feature without tests is an incomplete feature.

**Documentation:**
- Keep `docs/TODO.md` updated with task status
- Update `docs/DEVELOPMENT.md` when changing dev workflow
- Update API documentation in `api/README.md` for new endpoints
- All API operations return 202 Accepted immediately with job ID
- Check job status with `GET /api/v1/jobs/:id` or via WebSocket
- CSV column 5 (index 5) contains the Listen URL by default
- NAS path: `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- chromedp needs Chrome/Chromium installed
- yt-dlp must be available in PATH
- Frontend proxies `/api` requests to backend automatically

## Dependencies

### Backend (Go)
```go
github.com/gofiber/fiber/v2
github.com/gofiber/websocket/v2
github.com/chromedp/chromedp
github.com/chromedp/cdproto
github.com/google/uuid
golang.org/x/net/html
```

### Frontend (npm)
```json
react, react-dom, axios, date-fns
@types/react, @types/react-dom
vite, @vitejs/plugin-react
tailwindcss, autoprefixer, postcss
typescript, eslint
```

Install with: `go get <package>` or `npm install
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

# DoYou World Radio Scraper

## Project Overview
A full-stack music scraper and management system for DoYou World Radio. Scrapes daily playlists from doyoutrackid.com, downloads tracks from streaming platforms (SoundCloud, YouTube, Spotify, Apple Music), manages duplicates, and provides a React dashboard for monitoring and control.

**Tech Stack:**
- **Backend**: Go 1.23+ with Fiber v2, structured logging (zerolog), WebSocket support
- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS
- **Scraping**: chromedp (headless Chrome)
- **Downloading**: yt-dlp for audio extraction
- **Deployment**: Docker + docker-compose, Raspberry Pi/CasaOS compatible

## Documentation
- [Development Guide](docs/DEVELOPMENT.md) - Local setup, Docker workflows, debugging tips
- [Project Roadmap](docs/TODO.md) - Upcoming features and improvements
- [API Documentation](api/README.md) - REST API endpoints and usage
- [Docker Guide](docs/DOCKER.md) - Docker deployment instructions
- [Pi Setup](docs/PI-SETUP.md) - Raspberry Pi deployment guide

## Project Structure

```
doyou-world-scraper/
├── cmd/
│   └── api/              # API server entry point
│       └── main.go       # Application bootstrap
├── internal/             # Private application code
│   ├── api/              # HTTP handlers and routes
│   ├── config/           # Configuration management
│   ├── downloader/       # Track downloading logic
│   ├── jobs/             # Job tracking and WebSocket
│   ├── scraper/          # Web scraping logic
│   └── storage/          # File operations and utilities
├── pkg/
│   └── logger/           # Structured logging (zerolog)
├── frontend/             # React dashboard
├── data/                 # Data directory (gitignored)
│   ├── csv/              # Scraped CSV files
│   └── downloads/        # Downloaded MP3 files
├── api/                  # Legacy API code (deprecated)
├── docs/                 # Documentation
└── docker-compose.yml    # Docker orchestration
```

## Quick Start

### Local Development

**Prerequisites:**
- Go 1.23+
- Node.js 20+
- Chrome/Chromium
- yt-dlp (`pip install yt-dlp` or `brew install yt-dlp`)

**Run Backend:**
```bash
go run ./cmd/api
# API available at http://localhost:3000
```

**Run Frontend:**
```bash
cd frontend && npm install && npm run dev
# Dashboard available at http://localhost:5173
```

### Docker Deployment

```bash
# Build and run all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

**Services:**
- API: http://localhost:3000
- Frontend: http://localhost:8080

## Features

### ✅ Implemented
- **Scraping**: Scrape tracks by date, month, or year from doyoutrackid.com
- **Downloading**: Download from SoundCloud/YouTube/Spotify/Apple Music with metadata
- **Deduplication**: Remove duplicate MP3s using MD5 hashing
- **REST API**: Full REST API with async job tracking
- **WebSocket**: Real-time progress updates for long-running operations
- **React Dashboard**: Modern UI for monitoring and control
- **Statistics**: Track counts, storage usage, date ranges
- **NAS Support**: Optional NAS storage integration
- **Docker**: Multi-arch Docker images (amd64/arm64)
- **CI/CD**: GitHub Actions with auto-deployment

### 🚧 In Progress
- Music metadata extraction (BPM, key, audio features)
- Track browser UI with search/filter
- Visualization dashboard (genre distribution, audio features)

### 📋 Planned
- SQLite database for persistent job history
- Music recommendation system
- Playlist generation by mood/energy
- Genre classification using ML

## API Endpoints

**Scraping:**
- `POST /api/v1/scrape/date` - Scrape single date
- `POST /api/v1/scrape/month` - Scrape entire month
- `POST /api/v1/scrape/year` - Scrape entire year

**Downloading:**
- `POST /api/v1/download/csv` - Download tracks from CSV
- `POST /api/v1/download/all` - Download from all CSVs

**Utilities:**
- `POST /api/v1/util/deduplicate` - Remove duplicates
- `POST /api/v1/util/match` - Match downloads to CSVs
- `POST /api/v1/util/clear-csvs` - Clear CSV files
- `POST /api/v1/util/clear-downloads` - Clear downloads
- `GET /api/v1/util/stats` - Get statistics

**Jobs:**
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get job by ID
- `POST /api/v1/jobs/:id/cancel` - Cancel job
- `GET /api/v1/ws` - WebSocket endpoint

**Health:**
- `GET /api/v1/health` - Health check

## Configuration

Environment variables (`.env`):
```bash
PORT=3000                    # API port
DATA_DIR=./data              # Base data directory
NAS_PATH=/path/to/nas        # Optional NAS storage path
LOG_LEVEL=info               # debug, info, warn, error
ENVIRONMENT=development      # development, production
TZ=Europe/Athens             # Timezone
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/api/...
```

**Test Stats:**
- Total: 210 tests
- Coverage: ~60%
- Packages: internal/api, internal/jobs, internal/scraper, internal/downloader, internal/storage

## CLI Scripts (Legacy)

These standalone scripts are kept for backward compatibility:

- **main.go**: CLI scraper
- **download_soundcloud.go**: CLI downloader
- **remove_duplicates.go**: CLI deduplicator
- **match_downloads_to_csv.go**: CLI matcher
- **copy_songs.go**: NAS copy utility
- **scan_genres.go**: Genre folder organizer

Use the API/frontend instead for better monitoring and job tracking.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT

---
_Last updated: 2026-02-04_

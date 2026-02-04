# Project TODO List

This file tracks major tasks and features to complete the doyou-world-scraper project and expand it into a full-featured system with a backend, frontend, and advanced music management tools.

## High-Level Goals
- [ ] Build a robust backend (Go/Python) for scraping, downloading, deduplication, and metadata management
- [ ] Create a modern frontend (web UI) for control and monitoring
- [ ] Integrate advanced music metadata extraction and genre classification

## TODOs

### Backend
- [x] Scrape tracklists and download CSVs from doyoutrackid.com (Go)
- [x] Download tracks from SoundCloud/YouTube/Spotify/Apple Music (Go)
- [x] Remove duplicate tracks across local and NAS storage (Go)
- [x] Create REST API server with Go Fiber
- [x] Add API endpoints for all operations (scrape, download, deduplicate, match)
- [x] Implement job tracking system for long-running tasks
- [x] Wire up API endpoints to actual Go script logic (scraper.go, downloader.go, utilities.go)
- [x] Unit tests for API (16 tests, 53.3% coverage)
- [x] Add WebSocket support for real-time progress updates
- [x] Add Move to NAS utility for transferring downloads to NAS storage
- [x] Add Clear CSVs utility for removing scraped CSV files (52.8% test coverage)
- [x] Add Clear Downloads utility for removing all downloaded MP3 files
- [x] Fix WebSocket concurrent write panic with per-connection mutex
- [x] Increase test coverage to 56.2% (up from 50.3%)
- [ ] **Fix download progress tracking** - Progress only updates per CSV file, not per song (needs fine-grained tracking across all songs)
- [ ] **Increase test coverage to 80%+** - Add comprehensive unit and integration tests (currently at 56.2%)
- [ ] Document development workflow for faster iteration without Docker rebuilds

### Frontend
- [x] Build a web dashboard (React + TypeScript + Vite + Tailwind CSS)
- [x] UI to trigger scraping, downloads, deduplication, and metadata scan
- [x] Show progress, logs, and results for each operation
- [x] Real-time job monitoring with WebSocket integration
- [x] Statistics dashboard (track counts, storage usage, date ranges)
- [ ] **Fix frontend not updating properly** - UI doesn't reflect changes/updates consistently
- [ ] Browse downloaded tracks, search/filter by metadata/genre (PRIORITY 1)

## Next Steps Priority Order
1. **Browse/Search UI** - Track library with search/filter/play functionality
2. **Metadata Extraction** - Extract detailed audio metadata (BPM, key, genre, etc.)
3. **Genre Classification** - Auto-categorize tracks using ML models
4. **Docker Deployment** - Package for Raspberry Pi/CasaOS deployment
5. **Authentication** - User login and access control

## Code Quality & Architecture Improvements

### Project Structure Refactoring (HIGH PRIORITY)
**Current Issues:**
- ❌ **Duplicate CSV/downloads folders** - Both root and `api/` have csv/downloads directories causing confusion
- ❌ **Hardcoded paths scattered across codebase** - `./csv`, `./downloads`, `/app/csv`, `/app/downloads` everywhere
- ❌ **No centralized configuration** - Environment variables not properly managed
- ❌ **Standalone CLI scripts at root** - main.go, download_soundcloud.go, etc. duplicating API functionality
- ❌ **Mixed concerns** - Handlers contain business logic and path resolution
- ❌ **No proper logging** - Printf statements instead of structured logging
- ❌ **No database** - All state in memory/files, jobs lost on restart

**Proposed New Structure:**
```
doyou-world-scraper/
├── cmd/
│   ├── api/              # API server entry point
│   │   └── main.go
│   └── cli/              # CLI tools (scraper, downloader)
│       ├── scrape.go
│       ├── download.go
│       └── dedupe.go
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   │   ├── config.go     # Load from env/files
│   │   └── paths.go      # Centralized path resolution
│   ├── scraper/          # Scraping logic
│   │   ├── scraper.go
│   │   └── scraper_test.go
│   ├── downloader/       # Download logic
│   │   ├── downloader.go
│   │   ├── resolver.go
│   │   └── downloader_test.go
│   ├── storage/          # File operations
│   │   ├── csv.go
│   │   ├── downloads.go
│   │   └── storage_test.go
│   ├── jobs/             # Job management
│   │   ├── job.go
│   │   ├── manager.go    # Persistent job storage
│   │   └── jobs_test.go
│   ├── api/              # HTTP handlers
│   │   ├── handlers.go
│   │   ├── middleware.go
│   │   └── routes.go
│   └── metadata/         # Track metadata extraction
│       └── extractor.go
├── pkg/                  # Public/reusable packages
│   └── logger/           # Structured logging
│       └── logger.go
├── data/                 # Single data directory (NOT duplicated)
│   ├── csv/              # Scraped CSV files
│   ├── downloads/        # Downloaded MP3 files
│   └── db/               # SQLite database for jobs/metadata
├── frontend/             # React frontend (unchanged)
├── deployments/          # Deployment configs
│   ├── docker/
│   │   ├── api.Dockerfile
│   │   └── frontend.Dockerfile
│   └── kubernetes/       # Future K8s manifests
├── scripts/              # Build/deploy scripts
│   └── pi-deploy.sh
├── docs/                 # Documentation
├── .env.example          # Environment template
├── go.mod
└── README.md
```

**Benefits:**
- ✅ Single source of truth for data directories
- ✅ Centralized configuration with validation
- ✅ Clear separation of concerns (cmd/internal/pkg pattern)
- ✅ Easier to test and maintain
- ✅ Standard Go project layout
- ✅ Proper logging and error handling
- ✅ Database for persistent state

**Implementation Tasks:**
- [ ] Create new directory structure
- [ ] Extract configuration management into `internal/config`
- [ ] Centralize path resolution (stop hardcoding ./csv, ./downloads)
- [ ] Move API code to `cmd/api` and `internal/api`
- [ ] Refactor standalone scripts into `cmd/cli`
- [ ] Move business logic to `internal/` packages
- [ ] Add structured logging with zerolog or zap
- [ ] Add SQLite database for jobs and metadata
- [ ] Update Dockerfiles to use new structure
- [ ] Update all imports and tests
- [ ] Update documentation

### Configuration Management
- [ ] **Create config package** - Load from env vars with defaults
  ```go
  type Config struct {
    DataDir      string  // Base data directory
    CSVDir       string  // Computed: DataDir/csv
    DownloadsDir string  // Computed: DataDir/downloads
    NASPath      string
    Port         int
    LogLevel     string
  }
  ```
- [ ] **Environment validation** - Fail fast on startup if config invalid
- [ ] **Hot reload config** - Watch .env for changes without restart

### Code Organization
- [ ] **Extract handlers logic** - Move business logic out of handlers.go into service layer
- [ ] **Repository pattern** - Abstract file operations for easier testing
- [ ] **Dependency injection** - Pass dependencies instead of global variables
- [ ] **Consistent error handling** - Custom error types with context
- [ ] **Remove code duplication** - DRY principle for path resolution logic

### Data Management
- [ ] **SQLite database** - Persist jobs, track metadata, scan results
  ```sql
  CREATE TABLE jobs (id TEXT, type TEXT, status TEXT, progress INT, created_at DATETIME);
  CREATE TABLE tracks (id TEXT, filename TEXT, title TEXT, artist TEXT, metadata JSON);
  ```
- [ ] **Database migrations** - Version control for schema changes
- [ ] **Backup/restore** - Regular database backups to NAS

### Logging & Monitoring
- [ ] **Structured logging** - Use zerolog/zap instead of fmt.Printf
- [ ] **Log levels** - DEBUG, INFO, WARN, ERROR with proper filtering
- [ ] **Request logging** - HTTP request/response with timing
- [ ] **Performance metrics** - Track scrape/download speeds, success rates
- [ ] **Health checks** - More comprehensive health endpoint

### Testing Improvements
- [ ] **Test fixtures** - Reusable test data in testdata/ directory
- [ ] **Integration tests** - Test full workflows end-to-end
- [ ] **Mock external services** - yt-dlp, chromedp for reliable tests
- [ ] **Table-driven tests** - Consistent test patterns
- [ ] **Coverage goal: 80%+** - Currently at 56.7%

### Developer Experience
- [ ] **Makefile** - Common commands (test, build, run, docker)
  ```makefile
  test:
    cd internal && go test -v -cover ./...
  run:
    go run cmd/api/main.go
  docker-build:
    docker compose build
  ```
- [ ] **Pre-commit hooks** - Auto-format, lint, test before commit
- [ ] **VSCode launch configs** - Debug configurations for Go
- [ ] **API documentation** - OpenAPI/Swagger spec

### Music Metadata & Categorization
- [ ] Integrate a tool/library to extract audio metadata (artist, title, album, year, genre, etc.)
- [ ] Use a music classification library or ML model to auto-categorize tracks by genre/style
- [ ] Store metadata and categorization results in a database (SQLite/Postgres)
- [ ] Enable manual correction/override of metadata and genre via the frontend

### Infrastructure & Automation
- [x] Add Docker support for easy deployment (including ARM/Raspberry Pi compatibility)
- [x] Make the project deployable on Raspberry Pi/CasaOS
- [x] Expose the web dashboard on a configurable port for remote access to the music library
- [x] GitHub Actions CI/CD with Watchtower auto-deployment
- [ ] Set up scheduled jobs for scraping and downloads
- [ ] Add authentication/user management for the dashboard

### Development Workflow & Testing
- [x] **Document local development setup** - Guide for running without Docker for faster iteration (see DEVELOPMENT.md)
- [ ] **Improve test coverage** - Add unit tests for all core functions, integration tests for API endpoints
- [ ] Add test fixtures and mock data for reliable testing
- [ ] Set up pre-commit hooks for linting and testing
- [ ] **Improve GitHub Actions workflow**:
  - [ ] Protect `develop` branch - no direct pushes, only via PRs
  - [ ] Build Docker images only on PR merge to `develop` or `main`, not on every push
  - [ ] Add CI checks on PRs (tests, linting) before allowing merge
  - [ ] Consider separate staging/production workflows

---
Feel free to add, edit, or check off items as the project evolves!

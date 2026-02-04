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
- [x] Fix download progress tracking - Progress now updates per song with granular tracking (105 tests, 61.4% coverage)
- [x] Add detailed job messages - Jobs now show "Downloading: Track Name - Artist" with real-time updates
- [ ] **Increase test coverage to 80%+** - Add comprehensive unit and integration tests (currently at 57.9%)
- [ ] **Fix golangci-lint configuration** - .golangci.yml has invalid 'skip-dirs' property for v1.64, need to update config or find alternative to exclude legacy scripts
- [ ] **Fix all errcheck linting violations** - Add proper error checking throughout codebase (currently disabled in CI)
- [ ] Document development workflow for faster iteration without Docker rebuilds

### Frontend
- [x] Build a web dashboard (React + TypeScript + Vite + Tailwind CSS)
- [x] UI to trigger scraping, downloads, deduplication, and metadata scan
- [x] Show progress, logs, and results for each operation
- [x] Real-time job monitoring with WebSocket integration
- [x] Statistics dashboard (track counts, storage usage, date ranges)
- [x] Fix frontend not updating properly - Optimized stats refresh (30s→5s), removed expensive JSON comparisons, added React.memo
- [ ] Browse downloaded tracks, search/filter by metadata/genre (PRIORITY 1)
- [ ] Music library analysis and visualization dashboard (PRIORITY 1)

### Music Metadata & Analysis (NEW - PRIORITY 1)
- [ ] **Advanced Metadata Extraction** - Extract comprehensive audio features from downloaded tracks:
  - Album name and artwork
  - Release date
  - Label
  - BPM (beats per minute)
  - Musical key and mode
  - **Audio Features (Shazam-style analysis)**:
    - **Melodicness** - Clear and memorable tune with well-defined musical patterns
    - **Acousticness** - Reliance on acoustic vs electronic instruments
    - **Valence** - Musical positivity/emotional tone (happiness vs sadness)
    - **Danceability** - Tempo stability, rhythmic patterns, beat emphasis
    - **Energy** - Perceived intensity based on tempo, dynamics, and musical density
    - **Instrumentalness** - Likelihood of being instrumental vs vocal
    - **Speechiness** - Presence of spoken words
    - **Liveness** - Presence of live audience
- [ ] **Music Analysis Engine** - Build analysis system to process all 9,875 tracks
  - Batch processing with progress tracking
  - Store metadata in database (SQLite or PostgreSQL)
  - Cache analysis results to avoid re-processing
- [ ] **Visualization Dashboard** - Create interactive data visualizations:
  - **Genre Distribution** - Pie chart or treemap of music genres
  - **Audio Features Graphs**:
    - Scatter plots (Energy vs Danceability, Valence vs Energy, etc.)
    - Radar charts for track profiles
    - Heatmaps showing feature correlations
  - **Timeline Visualizations** - Track releases over time, genre trends
  - **BPM Distribution** - Histogram of tempo ranges
  - **Label Analytics** - Top labels, release counts
  - **Interactive Filtering** - Click to filter by genre, features, date ranges
  - **Spectrograms** - Visual frequency/time representation of audio for individual tracks
    - Mel spectrogram visualization for each song
    - Interactive spectrogram viewer with playback sync
    - Compare spectrograms between similar tracks
- [ ] **Music Recommendation System** - Suggest similar tracks based on audio features
- [ ] **Playlist Generation** - Auto-create playlists by mood, energy, genre clusters

### Separate Analytics Project (NEW)
- [ ] **Create doyou-music-analytics repository** - Separate repo for audio analysis and visualizations
- [ ] **Phase 1: Shared PostgreSQL Database** - Both services read/write to shared PostgreSQL
  - Design schema for tracks, metadata, audio features
  - Scraper writes: basic track info, download status
  - Analytics writes: BPM, key, energy, danceability, etc.
- [ ] **Phase 2: Event-Driven Architecture with Redis** - Decouple services with pub/sub messaging
  - Scraper publishes: track_downloaded, track_updated events
  - Analytics subscribes: processes new tracks for audio analysis
  - Redis pub/sub for real-time event streaming
  - Async job processing with retries
- [ ] **CV Keywords**: Event-driven, microservices, pub/sub, message queue, PostgreSQL, Redis, async processing

## Next Steps Priority Order
1. **Fix Chromium Process Cleanup** - Ensure browser processes are properly terminated (PRIORITY 1)
2. **Music Library Analysis & Visualization** - Advanced metadata extraction + interactive graphs/charts (separate repo)
3. **Browse/Search UI** - Track library with search/filter/play functionality
4. **PostgreSQL Integration** - Replace in-memory storage with persistent database
5. **Event-Driven Architecture** - Redis pub/sub for analytics integration
6. **Docker Deployment** - Package for Raspberry Pi/CasaOS deployment
7. **Authentication** - User login and access control

## Code Quality & Architecture Improvements

### Project Structure Refactoring ✅ COMPLETED
**Resolved Issues:**
- ✅ **Duplicate CSV/downloads folders** - Cleaned up, only root directories remain, proper .gitignore
- ✅ **Hardcoded paths scattered across codebase** - Centralized in config package
- ✅ **Centralized configuration** - Environment variables properly managed via internal/config
- ✅ **Standalone CLI scripts at root** - Kept for backward compatibility, new structure in cmd/api
- ✅ **Mixed concerns** - Handlers now use dependency injection with config
- ✅ **Proper logging** - Structured logging with zerolog in pkg/logger
- ⚠️ **No database** - All state in memory/files, jobs lost on restart (STILL TODO)

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
- [x] Create new directory structure
- [x] Extract configuration management into `internal/config`
- [x] Centralize path resolution (stop hardcoding ./csv, ./downloads)
- [x] Move API code to `cmd/api` and `internal/api`
- [ ] Refactor standalone scripts into `cmd/cli` (kept at root for now)
- [x] Move business logic to `internal/` packages (scraper, downloader, storage, jobs)
- [x] Add structured logging with zerolog
- [ ] Add SQLite database for jobs and metadata
- [x] Update Dockerfiles to use new structure (verified working)
- [x] Update all imports and tests (210 tests passing)
- [x] Update documentation (README, DEVELOPMENT.md, copilot-instructions.md)

### Configuration Management
- [x] **Create config package** - Load from env vars with defaults ✅
  ```go
  type Config struct {
    DataDir      string  // Base data directory
    CSVDir       string  // Computed: DataDir/csv
    DownloadsDir string  // Computed: DataDir/downloads
    NASPath      string
    Port         int
    LogLevel     string
    Environment  string
  }
  ```
- [x] **Environment validation** - Config validation on Load() ✅
- [ ] **Hot reload config** - Watch .env for changes without restart

###x] **Extract handlers logic** - Business logic in internal/ packages ✅
- [x] **Repository pattern** - File operations in internal/storage ✅
- [x] **Dependency injection** - Handlers struct with config dependency ✅
- [ ] **Consistent error handling** - Custom error types with context
- [x] **Remove code duplication** - Centralized path resolution in config ✅
- [ ] **Remove code duplication** - DRY principle for path resolution logic

### Data Management
- [ ] **SQLite database** - Persist jobs, track metadata, scan results
  ```sql
  CREATE TABLE jobs (id TEXT, type TEXT, status TEXT, progress INT, created_at DATETIME);
  CREATE TABLE tracks (id TEXT, filename TEXT, title TEXT, artist TEXT, metadata JSON);
  ```
- [ ] **Database migrations** - Version control for schema changes
- [ ] **Backup/restore** - Regular database backups to NAS

###x] **Structured logging** - Using zerolog in pkg/logger ✅
- [x] **Log levels** - DEBUG, INFO, WARN, ERROR with proper filtering ✅
- [ ] **Log levels** - DEBUG, INFO, WARN, ERROR with proper filtering
- [ ] **Request logging** - HTTP request/response with timing
- [ ] **Performance metrics** - Track scrape/download speeds, success rates
- [ ] **Health checks** - More comprehensive health endpoint

### Testing Improvements
- [ ] **Test fixtures** - Reusable test data in testdata/ directory
- [ ] **Integration tests** - Test full workflows end-to-end
- [x] **Table-driven tests** - Consistent test patterns ✅
- [ ] **Coverage goal: 80%+** - Currently at ~60% (210 tests passing)terns
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

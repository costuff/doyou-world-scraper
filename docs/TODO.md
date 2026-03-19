# Project TODO List

This file tracks major tasks and features to complete the doyou-world-scraper project and expand it into a full-featured system with a backend, frontend, and advanced music management tools.

## High-Level Goals
- [ ] Build a robust backend (Go/Python) for scraping, downloading, deduplication, and metadata management
- [ ] Create a modern frontend (web UI) for control and monitoring
- [ ] Integrate advanced music metadata extraction and genre classification

## TODOs

### Remaining Steps (Immediate)
- [ ] Raise backend coverage from ~62% to 80%+ (focus: `internal/api`, `internal/jobs`, `internal/storage`, `internal/scraper`)
- [ ] Create frontend test baseline with Vitest + RTL (currently 0% coverage)
- [ ] Fix `golangci-lint` config for current version and re-enable strict lint checks in CI
- [ ] Resolve `errcheck` violations in active code paths
- [ ] Add download failure statistics + log visibility (track and expose which songs failed to download per job/run)
- [ ] Keep dual-repo startup docs/tasks aligned with actual local workflow and ports

### Status Refresh (2026-03-16)
- [x] Analytics service split is completed and running in separate repo (`doyou-music-analytics`)
- [x] Scraper dashboard is integrated with analytics stats endpoint
- [x] Both stacks running on Mac Mini (scraper no longer on Raspberry Pi)
- [x] Watchtower auth fixed, all containers auto-updating from GHCR
- [x] Analytics proxy working for remote users via `/analytics-api/` nginx block
- [ ] Raise backend + frontend test coverage toward 80% target
- [ ] Finalize linting/errcheck cleanup and enforce in CI
- [ ] Keep dual-repo local development workflow documented and current
- [ ] **Consolidate deployment** — create root-level `~/srv/docker-compose.prod.yml` using Compose `include:` to wire both stacks together (single `docker compose up`, shared network, single watchtower instance)

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
- [x] NAS path detection tests - Comprehensive tests for Docker/local deployment scenarios (19 tests)
- [x] **✅ FIX CHROME CRASHES (COMPLETED)** - Memory leaks causing Chrome to crash during scraping
  - **Phase 1 (v1.1.0)**: Memory management and timeouts ✅
    - ✅ Fix 1: Create new context per date (not per month/year) for immediate cleanup
    - ✅ Fix 2: Add 30s timeouts to all chromedp operations (prevent hanging contexts)
    - ✅ Fix 3: Add Chrome memory limit flags (disable-images, disk-cache-size=1, etc.)
  - **Phase 2 (v1.1.1)**: Concurrency limits and cancellation ✅
    - ✅ Fix 4: Limit concurrent scraping jobs with semaphore (max 2 Chrome instances)
    - ✅ Fix 5: Improve job cancellation to force immediate Chrome cleanup
  - **Result**: Stable memory usage (~200-500MB), no crashes, responsive cancellation
  - **Tests**: 15 new tests (scraper_resource_test.go, scraper_concurrency_test.go, handlers_concurrency_test.go)
  - **See**: docs/CHROME_CRASH_FIX.md for detailed implementation
- [x] **NAS Path Resolution (v1.1.1)** - Auto-detect NAS paths, better error messages ✅
  - resolveDownloadsDir() helper with Docker/local/NAS support
  - Auto-detect common mount points on macOS/Linux
  - 10 new tests in handlers_nas_resolve_test.go
- [x] **Cleaned Docker Dev Stack (v1.5.7)** - Replaced obsolete `isponsorblocktv`-only `docker-compose.yml` with API + frontend local development stack ✅
- [x] **Pre-Push Checklist Added (v1.5.7)** - Added `docs/PREPUSH_CHECKLIST.md` and linked from docs for consistent local verification before pipeline ✅
- [ ] **Backend Test Coverage to 80%+ (PRIORITY 2)** - Currently at 62% average (249 test cases)
  - internal/api: 51.8% → 80%+ (add handler tests for error cases, edge cases, WebSocket scenarios)
  - internal/jobs: 53.7% → 80%+ (add job cleanup, cancellation, failure scenario tests)
  - internal/storage: 53.2% → 80%+ (add utility function tests for file operations, NAS operations)
  - internal/scraper: 60.3% → 80%+ (add scraper tests for malformed HTML, network failures, date edge cases)
- [ ] **Fix golangci-lint configuration** - .golangci.yml has invalid 'skip-dirs' property for v1.64, need to update config or find alternative to exclude legacy scripts
- [ ] **Fix all errcheck linting violations** - Add proper error checking throughout codebase (currently disabled in CI)
- [x] Document development workflow for faster iteration without Docker rebuilds

### Frontend
- [x] Build a web dashboard (React + TypeScript + Vite + Tailwind CSS)
- [x] UI to trigger scraping, downloads, deduplication, and metadata scan
- [x] Show progress, logs, and results for each operation
- [x] Real-time job monitoring with WebSocket integration
- [x] Statistics dashboard (track counts, storage usage, date ranges)
- [x] Fix frontend not updating properly - Optimized stats refresh (30s→5s), removed expensive JSON comparisons, added React.memo
- [x] **Version footer** - Add footer component showing deployed backend and frontend versions for version tracking
- [x] **✅ ANALYTICS INTEGRATION (v1.4.0)** - Connect scraper dashboard to analytics service
  - Display real-time analytics statistics in StatsPanel
  - Total tracks, completed, pending, analyzing, failed counts
  - Processing rate (tracks/hour) and completion percentage
  - Color-coded status indicators (green/yellow/cyan/red)
  - Auto-refresh every 5 seconds
  - Graceful degradation if analytics service unavailable
  - See docs/ANALYTICS_INTEGRATION.md for setup
- [ ] **Frontend Test Suite (PRIORITY 3)** - Install Vitest + React Testing Library + jsdom, add comprehensive tests for all components, hooks, and services (currently 0% coverage)
- [ ] **Music exploration graph (PRIORITY 4)** - Energy vs Danceability scatter plot, BPM histogram, key/mode distribution, interactive filtering by audio features; data sourced from analytics API `/api/v1/tracks`
  - Can start once analytics v0.6.0 is deployed and tracks are re-analyzed (mode + raw features available)
  - `key` and `mode` fields now returned in `GET /api/v1/tracks` response
- [ ] **Emotional graph explorer (PRIORITY 5)** — Force-directed graph (D3.js / react-force-graph) where each node = a song, edges = similarity score from analytics `track_similarity` table
  - **Blocked on**: analytics emotion layer (migration 004 + `track_emotions` table) and similarity graph (migration 005 + `track_similarity` table)
  - Start frontend scaffolding (install `react-force-graph`, stub component, mock data) while analytics endpoints are being built
  - Node color = dominant emotion; node size = confidence; edge thickness = similarity score (threshold 0.7)
  - Clicking a node plays the song in the existing audio player bar
  - Dragging toward emotion "pole" anchor nodes filters/pulls visible songs
  - The graph is infinite — you drift from emotion to emotion by following edges
  - Emotion dimensions: `tension`, `melancholy`, `euphoria`, `aggression`, `serenity`, `nostalgia`, `sensuality`, `power`
  - Sidebar on node select: track name, dominant emotion, feature breakdown, play button
  - Requires analytics endpoints: `GET /api/v1/tracks/:id/similar` + `GET /api/v1/tracks/:id/emotions`
- [ ] **UI overhaul** - Dark theme polish, animated chart transitions, improved visual hierarchy, better mobile layout

### Music Metadata & Analysis (Basic)
- [x] **FIX AUDIO ANALYSIS JSON PARSING** - Fixed librosa warnings and Python path detection
  - Issue: librosa FutureWarning going to stdout, breaking JSON parsing
  - Solution: Suppressed warnings, redirected progress to stderr, added --quiet flag
  - Result: Audio analysis returns valid JSON with all features (BPM, key, energy, etc.)
- [x] **High-level Audio Features** - Compute Spotify-like features from low-level audio data
  - Energy, Danceability, Valence, Acousticness, Instrumentalness, Speechiness, Liveness, Melodicness
  - All normalized to 0-1 range matching frontend expectations
- [x] **Single Track Analysis** - API endpoint for analyzing individual tracks (for preview/testing)
  - POST /api/v1/tracks/analyze - Analyze single track, save companion JSON
  - GET /api/v1/tracks/analyzed - List tracks with analysis data
  - Basic visualization in frontend Analytics tab
- [ ] **Current Repo Scope** - Keep this repo lightweight for Raspberry Pi deployment
  - Focus: Web scraping, track downloading, file management, basic metadata
  - NOT included: Batch audio analysis (CPU-intensive, 9,875 tracks)
  - See "Separate Analytics Project" section below for heavy computation

### Separate Analytics Project (NEW - Deploy on Powerful Machine)
**Rationale**: Audio analysis with librosa is CPU-intensive (9,875 tracks). Raspberry Pi cannot handle batch processing. Split into two services:
- **This repo (doyou-world-scraper)**: Lightweight scraper/downloader on Raspberry Pi
- **New repo (doyou-music-analytics)**: Heavy audio processing on powerful machine (desktop/server)

#### Architecture Options:

**Option A: Shared Network Storage (Simplest)**
- [ ] Scraper saves tracks to NAS (already implemented)
- [ ] Analytics service mounts same NAS, reads tracks directly
- [ ] Analytics writes companion JSON files next to MP3s
- [ ] Scraper frontend reads JSON files via NFS/SMB for visualization
- **Pros**: No API needed, file-based communication, simple
- **Cons**: Requires shared filesystem, tight coupling

**Option B: REST API Communication**
- [x] **✅ Create doyou-music-analytics repository (v0.6.0 COMPLETE)** - Separate repo for audio analysis
- [x] **✅ Phase 1: Shared PostgreSQL (DEPLOYED)** - Both services connect to same DB
  - Scraper writes: track metadata (file_path, file_name, download_date, title, artist)
  - Analytics reads: tracks WHERE analyzed_at IS NULL
  - Analytics writes: audio features (bpm, key, energy, danceability, etc.)
  - Schema: `tracks` table with columns for basic info + audio features
  - Analytics accesses audio files via NAS mount
  - **Status**: 3,752 tracks imported, 135 completed, analysis workers running
- [x] **✅ Analytics Service Dashboard Integration (v1.4.0)** - Display analytics in scraper UI
  - Real-time stats display in StatsPanel component
  - Processing rate, completion %, status breakdown
  - Auto-refresh every 5 seconds
**Option C: Shared PostgreSQL Database (Recommended, Active)**
- [x] **Create doyou-music-analytics repository** - Separate repo for audio analysis
- [x] **Phase 1: Shared PostgreSQL** - Both services connect to same DB
  - Scraper writes: track metadata (file_path, file_name, download_date, title, artist)
  - Analytics reads: tracks WHERE analyzed_at IS NULL
  - Analytics writes: audio features (bpm, key, energy, danceability, etc.)
  - Schema: `tracks` table with columns for basic info + audio features
  - Analytics accesses audio files via NAS mount
- [ ] **Phase 2: Event-Driven with Redis** (Future enhancement)
  - Scraper publishes: `track_downloaded` event to Redis pub/sub
  - Analytics subscribes: processes new tracks automatically
  - Async job queue with retries for failed analyses
  - Real-time progress updates via WebSocket
- [ ] **Analytics Service Features**:
  - Background worker processes entire library (9,875 tracks)
  - Batch processing with configurable parallelism
  - Progress tracking: X/9875 tracks analyzed
  - API endpoints: GET /api/tracks/analyzed, GET /api/analysis/stats
  - Web UI for monitoring analysis progress
- [ ] **Visualization Dashboard** (in Analytics repo):
  - Energy vs Danceability scatter plots
  - BPM distribution histogram, Key/mode distribution
  - Timeline visualizations (releases over time, genre trends)
  - Interactive filtering by audio features
  - Spectrograms, radar charts, heatmaps
  - Music recommendation system (similar tracks)
  - Playlist generation by mood/energy/genre
- [ ] **CV Keywords**: Microservices, PostgreSQL, Redis pub/sub, event-driven architecture, batch processing, file storage patterns, distributed systems

## Next Steps Priority Order

### Current Repo (doyou-world-scraper) - Raspberry Pi Deployment
1. **✅ FIX CHROME CRASHES (COMPLETED)** - Memory leaks fixed in v1.1.1
2. **🧪 Comprehensive Test Coverage (PRIORITY 1)** - Increase from 62% to 80%+
   - Backend: Add tests for internal/api (51.8%), internal/jobs (53.7%), internal/storage (53.2%), internal/scraper (60.3%)
   - Frontend: Set up Vitest + React Testing Library (currently 0% coverage)
   - Integration tests: WebSocket, API endpoints, Docker scenarios
   - Error scenarios: Network failures, invalid inputs, edge cases
3. **🧹 Code Quality (PRIORITY 2)** - Fix linting and error handling
   - Fix golangci-lint configuration (invalid skip-dirs for v1.64)
   - Fix all errcheck violations (proper error handling throughout)
  - Keep development workflow docs aligned with actual ports/commands
4. **📉 Download failure observability (PRIORITY 3)**
  - Add per-run failed download summary (count + song identifiers)
  - Persist failed downloads to a log/artifact for later inspection
  - Expose failed-download list in API and/or UI job details
5. **📊 Basic Visualization (PRIORITY 4)** - Simple charts for analyzed tracks
   - Bar charts for analyzed tracks count by date
   - Table view with sortable columns (BPM, Energy, Key)
   - Keep lightweight for Raspberry Pi

### Analytics Repo (doyou-music-analytics) - Current Priorities
1. **🔧 Fix feature extraction algorithm (PRIORITY 1)**
  - Audit and retune `scripts/analyze_audio.py` high-level formulas (energy, danceability, valence, etc.)
  - These are hand-crafted approximations from librosa low-level output — quality needs validation
2. **🧪 Coverage expansion (PRIORITY 2)**
  - Focus packages: `internal/analyzer`, `internal/api`, `internal/db`
  - Add more failure-mode and integration tests
3. **📊 Visualization dashboard (PRIORITY 3)**
  - Scatter plots, BPM histograms, key distribution, timeline, feature-based filtering
4. **📤 Export endpoints (PRIORITY 4)**
  - CSV/JSON export for analyzed tracks with filters
5. **🔁 Bulk re-analysis controls (PRIORITY 5)**
  - Re-run failed/outdated analyses with safe trigger semantics
6. **📈 Better observability (PRIORITY 6)**
  - Improve worker progress/error metrics and operator runbook
7. **🔄 Event-driven enhancements (PRIORITY 7, future)**
  - Redis pub/sub after current API + worker priorities are complete

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
- [ ] **Authentication (personal use → multi-user)**
  - Short term: add basic auth or a simple token gate to the dashboard (currently open to anyone on the tunnel)
  - Long term: proper user accounts so others can have their own library (scrape history, downloads, NAS folder per user)
  - Candidate: Cloudflare Access (zero-config, sits in front of the tunnel) for the quick win, then migrate to in-app auth when multi-user is needed

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

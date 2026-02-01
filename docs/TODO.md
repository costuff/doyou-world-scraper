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
- [ ] **Fix download progress tracking** - Progress only updates per CSV file, not per song (needs fine-grained tracking across all songs)
- [ ] **Increase test coverage to 80%+** - Add comprehensive unit and integration tests
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

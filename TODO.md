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
- [ ] Refactor backend as a REST API (Python FastAPI or Go Fiber)
- [ ] Add endpoints for triggering downloads, deduplication, and metadata extraction
- [ ] Implement job/status tracking for long-running tasks

### Frontend
- [ ] Build a web dashboard (React or similar)
- [ ] UI to trigger scraping, downloads, deduplication, and metadata scan
- [ ] Show progress, logs, and results for each operation
- [ ] Browse/downloaded tracks, search/filter by metadata/genre

### Music Metadata & Categorization
- [ ] Integrate a tool/library to extract audio metadata (artist, title, album, year, genre, etc.)
- [ ] Use a music classification library or ML model to auto-categorize tracks by genre/style
- [ ] Store metadata and categorization results in a database (SQLite/Postgres)
- [ ] Enable manual correction/override of metadata and genre via the frontend

### Infrastructure & Automation
- [ ] Add Docker support for easy deployment (including ARM/Raspberry Pi compatibility)
- [ ] Make the project deployable on Raspberry Pi/CasaOS
- [ ] Expose the web dashboard on a configurable port for remote access to the music library
- [ ] Set up scheduled jobs for scraping and downloads
- [ ] Add authentication/user management for the dashboard

---
Feel free to add, edit, or check off items as the project evolves!

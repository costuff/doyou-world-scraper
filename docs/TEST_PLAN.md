# Test Coverage Improvement Plan

## Goal: Increase test coverage from 61.5% to 80%+

## Current Status (229 test cases, 61.5% average)
- ✅ cmd/api: 1 test, 0.0% (main not testable)
- 🟡 internal/api: 40 tests, 51.8% → **TARGET: 80%**
- ✅ internal/config: 18 tests, 96.4% (EXCELLENT)
- 🟡 internal/downloader: 79 tests, 69.7% → **TARGET: 80%**
- 🟡 internal/jobs: 13 tests, 53.7% → **TARGET: 80%**
- 🟡 internal/scraper: 3 tests, 60.3% → **TARGET: 80%**
- 🟡 internal/storage: 62 tests, 53.2% → **TARGET: 80%**
- ✅ pkg/logger: 13 tests, 74.1% (CLOSE - add a few more edge cases)
- ❌ frontend: 0 tests, 0% → **TARGET: 60%+**

## Backend Test Priorities

### 1. internal/api (51.8% → 80%) - PRIORITY 1
**Missing Test Scenarios:**
- [ ] Error handling: invalid JSON, malformed requests, missing required fields
- [ ] Edge cases: empty strings, special characters in paths, null values
- [ ] HandleAnalyzeTrack: test with missing Python, invalid audio files
- [ ] WebSocket connection tests: connect, disconnect, reconnect scenarios
- [ ] Concurrent request handling: multiple simultaneous API calls
- [ ] Request timeouts and cancellations
- [ ] File upload/download edge cases: large files, corrupted files
- [ ] NAS path validation: more edge cases (symlinks, permissions, disk full)

**Estimated Coverage Gain:** +28% (51.8% → 80%)

### 2. internal/jobs (53.7% → 80%) - PRIORITY 2
**Missing Test Scenarios:**
- [ ] Job lifecycle edge cases: double complete, double fail, complete after cancel
- [ ] Concurrent job operations: multiple goroutines updating same job
- [ ] Job cleanup: test CleanupOldJobs with various time ranges
- [ ] Job progress: test UpdateProgress with invalid percentages (negative, >100)
- [ ] Job cancellation scenarios: cancel before start, cancel after complete
- [ ] Memory leaks: stress test with thousands of jobs
- [ ] Job status transitions: all possible state changes
- [ ] Error message handling: very long error messages, special characters

**Estimated Coverage Gain:** +26% (53.7% → 80%)

### 3. internal/storage (53.2% → 80%) - PRIORITY 3
**Missing Test Scenarios:**
- [ ] RemoveDuplicates: test with symbolic links, hard links
- [ ] MoveToNAS: test with insufficient disk space, permission errors
- [ ] File operations: test with read-only files, empty directories
- [ ] GetTracks: test with corrupted MP3 files, non-MP3 files in downloads
- [ ] ClearCSVs/ClearDownloads: test with files in use, locked files
- [ ] Concurrent file operations: multiple threads accessing same files
- [ ] Large file handling: multi-GB files, very long filenames
- [ ] Network mount edge cases: mounted/unmounted NAS during operation

**Estimated Coverage Gain:** +27% (53.2% → 80%)

### 4. internal/scraper (60.3% → 80%) - PRIORITY 4
**Missing Test Scenarios:**
- [ ] Malformed HTML: missing elements, unexpected structure
- [ ] Network failures: timeouts, connection refused, DNS errors
- [ ] Date edge cases: invalid dates, leap years, future dates, year 2038
- [ ] Rate limiting: test chromedp retries and backoff
- [ ] Empty responses: no tracks found, empty page
- [ ] Special characters in track data: emoji, unicode, HTML entities
- [ ] Month/year scraping: test across month boundaries, year wrap-around
- [ ] CSV writing errors: disk full, permission denied

**Estimated Coverage Gain:** +20% (60.3% → 80%)

### 5. internal/downloader (69.7% → 80%) - PRIORITY 5
**Missing Test Scenarios:**
- [ ] yt-dlp failures: command not found, version incompatibility
- [ ] Platform-specific errors: SoundCloud 404, YouTube age restriction
- [ ] Download interruptions: partial downloads, resume capability
- [ ] Metadata extraction failures: missing tags, corrupted files
- [ ] Concurrent downloads: rate limiting, connection pooling
- [ ] Very long URLs, special characters in URLs
- [ ] Network proxy scenarios: proxy timeout, authentication
- [ ] File format conversions: opus→mp3, m4a→mp3 edge cases

**Estimated Coverage Gain:** +10% (69.7% → 80%)

### 6. pkg/logger (74.1% → 80%) - PRIORITY 6
**Missing Test Scenarios:**
- [ ] Log rotation: test file size limits, log file cleanup
- [ ] Concurrent logging: many goroutines logging simultaneously
- [ ] Special characters in log messages: newlines, tabs, unicode
- [ ] Very long log messages: >1MB single message
- [ ] Log level filtering: ensure DEBUG doesn't appear in INFO mode

**Estimated Coverage Gain:** +6% (74.1% → 80%)

## Frontend Test Priorities

### 1. Setup Testing Infrastructure - PRIORITY 1
- [ ] Install dependencies: `npm install --save-dev vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom`
- [ ] Add vitest config to vite.config.ts
- [ ] Add test scripts to package.json: `"test": "vitest"`, `"test:ui": "vitest --ui"`, `"test:coverage": "vitest --coverage"`
- [ ] Create `frontend/src/setupTests.ts` for @testing-library/jest-dom

### 2. Component Tests (TARGET: 60%+)
**Priority Order:**
- [ ] **hooks/useWebSocket.ts** (HIGH - critical for real-time updates)
  - Connect/disconnect/reconnect scenarios
  - Message handling (job updates, errors)
  - Heartbeat/ping-pong
  - Error recovery
- [ ] **services/api.ts** (HIGH - all API calls)
  - Mock axios responses
  - Error handling
  - Timeout scenarios
- [ ] **StatsPanel.tsx** (MEDIUM - complex state)
  - Loading states
  - Auto-refresh behavior
  - Error display
- [ ] **JobsList.tsx** and **JobCard.tsx** (MEDIUM - important UI)
  - Job status colors
  - Progress bars
  - Cancel functionality
- [ ] **ScrapeControls.tsx** and **DownloadControls.tsx** (MEDIUM)
  - Form validation
  - Button states
  - API call integration
-[ ] **ConnectionStatus.tsx** (LOW - simple component)
  - Connected/disconnected states
- [ ] **Footer.tsx** (LOW - simple component)
  - Version display
- [ ] **ErrorBoundary.tsx** (MEDIUM - error handling)
  - Catch and display errors
  - Fallback UI

### 3. Integration Tests
- [ ] App.tsx full render test
- [ ] WebSocket + JobsList integration
- [ ] API error scenarios with UI feedback
- [ ] Dark mode theme consistency

## Common Error Patterns to Test (Based on Previous Bugs)

### Configuration Errors
- [ ] Missing environment variables: PORT, NAS_PATH, etc.
- [ ] Invalid paths: non-existent directories, permission denied
- [ ] Port conflicts: API server can't bind to port
- [ ] YAML parsing: docker-compose.yml syntax errors

### Runtime Errors
- [ ] File not found: missing CSV, missing audio file
- [ ] Network errors: connection timeout, DNS failure
- [ ] External command failures: yt-dlp not installed, chromedp fails
- [ ] Concurrent writes: WebSocket panic, file corruption
- [ ] Memory leaks: job cleanup not running, goroutine leaks

### Deployment Errors
- [ ] Docker vs local path differences: /nas vs NAS_PATH
- [ ] Volume mount failures: NAS not accessible
- [ ] Environment variable differences: dev vs prod
- [ ] Version mismatch: backend/frontend out of sync

## Test Execution Commands

### Backend
```bash
# Run all tests with coverage
go test ./... -cover -v

# Run specific package
go test ./internal/api/... -cover -v

# Generate HTML coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Count test cases
go test -v ./... 2>&1 | grep -c "^=== RUN"
```

### Frontend
```bash
cd frontend

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Run tests in UI mode
npm run test:ui

# Run specific test file
npm test -- ComponentName.test.tsx
```

## Success Criteria

### Phase 1 (Current Focus)
- [ ] Backend coverage: 61.5% → 70%+ (add 40+ test cases)
- [ ] Frontend testing infrastructure setup complete
- [ ] Frontend coverage: 0% → 30%+ (add 20+ test cases)

### Phase 2
- [ ] Backend coverage: 70% → 80%+ (add 40+ test cases)
- [ ] Frontend coverage: 30% → 60%+ (add 30+ test cases)
- [ ] All common error patterns tested

### Phase 3 (Stretch Goal)
- [ ] Backend coverage: 85%+
- [ ] Frontend coverage: 70%+
- [ ] Integration tests: API + Frontend end-to-end
- [ ] Performance tests: load testing, stress testing

## Timeline Estimate
- **Phase 1**: 2-3 days (backend improvements + frontend setup)
- **Phase 2**: 3-4 days (reach 80% backend, 60% frontend)
- **Phase 3**: 2-3 days (stretch goals)

**Total**: ~7-10 days for comprehensive test coverage

## Tracking Progress
Update this file and `.github/copilot-instructions.md` after each batch of tests:
1. Run tests: `go test -cover ./...`
2. Count cases: `go test -v ./... 2>&1 | grep -c "^=== RUN"`
3. Update stats in both files
4. Commit with message format: `test: add [category] tests ([X] new tests, [Y]% coverage)`

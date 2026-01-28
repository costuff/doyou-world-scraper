# DoYou World Radio Scraper API

REST API for the DoYou World Radio Scraper project.

## Running the API

```bash
cd api
go run .
```

The API will start on port 3000 by default. You can change this by setting the `PORT` environment variable:

```bash
PORT=8080 go run .
```

## API Endpoints

### Health Check
- `GET /api/v1/health` - Check if API is running

### Scraping
- `POST /api/v1/scrape/date` - Scrape tracks for a specific date
  ```json
  {
    "date": "2026-01-27"  // or "today"
  }
  ```

- `POST /api/v1/scrape/month` - Scrape all days in a month
  ```json
  {
    "month": "2026-01"
  }
  ```

- `POST /api/v1/scrape/year` - Scrape all days in a year
  ```json
  {
    "year": "2025"
  }
  ```

### Downloads
- `POST /api/v1/download/csv` - Download tracks from a specific CSV
  ```json
  {
    "csv_path": "out/tracks-2026-01-27.csv",
    "column": 5
  }
  ```

- `POST /api/v1/download/all` - Download tracks from all CSVs in out/ folder
  ```json
  {}
  ```

### Utilities
- `POST /api/v1/util/deduplicate` - Remove duplicate tracks
  ```json
  {
    "downloads_dir": "downloads",
    "nas_dir": "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"
  }
  ```

- `POST /api/v1/util/match` - Match downloads to CSV files
  ```json
  {}
  ```

- `GET /api/v1/util/stats` - Get statistics about downloads and CSVs

### Jobs
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get status of a specific job

### WebSocket
- `GET /api/v1/ws` - WebSocket endpoint for real-time job updates

## WebSocket Support

Connect to `ws://localhost:3000/api/v1/ws` for real-time job progress updates.

### Message Format
```json
{
  "type": "job_update",
  "data": {
    "id": "job-uuid",
    "type": "scrape_date",
    "status": "running",
    "progress": 50
  }
}
```

### JavaScript Example
```javascript
const ws = new WebSocket('ws://localhost:3000/api/v1/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === 'job_update') {
    console.log('Job update:', data.data);
    // Update UI with progress
  }
};
```

See [websocket-test.html](../websocket-test.html) for a full example.

## Job Status

All long-running operations return a job object:

```json
{
  "job": {
    "id": "uuid",
    "type": "scrape_date",
    "status": "running",  // pending, running, completed, failed
    "progress": 50,
    "created_at": "2026-01-27T10:00:00Z",
    "updated_at": "2026-01-27T10:01:00Z",
    "input": { ... },
    "result": { ... }  // only when completed
  }
}
```

## Example Usage

```bash
# Health check
curl http://localhost:3000/api/v1/health

# Scrape today's tracks
curl -X POST http://localhost:3000/api/v1/scrape/date \
  -H "Content-Type: application/json" \
  -d '{"date": "today"}'

# Check job status
curl http://localhost:3000/api/v1/jobs/JOB_ID

# Download from all CSVs
curl -X POST http://localhost:3000/api/v1/download/all \
  -H "Content-Type: application/json" \
  -d '{}'

# Get stats
curl http://localhost:3000/api/v1/util/stats
```

## Next Steps

- Integrate actual Go script logic (main.go, download_soundcloud.go, etc.)
- Add WebSocket support for real-time progress updates
- Add authentication
- Add database for persistent job history

# Analytics Service Integration

The scraper dashboard now displays real-time analytics from the `doyou-music-analytics` service.

## Setup

### 1. Configure Analytics URL

**Local Development:**
```bash
# frontend/.env
VITE_ANALYTICS_API_URL=http://localhost:8080
```

**Production:**
```bash
# frontend/.env
VITE_ANALYTICS_API_URL=http://macmini.local:8080
# or
VITE_ANALYTICS_API_URL=http://192.168.1.100:8080
```

### 2. Start Analytics Service

The analytics service should be running on your Mac mini or local machine:

```bash
cd ../doyou-music-analytics
docker-compose up -d
```

Verify it's running:
```bash
curl http://localhost:8080/api/v1/stats
```

### 3. View Analytics in Dashboard

The scraper's Stats Panel will automatically fetch and display:

- **Total Tracks**: Number of tracks imported to analytics DB
- **Completed**: Successfully analyzed tracks with BPM, key, energy, etc.
- **Pending**: Waiting for analysis
- **Analyzing**: Currently being processed
- **Failed**: Analysis errors (usually corrupted MP3s)
- **Processing Rate**: Tracks analyzed per hour
- **Average Time**: Seconds per track

## Features

### Real-Time Updates

The dashboard refreshes analytics stats every 5 seconds, showing:
- Analysis progress with color-coded status
- Live processing rate (tracks/hour)
- Completion percentage
- Failed track count for troubleshooting

### Graceful Degradation

If the analytics service is unavailable:
- The dashboard still works normally
- Only scraper stats are shown
- No errors displayed to user
- Analytics section appears when service comes online

## API Endpoints Used

- `GET /api/v1/stats/detailed` - Comprehensive statistics with performance metrics
- `GET /api/v1/stats` - Basic statistics (fallback)

## Troubleshooting

### Analytics Stats Not Showing

1. **Check analytics service is running:**
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

2. **Check environment variable:**
   ```bash
   # In frontend directory
   cat .env
   # Should contain: VITE_ANALYTICS_API_URL=http://localhost:8080
   ```

3. **Restart frontend dev server:**
   ```bash
   cd frontend
   npm run dev
   ```

4. **Check browser console for errors** (F12 → Console)

### CORS Errors

If you see CORS errors, the analytics service is configured to allow requests from:
- `http://localhost:5173` (Vite dev server)
- `http://raspberrypi.local`

To add more origins, update the analytics service config.

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    Browser Dashboard                       │
│  ┌──────────────┐                    ┌─────────────────┐  │
│  │ Scraper      │                    │ Analytics       │  │
│  │ Stats        │                    │ Stats           │  │
│  │ (port 3002)  │                    │ (port 8080)     │  │
│  └──────┬───────┘                    └────────┬────────┘  │
└─────────┼──────────────────────────────────────┼──────────┘
          │                                      │
          ▼                                      ▼
┌─────────────────────┐          ┌──────────────────────────┐
│ Scraper Backend     │          │ Analytics Service        │
│ (Go + Fiber)        │          │ (Go + PostgreSQL)        │
│                     │          │                          │
│ - Scraping          │          │ - Audio Analysis         │
│ - Downloading       │          │ - Feature Extraction     │
│ - File Management   │          │ - Statistics API         │
└─────────────────────┘          └──────────────────────────┘
```

## See Also

- [Analytics Service Documentation](../doyou-music-analytics/docs/SCRAPER_INTEGRATION.md)
- [Analytics API Usage](../doyou-music-analytics/docs/API_USAGE.md)
- [Database Schema](../doyou-music-analytics/docs/DATABASE_USAGE.md)

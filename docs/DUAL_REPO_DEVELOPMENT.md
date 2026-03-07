# Dual-Repo Local Development Workflow

This workflow lets you run and develop both services together on one machine:
- `doyou-world-scraper` (API + frontend)
- `doyou-music-analytics` (API + worker + PostgreSQL)

## Port Map (Local)

- Scraper API: `http://localhost:3002`
- Scraper Frontend: `http://localhost:5173`
- Analytics API: `http://localhost:8080`
- Analytics PostgreSQL: `localhost:5432`

## One-Time Setup

### 1) Scraper repo

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-world-scraper
npm install
cd frontend && npm install
```

Create/update frontend env:

```bash
# /Users/skoutelis/Desktop/workspace/doyou-world-scraper/frontend/.env.local
VITE_ANALYTICS_API_URL=http://localhost:8080
```

### 2) Analytics repo

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
cp .env.example .env
```

Ensure these values are set in `.env`:

```bash
DATABASE_URL=postgres://doyou:doyou_secure_2026_music@localhost:5432/doyou_music?sslmode=disable
API_PORT=8080
CORS_ORIGINS=http://localhost:5173
NAS_MOUNT_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks
```

## Daily Startup (Fast Iteration)

### Terminal 1: Analytics Postgres

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
docker compose up -d postgres
```

### Terminal 2: Analytics worker

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
go run ./cmd/worker
```

### Terminal 3: Analytics API

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
PORT=8080 go run ./cmd/api
```

### Terminal 4: Scraper API

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-world-scraper
PORT=3002 go run ./cmd/api
```

### Terminal 5: Scraper frontend

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-world-scraper/frontend
npm run dev
```

## Smoke Check (2 Minutes)

```bash
# Scraper API health
curl http://localhost:3002/api/v1/health

# Analytics API health
curl http://localhost:8080/health

# Analytics stats consumed by scraper frontend
curl http://localhost:8080/api/v1/stats/detailed
```

Open `http://localhost:5173` and confirm analytics stats render in the dashboard.

## Recommended Iteration Loop

1. Develop locally (no Docker rebuilds for API/frontend code changes).
2. Run targeted tests in the repo you changed.
3. Run quick smoke checks above.
4. Only then run Docker validation before push (`docker compose up -d --build`).

## VS Code Task Shortcut (Scraper)

In the scraper workspace:

- `Start Full Stack (Fast Analytics)` for normal startup
- `Restart Full Stack (Fast Analytics)` when you want a clean reset
- `Start Scraper Only (No Docker)` if analytics is not needed
- `Restart Scraper Only (No Docker)` for clean scraper-only reset

### One-Click Full Stack (Recommended)

Use VS Code task: `Start Full Stack (Fast Analytics)`

This starts:
- analytics postgres
- analytics API
- analytics worker in lightweight local mode (`MAX_WORKERS=2`, `ANALYSIS_TIMEOUT=90s`, `SCAN_INTERVAL=6h`, `ENABLE_CLEANUP=false`)
- scraper API
- scraper frontend

NAS path behavior in local tasks:
- Preferred: mounted music directory with actual `.mp3` files
- Fallback: `${workspaceFolder}/data/downloads` if no mounted path has MP3 files

Use this mode for UI/backend iteration. For full throughput validation before push, run analytics worker with production-like settings (e.g. `MAX_WORKERS=6`, cleanup enabled).

## Quick Debug Sequence

```bash
curl http://localhost:3002/api/v1/health
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/stats
```

If all three work, the stack is healthy. If `pending=0`, worker is idle by design.

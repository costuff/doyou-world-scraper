# Development Workflow Guide

This guide explains the best practices for developing new features without rebuilding Docker constantly.

## Quick Start: Local Development (Recommended)

For fast iteration on backend changes, run the API server **locally** instead of in Docker:

```bash
# Terminal 1: Run API locally (new structure)
go run ./cmd/api

# Terminal 2: Run frontend dev server
cd frontend
npm run dev

# Or use the convenience command from project root:
npm run dev
```

**Benefits:**
- ⚡ Instant restarts on code changes
- 🐛 Easy debugging with breakpoints
- 📝 Live reload without rebuilds
- 🔍 Direct log output in terminal

**Prerequisites:**
- Go 1.23+ installed
- Chrome/Chromium installed (for scraping)
- yt-dlp in PATH (for downloads)

## When to Use Docker vs Local

| Use Case | Use Docker | Use Local | Why |
|----------|-----------|-----------|-----|
| Quick backend changes | ❌ | ✅ | No rebuild needed |
| Frontend changes | ❌ | ✅ | Vite HMR works locally |
| Testing multi-arch builds | ✅ | ❌ | Need ARM/AMD64 |
| Testing production config | ✅ | ❌ | Verify container behavior |
| Path/permission issues | ✅ | ❌ | Reproduce Docker environment |
| Deploy to Pi/production | ✅ | ❌ | Need containerized app |

## Development Workflows

### Workflow 1: Adding a New Feature (Backend)

```bash
# 1. Run locally for development
go run ./cmd/api

# 2. Make your changes in VSCode
# API auto-reloads on save (if using air or manual restart)

# 3. Test manually via curl or frontend UI
curl http://localhost:3000/api/v1/your-endpoint

# 4. Write tests
go test ./internal/... ./api/...

# 5. When ready, test in Docker
docker compose build api
docker compose up -d api

# 6. If everything works, commit and push
git add .
git commit -m "Add new feature"
git push  # Auto-deploys to Pi via Watchtower
```

### Workflow 2: Frontend Development

```bash
# Frontend has live reload built-in
cd frontend
npm run dev

# Make changes → browser auto-refreshes
# No need for Docker unless testing production build
```

### Workflow 3: Full Stack Testing

```bash
# Option 1: Both servers locally (fastest)
npm run dev  # Runs both API and frontend

# Option 2: Docker (for production testing)
docker compose up -d
docker compose logs -f api  # Watch logs

# Option 3: Mixed (API in Docker, frontend local)
docker compose up -d api
cd frontend && npm run dev
```

## Hot Reload for Go (Optional)

Install `air` for automatic Go server restarts:

```bash
# Install air
go install github.com/air-verse/air@latest

# Run with hot reload
cd api
air
```

Create `api/.air.toml`:
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go"]
  exclude_dir = ["tmp"]
  delay = 1000
```

## Testing Strategy

### Unit Tests (Fast)
```bash
cd api
go test -v -short ./...  # Run all tests
go test -v -run TestJobCreation  # Run specific test
go test -cover ./...  # Check coverage
```

### Integration Tests (Docker)
```bash
# Start services
docker compose up -d

# Run tests against live containers
curl -X POST http://localhost:3000/api/v1/scrape/date \
  -H "Content-Type: application/json" \
  -d '{"date":"2024-01-01"}'

# Check job status
curl http://localhost:3000/api/v1/jobs
```

### Frontend Tests
```bash
cd frontend
npm run test  # (when tests are added)
npm run lint
npm run type-check
```

## Debugging Tips

### Backend Debugging
```bash
# Run with verbose logging
cd api
go run -v $(ls *.go | grep -v _test.go)

# Use Delve debugger
dlv debug . -- serve
(dlv) break handlers.go:123
(dlv) continue
```

### Check What's Running
```bash
# Local servers
lsof -i :3000  # API
lsof -i :5173  # Frontend dev

# Docker containers
docker ps
docker logs -f doyou-api
docker exec -it doyou-api sh
```

## Common Issues

### Port Already in Use
```bash
# Find and kill process
lsof -ti:3000 | xargs kill -9
```

### Docker Build Cache Issues
```bash
# Force rebuild without cache
docker compose build --no-cache api
```

### NAS Mount Not Working
```bash
# Check mount exists
ls -la /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks

# Or in Docker
docker exec doyou-api ls -la /nas
```

### Path Issues (Docker vs Local)
The code auto-detects environment:
- Docker: Uses `/app/csv`, `/app/downloads`, `/nas`
- Local: Uses `./csv`, `./downloads`, `/Volumes/...`

## Performance Tips

1. **Use local dev for 90% of work** - Only use Docker for final testing
2. **Keep Docker images small** - Multi-stage builds already implemented
3. **Use .dockerignore** - Excludes node_modules, downloads, csv
4. **Parallel testing** - Run frontend and backend tests simultaneously

## Deployment Checklist

Before pushing to production:

- [ ] Tests pass: `go test ./...`
- [ ] Linting clean: `golangci-lint run` (if installed)
- [ ] Frontend builds: `cd frontend && npm run build`
- [ ] Docker builds locally: `docker compose build`
- [ ] Test in Docker: `docker compose up -d`
- [ ] Check health: `curl http://localhost:3000/api/v1/health`
- [ ] Verify features work end-to-end
- [ ] Commit and push → Watchtower deploys automatically

## Environment Variables

### Local Development (.env.local)
```bash
API_PORT=3000
FRONTEND_PORT=5173
NAS_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks
TZ=Europe/Athens
```

### Production on Pi (.env)
```bash
API_PORT=3000
FRONTEND_PORT=3001
NAS_PATH=/mnt/mydrive/ExtHDD-8TB/Music/doyou-world-radio-tracks
TZ=Europe/Athens
```

## Summary

✅ **Do this for fast development:**
- Run API locally: `cd api && go run *.go`
- Run frontend locally: `cd frontend && npm run dev`
- Make changes, test immediately
- Use Docker only for final verification

❌ **Don't do this:**
- Rebuild Docker for every change
- Run both frontend and API in Docker during active development
- Skip tests before committing

🎯 **Golden Rule:** Docker is for deployment and final testing, not active development.

# Pre-Push Checklist

Use this checklist before every push to `main`.

## 1) Validate Local Workspace

- [ ] Confirm clean git state: `git status`
- [ ] Ensure both version files are bumped (`VERSION` and `frontend/VERSION`)
- [ ] Confirm `.env` values are correct (`API_PORT`, `FRONTEND_PORT`, `NAS_PATH`)
  - Note: `VITE_ANALYTICS_API_URL` is baked into the Dockerfile as `/analytics-api` — do **not** set it in `.env`

## 2) Run Backend Quality Gates

- [ ] Run backend tests: `go test ./internal/... ./api/...`
- [ ] Run full tests when touching shared code: `go test ./...`
- [ ] Verify there are no obvious runtime errors in logs during startup

## 3) Run Frontend Quality Gates

- [ ] Install/update dependencies if needed: `cd frontend && npm install`
- [ ] Build frontend: `npm run build`
- [ ] Lint frontend: `npm run lint`

## 4) Smoke Test Locally

- [ ] Start backend locally: `go run ./cmd/api`
- [ ] Start frontend locally: `cd frontend && npm run dev`
- [ ] Verify health endpoint: `curl http://localhost:3002/api/v1/health`
- [ ] Verify dashboard can trigger and display at least one job

## 5) Docker Verification

- [ ] Build and run local containers: `docker compose up -d --build`
- [ ] Check API health in Docker: `curl http://localhost:3000/api/v1/health`
- [ ] Check containers are healthy: `docker compose ps`
- [ ] Stop test stack: `docker compose down`

## 6) Final Push Readiness

- [ ] Update related docs/TODO entries
- [ ] Commit with clear message
- [ ] Push branch and verify GitHub Actions status

## 7) Production Verification (after push)

- [ ] Confirm GHCR images updated (GitHub Actions workflow completes green)
- [ ] Confirm Watchtower pulled new image(s): `docker logs --tail 30 watchtower`
- [ ] Confirm scraper API healthy: `curl https://do-you-world-scraper.costuff.work/api/v1/health`
- [ ] Confirm analytics proxy healthy: `curl https://do-you-world-scraper.costuff.work/analytics-api/api/v1/stats`
- [ ] Confirm frontend loads and analytics stats render in the dashboard

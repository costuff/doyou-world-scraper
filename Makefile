## Local development helpers
## Usage: make dev | make api | make frontend | make stop

# Load .env if it exists
ifneq (,$(wildcard .env))
  include .env
  export
endif

.PHONY: dev api frontend stop test

## Start everything (API + Frontend)
dev:
	@echo "Starting scraper API on :3002 and frontend on :5173..."
	@PORT=3002 go run ./cmd/api & echo $$! > /tmp/scraper-api.pid
	@cd frontend && npm run dev

## API only
api:
	PORT=3002 go run ./cmd/api

## Frontend only
frontend:
	cd frontend && npm run dev

## Kill background API process
stop:
	@kill $$(cat /tmp/scraper-api.pid 2>/dev/null) 2>/dev/null || true
	@echo "Stopped"

## Run Go tests
test:
	go test ./internal/... ./api/... -count=1

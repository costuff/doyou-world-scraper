# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates chromium

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code (new structure: cmd/, internal/, pkg/)
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build the binary from new entry point
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /build/api-server ./cmd/api

# Final stage
FROM alpine:latest

# Install runtime dependencies (including Python for pip)
RUN apk --no-cache add ca-certificates chromium python3 py3-pip tzdata && \
    pip3 install --no-cache-dir --break-system-packages yt-dlp

# Create app user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/api-server .

# Create necessary directories (using new structure)
RUN mkdir -p /app/data/csv /app/data/downloads && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 3000

ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/ \
    PORT=3000 \
    DATA_DIR=/app/data \
    LOG_LEVEL=info \
    ENVIRONMENT=production

CMD ["./api-server"]

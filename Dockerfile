# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    chromium \
    chromium-driver \
    python3 \
    python3-pip \
    ffmpeg \
    wget \
    && rm -rf /var/lib/apt/lists/* \
    && wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp \
    && ln -sf /usr/local/bin/yt-dlp /usr/bin/yt-dlp

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/api .

# Copy VERSION files for runtime version display
COPY VERSION ./VERSION
COPY frontend/VERSION ./frontend/VERSION

# Copy Python analysis scripts
COPY scripts/ ./scripts/

# Install Python dependencies (Debian has pre-built wheels for everything!)
RUN pip3 install --no-cache-dir -r scripts/requirements.txt --break-system-packages

# Create data directories
RUN mkdir -p /app/data/csv /app/data/downloads

# Set environment variables for Chrome
ENV CHROME_BIN=/usr/bin/chromium \
    CHROME_PATH=/usr/lib/chromium/ \
    PYTHONUNBUFFERED=1

# Expose API port
EXPOSE 3000

# Run the API server
CMD ["./api"]

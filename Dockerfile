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
FROM alpine:latest

# Install build dependencies and runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    python3 \
    py3-pip \
    ffmpeg \
    # Build dependencies for Python packages
    gcc \
    g++ \
    musl-dev \
    python3-dev \
    openblas-dev \
    gfortran

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/api .

# Copy Python analysis scripts
COPY scripts/ ./scripts/

# Install Python dependencies
RUN pip3 install --no-cache-dir -r scripts/requirements.txt --break-system-packages

# Remove build dependencies to reduce image size
RUN apk del gcc g++ musl-dev python3-dev gfortran

# Create data directories
RUN mkdir -p /app/data/csv /app/data/downloads

# Set environment variables for Chrome
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/ \
    PYTHONUNBUFFERED=1

# Expose API port
EXPOSE 3000

# Run the API server
CMD ["./api"]

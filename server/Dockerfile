# ==============================================================================
# NEXORA Enterprise Media Server - Multi-Stage Production Dockerfile
# ==============================================================================

# Stage 1: Build binary
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy server source code
COPY . .

# Build statically linked release binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/nexora-server ./cmd/api

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

# Install FFmpeg, FFprobe, timezone data and ca-certificates for metadata fetch
RUN apk add --no-cache ffmpeg ca-certificates tzdata su-exec

# Create unprivileged application user
RUN addgroup -S nexora && adduser -S -G nexora -h /app nexora

# Create directories for media, asset caches, and migrations
RUN mkdir -p /app/assets/images /app/migrations /media && chown -R nexora:nexora /app

# Copy binary from builder
COPY --from=builder /bin/nexora-server /app/nexora-server

# Copy database migration files
COPY migrations/ /app/migrations/

# Expose API port
EXPOSE 8080

# Environment defaults
ENV NEXORA_HTTP_ADDR=:8080 \
    NEXORA_MIGRATIONS_DIR=/app/migrations \
    NEXORA_ASSET_IMAGE_DIR=/app/assets/images

USER nexora:nexora

ENTRYPOINT ["/app/nexora-server"]

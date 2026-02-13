# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first (for layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations for smallest binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s -extldflags '-static'" \
  -o /app/server ./cmd/server

# Stage 2: Runtime (minimal Alpine image)
FROM alpine:3.19

# Install only essential runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget \
  && rm -rf /var/cache/apk/*

WORKDIR /app

# Create non-root user for security (but stay as root for setup)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary from builder
COPY --from=builder /app/server .

# Copy configs and migrations
COPY --chown=appuser:appgroup configs /app/configs
COPY --chown=appuser:appgroup migrations /app/migrations

# Create necessary directories and set permissions
RUN mkdir -p /app/logs /app/data/sqlite && \
  chown -R appuser:appgroup /app/logs /app/data && \
  chmod 755 /app/logs /app/data/sqlite

# Environment defaults
ENV CONFIG_PATH=/app/configs/config.yaml

EXPOSE 8080

# Switch to non-root user
USER appuser

# Environment defaults
ENV CONFIG_PATH=/app/configs/config.yaml

EXPOSE 8080

# Health check (using -O /dev/null to strictly use GET, spider uses HEAD)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["./server"]

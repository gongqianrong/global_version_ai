# ============================================
# Stage 1: Build Stage
# ============================================
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Enable Go toolchain auto-download for dependencies requiring newer versions
ENV GOTOOLCHAIN=auto

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations (GOTOOLCHAIN=auto allows automatic version upgrade)
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath \
    -o gateway ./cmd/gateway

# ============================================
# Stage 2: Runtime Stage
# ============================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget postgresql-client

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gateway /app/gateway

# Copy migration scripts
COPY scripts/migrations /app/migrations

# Copy entrypoint script
COPY docker-entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Set timezone
ENV TZ=Asia/Shanghai

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/gateway"]

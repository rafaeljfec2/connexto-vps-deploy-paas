# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source files
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/api \
    ./cmd/api

# Stage 2: Production
FROM alpine:3.19 AS production

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    docker-cli \
    docker-cli-compose \
    tzdata

# Create non-root user
RUN addgroup -S paasdeploy && adduser -S paasdeploy -G paasdeploy

# Create data directory
RUN mkdir -p /data/apps && chown -R paasdeploy:paasdeploy /data

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api /app/api

# Copy migrations
COPY --from=builder /app/migrations /app/migrations

# Set ownership
RUN chown -R paasdeploy:paasdeploy /app

# Switch to non-root user
USER paasdeploy

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["/app/api"]

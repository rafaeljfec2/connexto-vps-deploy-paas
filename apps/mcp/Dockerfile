FROM golang:1.25.5-alpine AS builder
WORKDIR /build
RUN apk add --no-cache git ca-certificates
COPY apps/mcp/go.mod apps/mcp/go.sum ./
RUN go mod download
COPY apps/mcp/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/flowdeploy-mcp \
    ./cmd/flowdeploy-mcp

FROM alpine:3.21 AS production
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S mcp && \
    adduser -S mcp -G mcp
WORKDIR /app
COPY --from=builder /build/flowdeploy-mcp /app/flowdeploy-mcp
USER mcp
EXPOSE 3001

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:3001/healthz || exit 1

ENTRYPOINT ["/app/flowdeploy-mcp"]
CMD ["serve", "--addr=:3001"]

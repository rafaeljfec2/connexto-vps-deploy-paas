FROM golang:1.24-alpine AS agent-builder
WORKDIR /build
RUN apk add --no-cache git ca-certificates
COPY AGENT_VERSION .
COPY apps/backend/go.mod apps/backend/go.sum ./backend/
COPY apps/backend/gen ./backend/gen
COPY apps/shared/ ./shared/
COPY apps/agent/ ./agent/
WORKDIR /build/agent
RUN go mod download
RUN AGENT_VER=$(cat /build/AGENT_VERSION | tr -d '\n') && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/paasdeploy/agent/internal/agent.Version=${AGENT_VER}" \
    -o /build/agent-binary \
    ./cmd/agent

FROM golang:1.24-alpine AS backend-builder
WORKDIR /build
RUN apk add --no-cache git ca-certificates
COPY AGENT_VERSION .
COPY apps/shared/ ./shared/
COPY apps/backend/go.mod apps/backend/go.sum ./backend/
WORKDIR /build/backend
RUN go mod download
COPY apps/backend/ .
RUN AGENT_VER=$(cat /build/AGENT_VERSION | tr -d '\n') && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/paasdeploy/backend/internal/handler.LatestAgentVersion=${AGENT_VER}" \
    -o /build/api \
    ./cmd/api

FROM alpine:edge AS production
RUN apk add --no-cache \
    ca-certificates \
    git \
    docker-cli \
    docker-cli-buildx \
    docker-cli-compose \
    tzdata && \
    addgroup -S paasdeploy && \
    adduser -S paasdeploy -G paasdeploy && \
    mkdir -p /data/apps /app && \
    chown -R paasdeploy:paasdeploy /data

WORKDIR /app

COPY --from=agent-builder /build/agent-binary /app/agent
COPY --from=backend-builder /build/api /app/api
COPY --from=backend-builder /build/backend/migrations /app/migrations

RUN chown -R paasdeploy:paasdeploy /app

USER paasdeploy

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

CMD ["/app/api"]

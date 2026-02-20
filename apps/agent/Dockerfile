FROM golang:1.24-alpine AS builder

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

FROM scratch
COPY --from=builder /build/agent-binary /agent

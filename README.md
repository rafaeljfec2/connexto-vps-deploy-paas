# FlowDeploy

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react&logoColor=black)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0-3178C6?style=flat&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-4169E1?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-24+-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?style=flat&logo=tailwindcss&logoColor=white)](https://tailwindcss.com/)
[![Vite](https://img.shields.io/badge/Vite-5.0-646CFF?style=flat&logo=vite&logoColor=white)](https://vitejs.dev/)
[![Traefik](https://img.shields.io/badge/Traefik-3.0-24A1C1?style=flat&logo=traefikproxy&logoColor=white)](https://traefik.io/)
[![gRPC](https://img.shields.io/badge/gRPC-Protocol_Buffers-244c5a?style=flat&logo=grpc&logoColor=white)](https://grpc.io/)
[![pnpm](https://img.shields.io/badge/pnpm-9+-F69220?style=flat&logo=pnpm&logoColor=white)](https://pnpm.io/)
[![License](https://img.shields.io/badge/License-Proprietary-red?style=flat)](LICENSE)

A lightweight, self-hosted PaaS (Platform as a Service) for automatic deployments from GitHub repositories with multi-server management via remote agents.

## Overview

FlowDeploy is a deployment platform that automatically deploys applications when changes are pushed to connected GitHub repositories. It supports both local deployments on the backend host and remote deployments on distributed servers via a gRPC agent architecture. It provides a modern, Vercel-like experience for self-hosted environments.

## Features

### Deployment Engine

- **Automatic Deployments**: Push to branch &rarr; automatic deploy via GitHub webhooks
- **Remote Agent Deploys**: Deploy applications to remote servers via gRPC agent communication
- **Real-time Logs**: Watch deployment progress via SSE (Server-Sent Events) with live streaming
- **Rollback Support**: One-click rollback to previous versions with health verification
- **Docker-based**: All applications run in isolated containers
- **Queue Management**: PostgreSQL-backed queue with `SELECT FOR UPDATE SKIP LOCKED`
- **Health Checks**: Automatic health verification with configurable retries, intervals, and rollback on failure
- **Monorepo Support**: Deploy specific applications from monorepo structures using `workdir` configuration

### Multi-Server Management

- **Remote Agents**: Lightweight Go agents installed on remote servers communicate via gRPC with mTLS
- **Agent Auto-Update**: Push binary updates to agents from the control plane
- **Server Registration**: Agents self-register with system info and Docker metadata
- **Heartbeat Monitoring**: Real-time server health and status tracking
- **SSH Provisioning**: Automated agent installation on new servers

### Container Management

- **Full Lifecycle**: Start, stop, restart, remove containers on local and remote servers
- **Live Logs**: Stream container logs in real-time with follow mode
- **Container Stats**: CPU, memory, network, and disk I/O monitoring
- **Interactive Terminal**: WebSocket-based `docker exec` with PTY support
- **Template Catalog**: Deploy pre-configured applications (PostgreSQL, MySQL, Redis, MongoDB, Nginx, RabbitMQ, Grafana, etc.) on local and remote servers

### Docker Resource Management

- **Image Management**: List, remove, and prune Docker images
- **Network Management**: Create, list, and remove Docker networks
- **Volume Management**: Create, list, and remove Docker volumes
- **Automated Cleanup**: Scheduled pruning of unused containers, images, and volumes with cleanup history logging

### Infrastructure

- **Traefik Integration**: Automatic reverse proxy configuration with dynamic routing
- **Let's Encrypt SSL**: Automatic TLS certificate provisioning and renewal
- **Cloudflare DNS**: Optional DNS management integration
- **Domain Management**: Custom domain routing per application

### Platform Features

- **GitHub OAuth**: Authentication via GitHub with role-based access control (admin/user)
- **Notification System**: Configurable alerts via Slack, Discord, and Email for deploy events
- **Audit Logging**: Platform event tracking with webhook payload history
- **Environment Variables**: Encrypted environment variable management per application
- **Light/Dark Theme**: System-aware theme with manual toggle

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Traefik                                │
│                    (Reverse Proxy + TLS)                        │
└─────────┬──────────────────────────────────┬────────────────────┘
          │                                  │
  ┌───────▼───────┐                 ┌────────▼────────┐
  │   Frontend    │                 │     Backend     │
  │  React + Vite │                 │  Go Fiber API   │
  └───────────────┘                 └────────┬────────┘
                                             │
                            ┌────────────────┼────────────────┐
                            │                │                │
                   ┌────────▼────────┐  ┌────▼─────┐  ┌──────▼──────┐
                   │  Deploy Engine  │  │ gRPC     │  │  PostgreSQL │
                   │   (Workers)     │  │ Server   │  │  (Queue +   │
                   └────────┬────────┘  └────┬─────┘  │   Data)     │
                            │                │        └─────────────┘
               ┌────────────┤                │
               │            │         ┌──────▼──────────────────┐
       ┌───────▼───────┐    │         │   Remote Agents (gRPC)  │
       │ Docker Engine │    │         │  ┌───────┐ ┌───────┐   │
       │   (Local)     │    │         │  │Agent 1│ │Agent N│   │
       └───────────────┘    │         │  │+Docker│ │+Docker│   │
                            │         │  └───────┘ └───────┘   │
                   ┌────────▼─────┐   └─────────────────────────┘
                   │   GitHub     │
                   │  (Webhooks)  │
                   └──────────────┘
```

## Tech Stack

| Component        | Technology                       |
| ---------------- | -------------------------------- |
| Frontend         | React 18 + Vite + TypeScript     |
| UI Library       | shadcn/ui + Tailwind CSS         |
| State Management | React Query (TanStack Query)     |
| Backend          | Go 1.24+ (Fiber framework)       |
| Agent            | Go 1.24+ (gRPC server)           |
| RPC              | gRPC + Protocol Buffers (buf)    |
| Database         | PostgreSQL 16+                   |
| Migrations       | golang-migrate                   |
| Containers       | Docker + Docker Compose v2       |
| Reverse Proxy    | Traefik 3.x                      |
| Authentication   | GitHub OAuth + session cookies   |
| Monorepo         | pnpm + Turborepo                 |

## Requirements

- Ubuntu Server 22.04+ (or compatible Linux)
- Docker Engine 24+
- Docker Compose v2
- Go 1.24+
- Node.js 20+ (for frontend build)
- pnpm 9+
- PostgreSQL 16+

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/your-org/flowdeploy.git
cd flowdeploy
```

### 2. Install dependencies

```bash
pnpm install
```

### 3. Configure environment

```bash
cp .env.example .env
# Edit .env with your settings
```

### 4. Start services

```bash
# Start PostgreSQL and Traefik
pnpm docker:up

# Start backend (development)
pnpm backend:dev

# Start frontend (development)
pnpm dev
```

### 5. Access the dashboard

Open `http://localhost:3000` in your browser.

## Project Structure

```
flowdeploy/
├── apps/
│   ├── frontend/              # React dashboard (SPA)
│   ├── backend/               # Go API + Deploy Engine + gRPC server
│   │   ├── cmd/api/           # Application entrypoint
│   │   ├── internal/
│   │   │   ├── agentclient/   # gRPC client for remote agents
│   │   │   ├── config/        # Configuration management
│   │   │   ├── di/            # Dependency injection (Wire)
│   │   │   ├── domain/        # Domain models and interfaces
│   │   │   ├── engine/        # Deploy engine and worker pool
│   │   │   ├── grpcserver/    # Backend gRPC server (agent registration)
│   │   │   ├── handler/       # HTTP handlers (Fiber)
│   │   │   ├── middleware/    # Auth, CORS, tracing middleware
│   │   │   ├── repository/    # PostgreSQL repositories
│   │   │   ├── requestctx/    # Request context utilities
│   │   │   └── service/       # Business logic services
│   │   ├── gen/go/            # Generated protobuf/gRPC code
│   │   └── migrations/        # SQL migration files
│   ├── agent/                 # Remote agent binary
│   │   ├── cmd/agent/         # Agent entrypoint
│   │   └── internal/
│   │       ├── agent/         # Agent core (registration, heartbeat)
│   │       ├── cleanup/       # Docker cleanup scheduler
│   │       ├── deploy/        # Deployment executor
│   │       ├── grpcserver/    # Agent gRPC handlers
│   │       │   ├── handlers_containers.go
│   │       │   ├── handlers_deploy.go
│   │       │   ├── handlers_exec.go
│   │       │   ├── handlers_images.go
│   │       │   ├── handlers_resources.go
│   │       │   └── handlers_update.go
│   │       └── sysinfo/       # System metrics collection
│   ├── shared/                # Shared Go packages
│   │   └── pkg/
│   │       ├── docker/        # Docker CLI client
│   │       ├── executor/      # Shell command executor
│   │       ├── health/        # Health check utilities
│   │       ├── paths/         # Path resolution utilities
│   │       └── version/       # App version detection
│   └── proto/                 # Protocol Buffer definitions
│       └── flowdeploy/v1/
│           ├── agent.proto    # Agent service definition
│           ├── server.proto   # Server/container messages
│           ├── deploy.proto   # Deployment messages
│           └── common.proto   # Shared messages
├── deploy/
│   ├── traefik/               # Traefik configuration
│   └── docker-compose.yml     # Infrastructure services
├── .github/workflows/         # CI/CD pipelines
├── AGENT_VERSION              # Current agent version
├── CHANGELOG.md               # Version history
├── package.json               # Root workspace (v0.2.0)
├── pnpm-workspace.yaml        # pnpm workspace config
└── turbo.json                 # Turborepo config
```

## Configuration

### Application Contract (paasdeploy.json)

Each deployed application must have a `paasdeploy.json` file in its root (or in the directory specified by `workdir` for monorepos):

```json
{
  "name": "my-app",
  "build": {
    "type": "dockerfile",
    "dockerfile": "./Dockerfile",
    "context": "."
  },
  "healthcheck": {
    "path": "/health",
    "interval": "30s",
    "timeout": "5s"
  },
  "port": 8080,
  "env": {
    "NODE_ENV": "production"
  },
  "resources": {
    "memory": "512m",
    "cpu": "0.5"
  }
}
```

### Monorepo Configuration

For monorepo projects, specify the `workdir` when creating an application to point to the subdirectory containing `paasdeploy.json` and `docker-compose.yml`:

```json
{
  "name": "my-monorepo-app",
  "repository_url": "https://github.com/org/monorepo.git",
  "branch": "main",
  "workdir": "apps/backend"
}
```

## Environment Variables

| Variable          | Description                              | Default                       |
| ----------------- | ---------------------------------------- | ----------------------------- |
| `DATABASE_URL`    | PostgreSQL connection string             | -                             |
| `PORT`            | Backend API port                         | `8080`                        |
| `DEPLOY_DATA_DIR` | Directory for cloned repositories        | `/data/apps`                  |
| `DOCKER_HOST`     | Docker daemon socket                     | `unix:///var/run/docker.sock` |
| `LOG_LEVEL`       | Logging level (debug, info, warn, error) | `info`                        |
| `CORS_ORIGINS`    | Allowed CORS origins                     | -                             |
| `GRPC_PORT`       | Backend gRPC server port                 | `50051`                       |
| `GRPC_AGENT_PORT` | Agent gRPC server port                   | `50052`                       |
| `GITHUB_CLIENT_ID`     | GitHub OAuth application client ID  | -                             |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth application secret     | -                             |

## API Endpoints

### Applications

| Method | Endpoint                    | Description                  |
| ------ | --------------------------- | ---------------------------- |
| GET    | `/health`                   | Health check                 |
| GET    | `/api/apps`                 | List all applications        |
| POST   | `/api/apps`                 | Register new application     |
| GET    | `/api/apps/:id`             | Get application details      |
| DELETE | `/api/apps/:id`             | Remove application           |
| GET    | `/api/apps/:id/deployments` | List deployments             |
| POST   | `/api/apps/:id/redeploy`    | Trigger manual redeploy      |
| POST   | `/api/apps/:id/rollback`    | Rollback to previous version |
| GET    | `/events/deploys`           | SSE stream for deploy events |

### Containers

| Method | Endpoint                       | Description                     |
| ------ | ------------------------------ | ------------------------------- |
| GET    | `/api/containers`              | List containers (?serverId=)    |
| POST   | `/api/containers`              | Create container                |
| POST   | `/api/containers/:id/start`    | Start container (?serverId=)    |
| POST   | `/api/containers/:id/stop`     | Stop container (?serverId=)     |
| POST   | `/api/containers/:id/restart`  | Restart container (?serverId=)  |
| DELETE | `/api/containers/:id`          | Remove container (?serverId=)   |
| GET    | `/api/containers/:id/logs`     | Stream container logs (SSE)     |

### Templates

| Method | Endpoint                            | Description                        |
| ------ | ----------------------------------- | ---------------------------------- |
| GET    | `/api/templates`                    | List available templates           |
| GET    | `/api/templates/:id`                | Get template details               |
| POST   | `/api/templates/:id/deploy`         | Deploy template (?serverId=)       |

### Infrastructure

| Method | Endpoint                       | Description                     |
| ------ | ------------------------------ | ------------------------------- |
| GET    | `/api/images`                  | List Docker images (?serverId=) |
| DELETE | `/api/images/:id`              | Remove image (?serverId=)       |
| GET    | `/api/networks`                | List networks (?serverId=)      |
| GET    | `/api/volumes`                 | List volumes (?serverId=)       |
| GET    | `/api/servers`                 | List registered servers         |
| GET    | `/api/certificates`            | List TLS certificates           |

## Development

### Frontend

```bash
cd apps/frontend
pnpm dev
```

### Backend

```bash
cd apps/backend
go run cmd/api/main.go
```

### Agent

```bash
cd apps/agent
go run cmd/agent/main.go --server-addr=localhost:50051 --server-id=<id> --agent-port=50052
```

### Protobuf Generation

```bash
cd apps/proto
buf generate
```

### Running Tests

```bash
# Frontend tests
cd apps/frontend && pnpm test

# Backend tests
cd apps/backend && go test ./...
```

## Deployment

### Production Build

```bash
# Build all
pnpm build

# Build backend binary
pnpm backend:build

# Build agent binary
cd apps/agent && go build -ldflags "-X github.com/paasdeploy/agent/internal/agent.Version=$(cat ../../AGENT_VERSION)" -o bin/agent cmd/agent/main.go
```

### Docker Deployment

```bash
docker compose -f deploy/docker-compose.yml up -d
```

## Security

- mTLS authentication between backend and agents
- GitHub OAuth for user authentication
- Role-based access control (admin/user)
- Encrypted environment variable storage
- All shell commands use explicit arguments (no `sh -c`)
- CORS restricted to explicit origins (no wildcard fallback)
- Input validation and sanitization on all endpoints
- Alpine-based Docker images for minimal attack surface

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and recent changes.

## License

Proprietary - see LICENSE file for details.

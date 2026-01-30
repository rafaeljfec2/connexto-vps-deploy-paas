# PaaSDeploy

A lightweight, self-hosted PaaS (Platform as a Service) for automatic deployments from GitHub repositories.

## Overview

PaaSDeploy is a local deployment platform that automatically deploys applications when changes are pushed to the `main` branch of connected GitHub repositories. It provides a simple, Vercel-like experience for self-hosted environments.

## Features

- **Automatic Deployments**: Push to `main` → automatic deploy
- **Real-time Logs**: Watch deployment progress via SSE (Server-Sent Events)
- **Rollback Support**: One-click rollback to previous versions
- **Docker-based**: All applications run in isolated containers
- **Queue Management**: One deploy per application at a time
- **Health Checks**: Automatic health verification with rollback on failure

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Traefik                               │
│                    (Reverse Proxy)                           │
└──────────────┬────────────────────────────┬─────────────────┘
               │                            │
       ┌───────▼───────┐           ┌────────▼────────┐
       │   Frontend    │           │     Backend     │
       │  React + Vite │           │    Go API       │
       └───────────────┘           └────────┬────────┘
                                            │
                                   ┌────────▼────────┐
                                   │  Deploy Engine  │
                                   │   (Workers)     │
                                   └────────┬────────┘
                                            │
               ┌────────────────────────────┼────────────────┐
               │                            │                │
       ┌───────▼───────┐           ┌────────▼────────┐      │
       │  PostgreSQL   │           │  Docker Engine  │      │
       │   (Queue)     │           │  (Containers)   │      │
       └───────────────┘           └─────────────────┘      │
```

## Tech Stack

| Component        | Technology                   |
| ---------------- | ---------------------------- |
| Frontend         | React 18 + Vite + TypeScript |
| UI Library       | shadcn/ui + Tailwind CSS     |
| State Management | React Query                  |
| Backend          | Go 1.23+                     |
| Database         | PostgreSQL                   |
| Containers       | Docker + Docker Compose v2   |
| Reverse Proxy    | Traefik                      |
| Monorepo         | pnpm + Turborepo             |

## Requirements

- Ubuntu Server 22.04 LTS (or compatible Linux)
- Docker Engine 24+
- Docker Compose v2
- Go 1.23+
- Node.js 20+ (for frontend build)
- pnpm 9+
- PostgreSQL 16+

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/your-org/paasdeploy.git
cd paasdeploy
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
paasdeploy/
├── apps/
│   ├── frontend/          # React dashboard
│   └── backend/           # Go API + Deploy Engine
├── deploy/
│   ├── traefik/           # Traefik configuration
│   └── docker-compose.yml # Infrastructure services
├── docs/                  # Documentation
├── package.json           # Root workspace
├── pnpm-workspace.yaml    # pnpm workspace config
└── turbo.json             # Turborepo config
```

## Configuration

### Application Contract (paasdeploy.json)

Each deployed application must have a `paasdeploy.json` file in its root:

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

## Environment Variables

| Variable          | Description                              | Default                       |
| ----------------- | ---------------------------------------- | ----------------------------- |
| `DATABASE_URL`    | PostgreSQL connection string             | -                             |
| `PORT`            | Backend API port                         | `8080`                        |
| `DEPLOY_DATA_DIR` | Directory for cloned repositories        | `/data/apps`                  |
| `DOCKER_HOST`     | Docker daemon socket                     | `unix:///var/run/docker.sock` |
| `LOG_LEVEL`       | Logging level (debug, info, warn, error) | `info`                        |

## API Endpoints

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
```

### Docker Deployment

```bash
docker compose -f deploy/docker-compose.yml up -d
```

## Security Considerations

- Never run the deploy engine as root
- All shell commands use explicit arguments (no `sh -c`)
- CORS is restricted to specific origins
- No sensitive data in frontend bundle
- Health checks prevent broken deployments

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for development guidelines.

## Architecture Details

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for in-depth architecture documentation.

## License

MIT License - see LICENSE file for details.

## Roadmap

- [ ] GitHub Webhook integration
- [ ] User authentication
- [ ] Multi-tenant support
- [ ] SSL/TLS with Let's Encrypt
- [ ] Metrics and monitoring dashboard
- [ ] Slack/Email notifications

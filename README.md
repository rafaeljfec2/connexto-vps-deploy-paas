# FlowDeploy

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react&logoColor=black)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0-3178C6?style=flat&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-4169E1?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-24+-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?style=flat&logo=tailwindcss&logoColor=white)](https://tailwindcss.com/)
[![Vite](https://img.shields.io/badge/Vite-5.0-646CFF?style=flat&logo=vite&logoColor=white)](https://vitejs.dev/)
[![Traefik](https://img.shields.io/badge/Traefik-3.0-24A1C1?style=flat&logo=traefikproxy&logoColor=white)](https://traefik.io/)
[![pnpm](https://img.shields.io/badge/pnpm-9+-F69220?style=flat&logo=pnpm&logoColor=white)](https://pnpm.io/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)

A lightweight, self-hosted PaaS (Platform as a Service) for automatic deployments from GitHub repositories.

## Overview

FlowDeploy is a local deployment platform that automatically deploys applications when changes are pushed to the `main` branch of connected GitHub repositories. It provides a simple, Vercel-like experience for self-hosted environments.

## Features

### Core Features

- **Automatic Deployments**: Push to `main` → automatic deploy
- **Real-time Logs**: Watch deployment progress via SSE (Server-Sent Events)
- **Rollback Support**: One-click rollback to previous versions
- **Docker-based**: All applications run in isolated containers
- **Queue Management**: One deploy per application at a time
- **Health Checks**: Automatic health verification with rollback on failure

### Recent Additions

- **Monorepo Support**: Deploy specific applications from monorepo structures using `workdir` configuration
- **Light/Dark Theme**: System-aware theme with manual toggle (light, dark, system)
- **Database Migrations**: Automatic schema management with golang-migrate
- **Reusable Components**: Modular UI with PageHeader, IconText, FormField, LoadingGrid, ErrorMessage
- **Developer Experience**: VSCode configurations, ESLint, Prettier integration

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

The deploy engine will look for configuration files at `{repo_root}/{workdir}/paasdeploy.json`.

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

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and recent changes.

## License

MIT License - see LICENSE file for details.

## Roadmap

### Completed

- [x] Core deployment pipeline with queue management
- [x] Real-time logs via Server-Sent Events (SSE)
- [x] Rollback support with health check verification
- [x] Docker-based containerization
- [x] PostgreSQL-backed deployment queue
- [x] Monorepo support with `workdir` configuration
- [x] Light/Dark/System theme support
- [x] Database migrations with golang-migrate
- [x] Reusable frontend components
- [x] ESLint + Prettier code quality tools
- [x] VSCode debug and task configurations

### In Progress

- [ ] GitHub Webhook integration for automatic triggers
- [ ] Build logs persistence and search

### Planned

- [ ] User authentication (JWT-based)
- [ ] Multi-tenant support with workspaces
- [ ] SSL/TLS with Let's Encrypt auto-renewal
- [ ] Metrics and monitoring dashboard (Prometheus/Grafana)
- [ ] Slack/Discord/Email notifications
- [ ] Environment variable management UI
- [ ] Deploy previews for pull requests
- [ ] Custom domain routing
- [ ] Horizontal scaling with multiple workers
- [ ] CLI tool for local development

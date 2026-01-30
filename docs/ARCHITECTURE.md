# PaaSDeploy Architecture

This document describes the technical architecture of PaaSDeploy.

## System Overview

PaaSDeploy is a self-hosted deployment platform that orchestrates automatic deployments from GitHub repositories. The system follows a layered architecture with clear separation of concerns.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Client Layer                               │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    React Dashboard                            │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │   │
│  │  │ Dashboard│  │App Detail│  │  Deploy  │  │  New App │    │   │
│  │  │   Page   │  │   Page   │  │  Logs    │  │   Form   │    │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │   │
│  │                                                               │   │
│  │  ┌─────────────────────┐  ┌─────────────────────┐           │   │
│  │  │    React Query      │  │     SSE Client      │           │   │
│  │  │   (State Cache)     │  │   (Real-time)       │           │   │
│  │  └─────────────────────┘  └─────────────────────┘           │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP / SSE
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         API Layer (Go)                               │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                      HTTP Handlers                            │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │   │
│  │  │  Health  │  │   Apps   │  │ Deploys  │  │   SSE    │    │   │
│  │  │ Handler  │  │ Handler  │  │ Handler  │  │ Handler  │    │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                    │                                 │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                      Service Layer                            │   │
│  │  ┌──────────────────────┐  ┌──────────────────────┐         │   │
│  │  │     App Service      │  │   Deploy Service     │         │   │
│  │  └──────────────────────┘  └──────────────────────┘         │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                    │                                 │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Repository Layer                           │   │
│  │  ┌──────────────────────┐  ┌──────────────────────┐         │   │
│  │  │   App Repository     │  │ Deployment Repository│         │   │
│  │  └──────────────────────┘  └──────────────────────┘         │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Deploy Engine (Go)                              │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                        Engine                                 │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │   │
│  │  │Dispatcher│  │  Worker  │  │  Queue   │  │ Notifier │    │   │
│  │  │          │──│  Pool    │──│          │──│  (SSE)   │    │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │   │
│  │                      │                                        │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │   │
│  │  │   Git    │  │  Docker  │  │ Executor │  │  Health  │    │   │
│  │  │  Client  │  │  Client  │  │          │  │ Checker  │    │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
                    ▼               ▼               ▼
            ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
            │  PostgreSQL  │ │    Docker    │ │  File System │
            │   (Queue)    │ │   Engine     │ │   (Repos)    │
            └──────────────┘ └──────────────┘ └──────────────┘
```

## Component Details

### Frontend (React)

The frontend is a single-page application built with React and Vite.

#### Structure

```
src/
├── app/                 # Application bootstrap
│   ├── providers.tsx    # React Query, Router providers
│   ├── routes.tsx       # Route definitions
│   └── layout.tsx       # Main layout
├── pages/               # Route components
├── features/            # Domain-specific logic
│   ├── apps/            # Application management
│   └── deploys/         # Deployment management
├── components/          # Reusable UI components
│   └── ui/              # shadcn/ui components
├── services/            # API communication
├── hooks/               # Global hooks
└── types/               # TypeScript definitions
```

#### Data Flow

1. **Initial Load**: Pages fetch data via React Query
2. **Real-time Updates**: SSE connection receives events
3. **Cache Invalidation**: SSE events update React Query cache
4. **UI Update**: Components re-render automatically

```
Page Load → React Query → REST API → Cache → Render
                                        ↑
SSE Event → Parse → Update Cache ───────┘
```

### Backend (Go)

The backend is a Go application with clear layered architecture.

#### Structure

```
internal/
├── config/              # Configuration management
├── domain/              # Domain entities and errors
├── handler/             # HTTP handlers
├── service/             # Business logic
├── repository/          # Data access
└── engine/              # Deploy engine
```

#### Layers

1. **Handler Layer**: HTTP request/response handling
2. **Service Layer**: Business logic orchestration
3. **Repository Layer**: Database operations
4. **Engine Layer**: Deploy execution (separate concern)

### Deploy Engine

The deploy engine is the core of PaaSDeploy. It runs as a background process within the API server.

#### Components

| Component          | Responsibility                            |
| ------------------ | ----------------------------------------- |
| **Engine**         | Lifecycle management, worker coordination |
| **Dispatcher**     | Selects pending deploys from queue        |
| **Worker**         | Executes deploy pipeline                  |
| **Queue**          | PostgreSQL-backed deploy queue            |
| **Executor**       | Safe OS command execution                 |
| **Git Client**     | Repository operations                     |
| **Docker Client**  | Container operations                      |
| **Health Checker** | Application health verification           |
| **Notifier**       | SSE event emission                        |
| **Locker**         | Per-app deployment locking                |

#### Deploy Pipeline

```
┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐
│ PENDING │──▶│ RUNNING │──▶│  BUILD  │──▶│  DEPLOY │──▶│ HEALTH  │
└─────────┘   └─────────┘   └─────────┘   └─────────┘   └─────────┘
                                                              │
                                          ┌───────────────────┴───────────────────┐
                                          │                                       │
                                          ▼                                       ▼
                                    ┌─────────┐                             ┌─────────┐
                                    │ SUCCESS │                             │ ROLLBACK│
                                    └─────────┘                             └────┬────┘
                                                                                 │
                                                                                 ▼
                                                                           ┌─────────┐
                                                                           │ FAILED  │
                                                                           └─────────┘
```

#### Queue Management

The queue uses PostgreSQL with `SELECT FOR UPDATE SKIP LOCKED` to ensure:

- Only one deploy per application runs at a time
- Multiple workers can process different apps concurrently
- No deploy is processed twice

```sql
SELECT * FROM deployments
WHERE status = 'pending'
AND app_id NOT IN (
    SELECT app_id FROM deployments WHERE status = 'running'
)
ORDER BY created_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
```

## Database Schema

### Entity Relationship

```
┌─────────────────┐       ┌─────────────────────┐
│      apps       │       │    deployments      │
├─────────────────┤       ├─────────────────────┤
│ id (PK)         │──────<│ app_id (FK)         │
│ name            │       │ id (PK)             │
│ repository_url  │       │ commit_sha          │
│ branch          │       │ commit_message      │
│ config (JSONB)  │       │ status              │
│ status          │       │ started_at          │
│ last_deployed_at│       │ finished_at         │
│ created_at      │       │ error_message       │
│ updated_at      │       │ logs                │
└─────────────────┘       │ previous_image_tag  │
                          │ current_image_tag   │
                          │ created_at          │
                          └─────────────────────┘
```

### Indexes

- `idx_deployments_app_id`: Fast lookup by application
- `idx_deployments_status`: Filter by deployment status
- `idx_deployments_pending`: Partial index for queue queries

## Communication Patterns

### REST API

Used for CRUD operations and actions:

- List/Create/Update/Delete applications
- Trigger redeploy/rollback
- Fetch deployment history

### Server-Sent Events (SSE)

Used for real-time updates:

- Deployment status changes
- Build logs streaming
- Application status updates

#### Event Format

```
event: deploy
data: {"type":"RUNNING","deployId":"abc123","appId":"xyz","timestamp":"..."}

event: log
data: {"type":"LOG","deployId":"abc123","message":"Building...","timestamp":"..."}

event: deploy
data: {"type":"SUCCESS","deployId":"abc123","appId":"xyz","timestamp":"..."}
```

## Security Architecture

### Command Execution

All OS commands are executed using `os/exec` with explicit arguments:

```go
// CORRECT: Explicit arguments
cmd := exec.Command("git", "-C", workDir, "fetch", "origin")

// WRONG: Shell injection risk
cmd := exec.Command("sh", "-c", "git -C " + workDir + " fetch origin")
```

### Isolation

- Each application runs in its own Docker container
- Resource limits enforced via Docker
- Network isolation between applications

### Access Control

- CORS restricted to specific origins
- No sensitive data in frontend bundle
- API authentication (future: JWT tokens)

## Scalability Considerations

### Current Design (Single Node)

- Single backend instance
- Worker pool within the same process
- PostgreSQL for queue persistence

### Future Scaling Options

1. **Horizontal API Scaling**
   - Stateless API servers behind load balancer
   - Shared PostgreSQL database
   - SSE via Redis pub/sub

2. **Worker Scaling**
   - Separate worker processes
   - Distributed locking via PostgreSQL
   - Each worker handles specific app groups

## Monitoring and Observability

### Logging

- Structured logging with `slog`
- Log levels: debug, info, warn, error
- Deploy logs stored in database

### Health Checks

- `/health` endpoint for API health
- Application health checks during deploy
- Container health via Docker

### Metrics (Future)

- Deploy duration
- Success/failure rates
- Queue depth
- Resource utilization

## Technology Decisions

### Why Go for Backend?

- Single binary deployment
- Excellent concurrency primitives
- Low resource usage
- Strong standard library

### Why React Query?

- Automatic cache management
- Background refetching
- Optimistic updates
- SSE integration friendly

### Why PostgreSQL?

- `SELECT FOR UPDATE SKIP LOCKED` for queue
- JSONB for flexible config storage
- Proven reliability
- Good tooling ecosystem

### Why SSE over WebSocket?

- Simpler implementation
- Native browser reconnection
- Sufficient for unidirectional updates
- Works with HTTP/2

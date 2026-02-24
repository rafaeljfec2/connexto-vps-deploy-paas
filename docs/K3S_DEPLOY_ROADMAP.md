# K3s Remote Deploy — Feature Roadmap

## Overview

Add K3s as an **optional runtime** for remote deploys. Users choose the runtime when adding a server: Docker (default, current behavior) or K3s. K3s enables rolling updates, replicas, self-healing, and Ingress with automatic TLS.

Docker remains the default and is unchanged. K3s is offered as an advanced option for users who need zero-downtime deploys and horizontal scaling on their VPS.

## Architecture

```
                       ┌──────────────────┐
                       │  Add Server UI   │
                       └────────┬─────────┘
                                │
                         ┌──────▼──────┐
                         │  Runtime?   │
                         └──┬───────┬──┘
                            │       │
                   ┌────────▼──┐ ┌──▼────────┐
                   │  Docker   │ │    K3s     │
                   └────────┬──┘ └──┬────────┘
                            │       │
              ┌─────────────▼──┐ ┌──▼──────────────┐
              │ Docker+Traefik │ │ K3s (single-node)│
              │ + Agent        │ │ + Traefik Ingress│
              └─────────────┬──┘ │ + Agent          │
                            │    └──┬──────────────┘
                            │       │
              ┌─────────────▼──┐ ┌──▼──────────────┐
              │ docker compose │ │ kubectl apply    │
              │ up -d          │ │ Deployment+Svc   │
              └────────────────┘ │ +IngressRoute    │
                                 └─────────────────┘
```

## Why K3s

- **K3s is certified Kubernetes** in a single ~50MB binary
- Minimum ~512MB RAM (fits 1-2GB VPS)
- Actively maintained by SUSE/Rancher, CNCF project
- **Traefik is the default Ingress Controller** (already used by flowDeploy)
- Full k8s ecosystem compatibility (Helm, kubectl, HPA, etc.)
- Rolling updates, replicas, self-healing, rollback — all native

## Why Not Full Kubernetes or Docker Swarm

- **Full Kubernetes**: ~2GB RAM minimum, complex installation, overkill for single-node VPS
- **Docker Swarm**: Development stagnated since ~2019, no new features, risky long-term dependency

---

## Phase 1 — Data Model and Backend (foundation)

**Goal:** Prepare the backend to support two runtimes.

### 1.1 Server Model

Add `Runtime` field to `apps/backend/internal/domain/server.go`:

```go
type ServerRuntime string

const (
    RuntimeDocker ServerRuntime = "docker"
    RuntimeK3s    ServerRuntime = "k3s"
)
```

- New migration: `ALTER TABLE servers ADD COLUMN runtime VARCHAR(20) NOT NULL DEFAULT 'docker'`
- Update repository scan fields, handler API response, and frontend types

### 1.2 Worker Strategy Pattern

Refactor `apps/backend/internal/engine/worker.go` to dispatch by runtime:

```
remote + docker  → runRemoteDeploy (current, unchanged)
remote + k3s     → runK3sDeploy (new)
local            → runLocalDeploy (unchanged)
```

### 1.3 Proto Update

Add `runtime` field to `DeployRequest` in `apps/proto/flowdeploy/v1/deploy.proto` so the agent knows which executor to use.

**Estimated effort: 2-3 days**

---

## Phase 2 — K3s Provisioning via SSH

**Goal:** Automatically provision K3s on user's VPS via SSH.

### 2.1 New K3s Provisioner

Create `apps/backend/internal/provisioner/provision_k3s.go` following the pattern of `infrastructure.go`:

**Provisioning steps:**

1. SSH connect (reuse existing)
2. Install K3s: `curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik" sh -`
   - Disable builtin Traefik to install separately with custom config
3. Wait for K3s to be ready: `k3s kubectl get nodes`
4. Install Traefik as Ingress Controller with Let's Encrypt (cert-manager or Traefik CRDs)
5. Create namespace `flowdeploy` for apps
6. Copy kubeconfig (`/etc/rancher/k3s/k3s.yaml`) to agent directory
7. Install agent binary (same binary, new flag `--runtime=k3s`)
8. Configure and start systemd unit

### 2.2 Frontend — Add Server Dialog

Update `apps/frontend/src/features/servers/components/add-server-dialog.tsx`:

- Add runtime select: "Docker" (default) | "Kubernetes (K3s)"
- When K3s is selected: domain field becomes required (for Ingress + TLS)
- Show K3s-specific info/tooltip explaining the benefits

### 2.3 Provision Progress (SSE)

New SSE steps for K3s provisioning: `install_k3s`, `wait_k3s_ready`, `configure_ingress`, `create_namespace`, `copy_kubeconfig`

**Estimated effort: 3-5 days**

---

## Phase 3 — Kubernetes Manifest Generator

**Goal:** Generate Kubernetes manifests instead of docker-compose.yml for K3s deploys.

### 3.1 New Package

Create `apps/shared/pkg/k8smanifest/generator.go` (equivalent to `apps/shared/pkg/compose/generator.go`).

Generate 3 resources per app:

**Deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {appName}
  namespace: flowdeploy
  labels:
    app: {appName}
    managed-by: flowdeploy
spec:
  replicas: {replicas}  # default 1, configurable
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  selector:
    matchLabels:
      app: {appName}
  template:
    metadata:
      labels:
        app: {appName}
    spec:
      containers:
      - name: {appName}
        image: {image}:{tag}
        ports:
        - containerPort: {port}
        envFrom:
        - secretRef:
            name: {appName}-env
        livenessProbe:
          httpGet:
            path: {healthPath}
            port: {port}
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: {healthPath}
            port: {port}
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            memory: {memLimit}
            cpu: {cpuLimit}
```

**Service:**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {appName}
  namespace: flowdeploy
spec:
  selector:
    app: {appName}
  ports:
  - port: {port}
    targetPort: {port}
```

**IngressRoute (Traefik CRD):**

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: {appName}
  namespace: flowdeploy
spec:
  entryPoints:
  - websecure
  routes:
  - match: Host(`{domain}`)
    kind: Rule
    services:
    - name: {appName}
      port: {port}
  tls:
    certResolver: letsencrypt
```

### 3.2 Inputs

Reuse the same inputs from the compose generator: AppConfig, RuntimeConfig, HealthCheckConfig, domains, env vars. The generator interface should be shared so the worker can call either generator transparently.

**Estimated effort: 3-4 days**

---

## Phase 4 — Agent: K3s Support

**Goal:** The agent executes deploys via kubectl on the K3s server.

### 4.1 New K3s Deploy Executor

Create `apps/agent/internal/deploy/k3s_executor.go` alongside the existing `executor.go`:

**K3s deploy pipeline:**

1. Git sync (reuse existing)
2. Docker build (reuse — K3s uses Docker/OCI images)
3. `k3s ctr images import` (import local image into K3s containerd)
4. Generate manifests (using the generator from Phase 3)
5. Create/update Secret with env vars: `kubectl apply -f secret.yaml`
6. `kubectl apply -f deployment.yaml -f service.yaml -f ingress.yaml`
7. `kubectl rollout status deployment/{appName} --timeout=120s` (wait for rolling update)
8. On failure: `kubectl rollout undo deployment/{appName}`

### 4.2 Container Management RPCs Adaptation

The agent needs to adapt container management RPCs for K3s:

| Current RPC | Docker Implementation | K3s Implementation |
|---|---|---|
| ListContainers | `docker ps` | `kubectl get pods -n flowdeploy` |
| GetContainerLogs | `docker logs` | `kubectl logs pod/{name}` |
| GetContainerStats | `docker stats` | `kubectl top pod` (requires metrics-server) |
| RestartContainer | `docker restart` | `kubectl rollout restart deployment/{name}` |
| StopContainer | `docker stop` | `kubectl scale deployment --replicas=0` |
| StartContainer | `docker start` | `kubectl scale deployment --replicas=N` |
| RemoveContainer | `docker rm` | `kubectl delete deployment/{name}` |
| ExecContainer | `docker exec -it` | `kubectl exec -it pod/{name}` |

**Approach:** Strategy pattern in the agent — detect runtime from `--runtime` flag and dispatch to the correct executor/handler.

### 4.3 Agent Flag

The agent receives `--runtime=docker|k3s` at startup. This determines which executor and container management implementation to use.

**Estimated effort: 5-7 days**

---

## Phase 5 — Frontend: K3s Features

**Goal:** Expose K3s capabilities in the UI.

### 5.1 App Settings (K3s servers only)

- **Replicas**: number input 1-10 (default 1)
- **Update Strategy**: Rolling Update (default) | Recreate
- **Resource limits**: CPU and memory (applies to pod spec)

### 5.2 Container/Pod List

Adapt the container list to show pods when the server runtime is K3s:

- Pod name, status (Running / Pending / CrashLoopBackOff), restarts, age
- Visual indicator: green (Running), yellow (Pending), red (Error)
- Show replica count: "2/3 ready"

### 5.3 Rollback

For K3s servers, rollback uses `kubectl rollout undo` which is faster and maintains revision history.

### 5.4 Deploy Status

Show rolling update progress: "Updating: 1/3 pods ready"

**Estimated effort: 3-4 days**

---

## Phase 6 (future) — Multi-node Clustering

Not part of the MVP, but the architecture supports it:

- Add worker nodes: `k3s agent --server https://{manager}:6443 --token {token}`
- K3s scheduler automatically distributes pods across nodes
- Requires: private registry for shared image storage (instead of local `ctr images import`)
- UI: show node list, pod distribution per node

---

## Effort Summary

| Phase | Scope | Estimated Effort |
|---|---|---|
| 1. Data Model + Strategy | Backend: migration, model, worker refactor | 2-3 days |
| 2. K3s Provisioning | SSH provisioner + frontend dialog | 3-5 days |
| 3. Manifest Generator | Deployment + Service + IngressRoute YAML | 3-4 days |
| 4. Agent K3s Support | Executor + container management RPCs | 5-7 days |
| 5. Frontend | Replicas, pod status, rollback UI | 3-4 days |
| 6. Multi-node | Future | TBD |
| **Total MVP** | | **~3-4 weeks** |

## Key Files Affected

**Backend:**

- `apps/backend/internal/domain/server.go` — Runtime field
- `apps/backend/internal/provisioner/` — new `provision_k3s.go`
- `apps/backend/internal/engine/worker.go` — strategy pattern by runtime
- `apps/backend/migrations/` — new migration for runtime column

**Shared:**

- `apps/shared/pkg/k8smanifest/` — new package (manifest generator)

**Agent:**

- `apps/agent/internal/deploy/` — new `k3s_executor.go`
- `apps/agent/internal/grpcserver/server.go` — dispatch by runtime

**Proto:**

- `apps/proto/flowdeploy/v1/deploy.proto` — `runtime` field in DeployRequest

**Frontend:**

- `apps/frontend/src/features/servers/components/add-server-dialog.tsx` — runtime select
- `apps/frontend/src/features/apps/` — replicas config, pod status
- `apps/frontend/src/features/containers/` — pod list adaptation

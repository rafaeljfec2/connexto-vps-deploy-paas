# K3s Remote Deploy — Roadmap

> **Status: planned (📋).** No production code lives in the repository for
> this feature yet. This document is the canonical proposal we are committing
> to before implementation starts.

## 1. Goal

Add **K3s** as an *optional* runtime for remote deploys. When a server is
registered, the user picks the runtime:

- **Docker** (default, current behavior, no behavioral change).
- **K3s** (new path, opt-in per server) — enables rolling updates, replicas,
  self-healing, and Ingress with automatic TLS via cert-manager.

The Docker path stays untouched. K3s is offered for users who need
zero-downtime deploys and basic horizontal scaling on a single VPS without
adopting full Kubernetes.

## 2. Why K3s and not full Kubernetes

| Concern              | Full Kubernetes        | K3s (chosen target)             |
| -------------------- | ---------------------- | ------------------------------- |
| Install footprint    | Multi-node, complex    | Single binary, single node ok   |
| Memory baseline      | ~2 GB                  | < 512 MB                        |
| Operational burden   | High                   | Low — managed via systemd       |
| API surface          | Same (`kubectl`)       | Same (`kubectl`)                |
| Multi-node growth    | Native                 | Optional via `k3s server`/`agent`|
| TLS                  | Bring-your-own         | Built-in cert-manager templates |

K3s gives us 90% of the user-visible benefits with a fraction of the
operational cost. It also lets users **graduate** to multi-node Kubernetes
later without changing manifests.

## 3. Non-goals

- Replacing Docker entirely for existing users.
- Supporting arbitrary upstream Kubernetes distributions in v1.
- Hosting our own multi-tenant K3s cluster.
- Cross-server deployments (a deploy still targets one VPS at a time).

## 4. User-facing model

When adding a server (UI: **Servers → New Server**), users see a new field
**Runtime** with two options:

1. **Docker (default)** — current behavior, nothing changes.
2. **K3s (advanced)** — installs K3s instead of plain Docker; enables
   replica controls and rolling deploy strategy on the app form.

When creating an app on a K3s server, the form gains:

- `replicas` (default `1`).
- `strategy` (`rolling` default; `recreate` available).
- `resources.requests` and `resources.limits`.
- `ingress` toggle (default `true`) reusing the existing custom domain UX.

The dashboard surfaces:

- Pod count and per-pod health.
- Deployment status (desired vs. ready).
- Rollout history with one-click rollback to a previous revision.

## 5. Architecture

```
┌──────────────────┐
│  Add Server UI   │
└────────┬─────────┘
         │
   ┌─────▼──────┐
   │  Runtime?  │
   └──┬──────┬──┘
      │      │
 docker     k3s
   │         │
   │         ▼
   │   provisioner/
   │     install_k3s.go
   │     (apt + systemd unit)
   │         │
   │         ▼
   │   apps/agent/internal/runtime/k3s/
   │     deploy.go        (kubectl apply / helm)
   │     status.go        (pods, deployment status)
   │     ingress.go       (cert-manager + Traefik IngressRoute)
   │
   ▼
docker path
(unchanged)
```

### Backend touchpoints

- `internal/domain/server.go` — add `Runtime` field (`docker` | `k3s`).
- `internal/provisioner/` — branch on runtime; `install_k3s.go` handles K3s
  setup, `kubeconfig` retrieval, and CA injection.
- `internal/agentclient/` — new methods `ListPods`, `RolloutHistory`,
  `RollbackDeployment`.
- `internal/engine/worker.go` — pick the runtime adapter based on
  `server.Runtime`.

### Agent touchpoints

- `internal/runtime/docker/` — extracted current implementation (no
  behavior change).
- `internal/runtime/k3s/` — new package wrapping `kubectl` (or the official
  Go client). Generates manifests from `paasdeploy.json`:
  - `Deployment` with rolling strategy and `paasdeploy.json` env vars.
  - `Service` of type `ClusterIP`.
  - `Traefik IngressRoute` (or built-in Ingress) with TLS via cert-manager.
- `cleanup/` — extend pruning to handle dangling K3s resources.

### Proto changes

`apps/proto/flowdeploy/v1/agent.proto` gains:

- `ListPods(ListPodsRequest) returns (ListPodsResponse)`
- `GetPodLogs(PodLogsRequest) returns (stream PodLogEntry)`
- `RolloutHistory(RolloutHistoryRequest) returns (RolloutHistoryResponse)`
- `RollbackDeployment(RollbackDeploymentRequest) returns (RollbackDeploymentResponse)`

These are **additive**. Existing RPCs stay unchanged. `buf breaking` must
remain green.

## 6. Migration / opt-in

- Existing servers default to `runtime = "docker"` (column already exists,
  added by migration `000005_add_runtime`, currently unused for K3s).
- New servers can opt into K3s at provisioning time only. Switching runtime
  on an existing server is **out of scope** for v1 (would require draining
  Docker workloads and re-provisioning).
- Apps are bound to the runtime of their server. Moving an app from a
  Docker server to a K3s server is treated as creating a new app.

## 7. Phasing

### Phase 1 — Foundation (small PR)

- Surface `Runtime` in the domain, repository, API and UI.
- No behavioral change yet (only Docker path executes).
- Provisioner refuses `runtime = "k3s"` with an explicit "coming soon".

### Phase 2 — Provisioning K3s

- `provisioner/install_k3s.go` installs K3s + cert-manager + Traefik addon.
- Backend retrieves `kubeconfig` over SSH and stores it encrypted on the
  server row.
- Agent on a K3s server uses the in-cluster service account instead.

### Phase 3 — Agent runtime adapter

- Extract current Docker logic into `internal/runtime/docker`.
- Implement `internal/runtime/k3s` for Deployment/Service/IngressRoute.
- Reuse the existing deploy queue and worker; only the adapter changes.

### Phase 4 — Dashboard surface

- Pod list, rollout status, replicas controls in the UI.
- Per-pod logs (using the new `GetPodLogs` stream).
- Rollback button maps to `kubectl rollout undo`.

### Phase 5 — Operations

- Cleanup of orphan K3s resources.
- Backups / disaster recovery story for `etcd` (single-node).
- Auto-update of the agent on K3s servers (run as DaemonSet vs. systemd?).

## 8. Risks and open questions

| Risk / Question                                                | Mitigation / Plan                                  |
| -------------------------------------------------------------- | -------------------------------------------------- |
| K3s footprint may be too large for small VPSs                  | Document minimum specs (2 vCPU / 2 GB RAM)         |
| Cert-manager + Traefik addon vs. existing Traefik install      | Use Traefik addon shipped with K3s; isolate Docker Traefik per server |
| Agent runtime when the host runs K3s                           | Run agent as a `Deployment` with hostPath socket access |
| Rollback semantics differ from Docker (revisions, history)     | Map "rollback" to `kubectl rollout undo`           |
| Multi-replica support requires shared volumes                  | v1 documents the limitation; v2 considers CSI     |
| Increased mTLS complexity (kube-apiserver)                     | Keep mTLS on the agent boundary only               |
| User confusion between Docker and K3s features                 | Conditional UI based on `server.runtime`           |

## 9. Acceptance criteria for v1 GA

- [ ] User can register a server selecting `runtime = "k3s"`.
- [ ] Provisioning installs K3s, cert-manager and Traefik addon idempotently.
- [ ] User can deploy a sample app with `replicas = 3` and observe a rolling
      update on a new push.
- [ ] Per-pod logs are visible in the dashboard via SSE-bridged stream.
- [ ] Rollback restores the previous revision and the dashboard reflects it.
- [ ] Custom domains keep working (Cloudflare integration is reused).
- [ ] All existing Docker-based features remain unchanged.
- [ ] Documentation: this file becomes a real architecture doc, plus a new
      tutorial `K3S_SERVER_TUTORIAL.md`.

## 10. References

- [K3s docs](https://docs.k3s.io/)
- [Traefik on K3s](https://docs.k3s.io/networking#traefik-ingress-controller)
- [cert-manager on K3s](https://cert-manager.io/docs/)
- Internal: [`docs/REMOTE_SSH_DEPLOY.md`](./REMOTE_SSH_DEPLOY.md),
  [`docs/REMOTE_SERVER_TUTORIAL.md`](./REMOTE_SERVER_TUTORIAL.md),
  [`docs/FEATURES_ROADMAP.md`](./FEATURES_ROADMAP.md)

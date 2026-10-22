# Tutorial: Register and Provision a Remote Server

Step-by-step to prepare a VPS, register it in FlowDeploy, and provision the agent automatically. For architecture behind each step, see [`REMOTE_SSH_DEPLOY.md`](./REMOTE_SSH_DEPLOY.md).

## 1. Overview

You will:

1. Prepare the target VPS (operating system, network, user).
2. Ensure the FlowDeploy backend is ready to issue certificates
   and expose the gRPC port.
3. Register the server in the dashboard.
4. Trigger **Provision** and follow the logs.
5. Validate that the agent is `online` and use the machine for deploys.

> Estimated time: 5 to 10 minutes per server.

## 2. VPS prerequisites

Supported OS: **Ubuntu Server 22.04+** (any Linux with
`systemd` and Docker available also works).

| Item                                   | Why                                                            |
| -------------------------------------- | -------------------------------------------------------------- |
| SSH access (key **or** password)       | provisioning uses SSH                                          |
| User with `sudo` (or root)             | install Docker and create the `systemd` unit                   |
| Internet connection                    | download Docker, Traefik, and the agent binary                 |
| TCP port 22 open                       | SSH during provisioning                                        |
| TCP ports 80/443 open                  | Traefik serves app HTTP/HTTPS                                  |
| TCP port 50052 (or custom)             | agent gRPC port; **only the backend needs to reach it**        |
| Public hostname or fixed IP            | shown in the dashboard and on certificates                     |

> The backend does not need a fixed IP, but it must be reachable from the VPS
> over HTTPS (for `Register`/`Heartbeat`) and gRPC (for deploy).

## 3. Backend prerequisites

Before the first provisioning run, ensure:

1. The variables below are in the backend `.env`:
   - `GRPC_PORT` (default `50051`) — backend internal gRPC port.
   - `GRPC_AGENT_PORT` (default `50052`) — port exposed for agents.
   - `BACKEND_PUBLIC_HOST` (or equivalent) — backend public hostname.
2. Traefik has a **TCP route SNI** publishing `GRPC_AGENT_PORT` on the
   same hostname (see `deploy/traefik/`).
3. Migration `000014_pki_ca` has been applied (generates the internal CA
   automatically on startup).
4. The agent binary matching `AGENT_VERSION` is built and
   available through `agentdownload`.

> In development you can run the agent pointing directly
> at `localhost:50051` without Traefik, as described in `README.md`.

## 4. Register the server in the dashboard

1. Log in to the dashboard as `admin`.
2. Go to **Servers → New Server**.
3. Fill in:
   - **Name**: friendly name (e.g. `prod-eu-1`).
   - **Host**: public IP or hostname of the VPS.
   - **SSH Port**: `22` (default).
   - **SSH User**: user with `sudo` or `root`.
   - **Authentication**: paste the private key (recommended) **or** provide a
     temporary password.
   - **ACME Email**: email for Let's Encrypt on this VPS (optional, but
     required for automatic SSL on apps).
   - **Agent Update Mode**: `auto` (recommended) or `manual`.
4. Save. The server appears with status `pending`.

## 5. Trigger provisioning

1. In the server list, click **Provision**.
2. A window with **real-time logs** opens. You will see steps such as:

```
[01/10] Connecting via SSH to 203.0.113.10
[02/10] Detecting privileges and configuring remote environment
[03/10] Installing Docker Engine and Compose v2
[04/10] Creating Docker network paasdeploy_net
[05/10] Installing Traefik (acme_email configured)
[06/10] Generating agent mTLS certificate
[07/10] Uploading agent binary (paasdeploy-agent vX.Y.Z)
[08/10] Creating systemd unit paasdeploy-agent.service
[09/10] Starting service and waiting for Register
[10/10] Server online
```

3. On error, the step turns red and the dashboard shows the full output
   of the command that failed. Logs are also in
   `apps/backend` (with `slog` correlated by `serverId`).

> The flow is idempotent: running **Provision** again on an already
> provisioned server does not reinstall what exists; it only ensures everything is
> in a consistent state.

## 6. Post-provisioning validation

After status is `online`, validate quickly:

```bash
ssh <user>@<host>
sudo systemctl status paasdeploy-agent     # active (running)
sudo journalctl -u paasdeploy-agent -n 50  # recent logs
sudo docker ps                              # paasdeploy-agent + traefik (if ACME)
sudo docker network ls | grep paasdeploy_net
```

In the dashboard:

- The server card shows CPU/RAM/disk metrics received via SSE.
- The **Containers** tab lists existing containers (you should see
  `paasdeploy-agent` and `traefik` if it was installed).
- The **Certificates** tab lists Traefik certificates for that VPS.

## 7. Register and deploy an app on this server

1. **Apps → New App**.
2. Fill in:
   - **Repository URL**: `https://github.com/org/repo.git` (or via GitHub App).
   - **Branch**: `main` (default).
   - **Workdir**: relative path in the monorepo, if applicable.
   - **Server**: select the newly provisioned VPS.
   - **Domain**: domain served by Traefik on that VPS.
3. Confirm. The backend creates a GitHub webhook (via GitHub App, if connected),
   enqueues the first deploy, and shows logs in real time.

## 8. Day-to-day operations

### Manual redeploy

**Redeploy** button (or `POST /api/apps/:id/redeploy`). Uses the last known SHA
on the branch.

### Rollback

**Rollback** button (or `POST /api/apps/:id/rollback`). Returns to the previous
version **and** re-runs the health check before reporting success.

### Web terminal

**Terminal** tab inside a container. Uses `ExecContainer` (bidirectional gRPC)
with a real PTY — you can run `htop`, `vim`, etc.

### Update the agent

- **Auto** (recommended): backend runs `PushUpdate` when it detects a newer
  version via heartbeat.
- **Manual**: on the server page, click **Push Update** after building
  a new agent binary.

### Scheduled cleanup

The cleanup scheduler runs inside the agent (`apps/agent/internal/cleanup`).
You can trigger a run now from the UI or via
`POST /api/cleanup/run?serverId=...`.

## 9. Troubleshooting

| Symptom                                         | What to check                                                               |
| ----------------------------------------------- | --------------------------------------------------------------------------- |
| Provisioning stops at **Connecting via SSH**    | port 22 open? correct key? does `sudo` require a password?                  |
| **Docker install failed** error                 | supported distro? internet allowed? `apt`/`dnf` available?                  |
| Server stuck in **registering**                 | backend gRPC port reachable from VPS? CA present in `pki_ca`?               |
| Agent restarts in a loop                        | `journalctl -u paasdeploy-agent` shows the real error (e.g. port 50052 in use) |
| Deploy stays **pending** for a long time        | `DEPLOY_WORKERS` zero? anything in dispatcher logs?                         |
| Healthcheck always fails                        | `paasdeploy.json` with correct `port` and `healthcheck.path`?               |
| App TLS does not work                           | `acme_email` set on the server? ports 80/443 open?                          |

## 10. Deprovision a server

1. **Servers → <server> → Delete**. You choose whether to:
   - **Remove from dashboard only** (leaves the agent running, useful for
     manual migration).
   - **Remove completely** (remote routine: stop service, remove
     `systemd` unit, delete binary, remove certificates).
2. Apps tied to the server become orphaned and should be
   reassigned before deletion.

## 11. Best practices

- Use **one server per logical environment** (e.g. `prod`, `staging`) and do not
  mix competing workloads on a single small VPS.
- Set `acme_email` on all public servers for automatic SSL.
- Monitor the server card regularly; red metrics appear
  via SSE within seconds.
- Keep `AGENT_VERSION` updated in the repository so auto-update
  keeps the fleet homogeneous.
- Rotate mTLS certificates periodically (script planned on the roadmap).

## 12. Useful commands on the VPS

```bash
sudo systemctl status paasdeploy-agent
sudo systemctl restart paasdeploy-agent
sudo journalctl -u paasdeploy-agent -f

sudo docker ps
sudo docker logs <container> -f
sudo docker compose -f /data/apps/<app>/docker-compose.yml logs

sudo cat /etc/systemd/system/paasdeploy-agent.service
sudo ls -l ~/paasdeploy-agent/
```

> For persistent issues, attach the output of `journalctl -u
> paasdeploy-agent --no-pager -n 200` when opening an issue.

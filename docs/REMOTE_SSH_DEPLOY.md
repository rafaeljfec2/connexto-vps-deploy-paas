# Arquitetura de Deploy Remoto com Agent

## Visao Geral

O FlowDeploy suporta deploy de aplicacoes em servidores remotos atraves de um **Agent leve** instalado em cada servidor. O agent pode ser instalado:

1. **Automaticamente** - Durante o onboarding de um app, o FlowDeploy conecta via SSH e instala o agent
2. **Manualmente** - Usuario executa script de instalacao no servidor

## Por que Agent ao inves de SSH Direto?

| Aspecto     | SSH Direto                    | Agent                              |
| ----------- | ----------------------------- | ---------------------------------- |
| Seguranca   | Chaves SSH no banco           | Token de autenticacao              |
| Conexao     | Inbound (requer porta aberta) | Outbound (agent conecta ao server) |
| Firewall    | Precisa abrir porta 22        | Nenhuma porta a abrir              |
| Logs        | Dificil de coletar            | Streaming em tempo real            |
| Status      | Sem visibilidade              | Heartbeat continuo                 |
| Atualizacao | Manual                        | Auto-update                        |

**Importante:** SSH e usado apenas para o provisionamento inicial (instalar o agent). As credenciais SSH NAO sao armazenadas - sao usadas uma unica vez e descartadas.

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      FlowDeploy Server (Cloud)                          │
│                                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │ Frontend │  │ Backend  │  │PostgreSQL│  │ Traefik  │               │
│  │  React   │  │   Go     │  │          │  │          │               │
│  └────┬─────┘  └────┬─────┘  └──────────┘  └──────────┘               │
│       │             │                                                   │
│       │             │ WebSocket/HTTPS                                   │
│       │             │ (conexao de SAIDA do agent)                       │
└───────┼─────────────┼───────────────────────────────────────────────────┘
        │             │
        │             ▼
┌───────┼─────────────────────────────────────────────────────────────────┐
│       │            Servidor Remoto 1 (Sao Paulo)                        │
│       │                                                                 │
│       │   ┌────────────────┐                                           │
│       │   │  FlowDeploy    │◄──── Conexao WebSocket para o Server      │
│       │   │    Agent       │                                           │
│       │   └───────┬────────┘                                           │
│       │           │                                                     │
│       │           ▼                                                     │
│       │   ┌──────────┐  ┌──────────┐  ┌──────────┐                    │
│       │   │ Docker   │  │  App A   │  │  App B   │                    │
│       │   │ Engine   │  │          │  │          │                    │
│       │   └──────────┘  └──────────┘  └──────────┘                    │
└───────┼─────────────────────────────────────────────────────────────────┘
        │             │
        │             ▼
┌───────┼─────────────────────────────────────────────────────────────────┐
│       │            Servidor Remoto 2 (Frankfurt)                        │
│       │                                                                 │
│       │   ┌────────────────┐                                           │
│       │   │  FlowDeploy    │◄──── Conexao WebSocket para o Server      │
│       │   │    Agent       │                                           │
│       │   └───────┬────────┘                                           │
│       │           │                                                     │
│       │           ▼                                                     │
│       │   ┌──────────┐  ┌──────────┐                                   │
│       │   │ Docker   │  │  App C   │                                   │
│       │   │ Engine   │  │          │                                   │
│       │   └──────────┘  └──────────┘                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Fluxo de Onboarding com Provisionamento Automatico

### Visao Geral do Fluxo

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ONBOARDING WIZARD                                    │
│                                                                             │
│  Step 1          Step 2           Step 3            Step 4                 │
│  Repository  →   Server       →   Environment   →   Review & Deploy        │
│                                                                             │
│  - Repo URL      - Select or      - Env vars        - Summary              │
│  - Branch          Add Server                       - Deploy button        │
│  - Workdir       - SSH Setup                                               │
│                    (if new)                                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Step 2 - Server Selection (Detalhado)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  SELECT DEPLOYMENT TARGET                                                   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  ○ Deploy Local (this server)                                       │   │
│  │     Deploy na mesma maquina onde o FlowDeploy esta rodando          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  ● Deploy Remote (another server)                                   │   │
│  │                                                                     │   │
│  │  Select a server:                                                   │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │ ● production-server-1     192.168.1.100    ● Online        │   │   │
│  │  │ ○ staging-server          192.168.1.101    ● Online        │   │   │
│  │  │ ○ dev-server              192.168.1.102    ○ Offline       │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │  + Add New Server                                           │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Add New Server Dialog

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ADD NEW SERVER                                                     [X]    │
│                                                                             │
│  Server Information                                                         │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │ Name:          [production-api                              ]      │    │
│  │ Host/IP:       [192.168.1.100                               ]      │    │
│  │ SSH Port:      [22                                          ]      │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  SSH Authentication (used only for agent installation)                      │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │ ○ Password                                                         │    │
│  │   Username:    [root                                        ]      │    │
│  │   Password:    [••••••••                                    ]      │    │
│  │                                                                    │    │
│  │ ● SSH Key                                                          │    │
│  │   Username:    [root                                        ]      │    │
│  │   Private Key: [Paste your private key here...              ]      │    │
│  │                [                                            ]      │    │
│  │                [                                            ]      │    │
│  │                                                                    │    │
│  │   ⚠️ The key is used only once and is NOT stored                   │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                     [ Test Connection ]                            │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  ✓ Connection successful                                          │    │
│  │  ✓ Docker installed (v24.0.7)                                     │    │
│  │  ✓ Git installed (v2.34.1)                                        │    │
│  │  ○ FlowDeploy Agent not installed                                 │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│                               [ Cancel ]  [ Install Agent & Add Server ]   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Provisioning Progress

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  INSTALLING FLOWDEPLOY AGENT                                                │
│                                                                             │
│  Server: production-api (192.168.1.100)                                     │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  ✓ Connecting via SSH...                                          │    │
│  │  ✓ Checking system requirements...                                │    │
│  │  ✓ Downloading FlowDeploy Agent...                                │    │
│  │  ✓ Installing agent binary...                                     │    │
│  │  ✓ Configuring systemd service...                                 │    │
│  │  ● Starting agent service...                                      │    │
│  │  ○ Waiting for agent connection...                                │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  $ Downloading agent from https://flowdeploy.io/agent/linux-amd64 │    │
│  │  $ Installing to /opt/flowdeploy/agent                            │    │
│  │  $ Creating systemd service...                                    │    │
│  │  $ systemctl enable flowdeploy-agent                              │    │
│  │  $ systemctl start flowdeploy-agent                               │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Provisioning Complete

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ✓ SERVER ADDED SUCCESSFULLY                                                │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  Server: production-api                                           │    │
│  │  IP: 192.168.1.100                                                │    │
│  │  Status: ● Online                                                 │    │
│  │  Agent Version: v1.0.0                                            │    │
│  │  OS: Ubuntu 22.04 (amd64)                                         │    │
│  │  Docker: v24.0.7                                                  │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  The SSH credentials have been discarded. All future communication         │
│  will be through the secure agent connection.                              │
│                                                                             │
│                                              [ Continue to Next Step ]     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Fluxo Tecnico de Provisionamento

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Frontend │     │ Backend  │     │  Server  │     │  Agent   │
│          │     │          │     │ (Remote) │     │          │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ 1. Add Server  │                │                │
     │ (with SSH creds)                │                │
     │───────────────>│                │                │
     │                │                │                │
     │                │ 2. SSH Connect │                │
     │                │───────────────>│                │
     │                │                │                │
     │                │ 3. Verify Docker/Git           │
     │                │<──────────────>│                │
     │                │                │                │
     │ 4. Progress    │                │                │
     │    (SSE)       │                │                │
     │<───────────────│                │                │
     │                │                │                │
     │                │ 5. Download &  │                │
     │                │    Install Agent               │
     │                │───────────────>│                │
     │                │                │                │
     │                │ 6. Start Agent │                │
     │                │───────────────>│────┐           │
     │                │                │    │ Start     │
     │                │                │<───┘           │
     │                │                │                │
     │                │ 7. Close SSH   │                │
     │                │       X        │                │
     │                │                │                │
     │                │ 8. Agent connects via WebSocket │
     │                │<───────────────────────────────│
     │                │                │                │
     │ 9. Server Online                │                │
     │<───────────────│                │                │
     │                │                │                │
```

## Banco de Dados

### Migration 000006 - Servidores

```sql
CREATE TYPE server_status AS ENUM ('pending', 'provisioning', 'online', 'offline', 'error');

CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    ssh_port INTEGER NOT NULL DEFAULT 22,
    agent_token VARCHAR(64) NOT NULL UNIQUE,
    agent_version VARCHAR(20),
    status server_status NOT NULL DEFAULT 'pending',
    last_heartbeat_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_agent_token ON servers(agent_token);

ALTER TABLE apps ADD COLUMN server_id UUID REFERENCES servers(id) ON DELETE SET NULL;
ALTER TABLE apps ADD COLUMN deploy_mode VARCHAR(20) NOT NULL DEFAULT 'local';

CREATE INDEX idx_apps_server_id ON apps(server_id);
CREATE INDEX idx_apps_deploy_mode ON apps(deploy_mode);
```

### Migration 000006 - Down

```sql
ALTER TABLE apps DROP COLUMN IF EXISTS deploy_mode;
ALTER TABLE apps DROP COLUMN IF EXISTS server_id;

DROP INDEX IF EXISTS idx_servers_agent_token;
DROP INDEX IF EXISTS idx_servers_status;
DROP TABLE IF EXISTS servers;
DROP TYPE IF EXISTS server_status;
```

## Backend Implementation

### Domain Layer

**Arquivo:** `apps/backend/internal/domain/server.go`

```go
package domain

import (
    "encoding/json"
    "time"
)

type ServerStatus string

const (
    ServerStatusPending      ServerStatus = "pending"
    ServerStatusProvisioning ServerStatus = "provisioning"
    ServerStatusOnline       ServerStatus = "online"
    ServerStatusOffline      ServerStatus = "offline"
    ServerStatusError        ServerStatus = "error"
)

type Server struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    Hostname        string          `json:"hostname"`
    IPAddress       *string         `json:"ipAddress,omitempty"`
    SSHPort         int             `json:"sshPort"`
    AgentToken      string          `json:"-"`
    AgentVersion    *string         `json:"agentVersion,omitempty"`
    Status          ServerStatus    `json:"status"`
    LastHeartbeatAt *time.Time      `json:"lastHeartbeatAt,omitempty"`
    Metadata        json.RawMessage `json:"metadata,omitempty"`
    CreatedAt       time.Time       `json:"createdAt"`
    UpdatedAt       time.Time       `json:"updatedAt"`
}

type ServerMetadata struct {
    OS            string `json:"os,omitempty"`
    Architecture  string `json:"arch,omitempty"`
    CPUCores      int    `json:"cpuCores,omitempty"`
    MemoryMB      int    `json:"memoryMb,omitempty"`
    DockerVersion string `json:"dockerVersion,omitempty"`
}

type CreateServerInput struct {
    Name     string `json:"name" validate:"required,min=2,max=63"`
    Hostname string `json:"hostname" validate:"required"`
    SSHPort  int    `json:"sshPort"`
}

type ProvisionServerInput struct {
    Name       string  `json:"name" validate:"required,min=2,max=63"`
    Hostname   string  `json:"hostname" validate:"required"`
    SSHPort    int     `json:"sshPort"`
    SSHUser    string  `json:"sshUser" validate:"required"`
    SSHPassword *string `json:"sshPassword,omitempty"`
    SSHKey     *string `json:"sshKey,omitempty"`
}

type ProvisionProgress struct {
    Step    string `json:"step"`
    Message string `json:"message"`
    Status  string `json:"status"`
    Log     string `json:"log,omitempty"`
}

type ServerWithToken struct {
    Server
    AgentToken string `json:"agentToken"`
}
```

### SSH Provisioner

**Arquivo:** `apps/backend/internal/provisioner/ssh_provisioner.go`

```go
package provisioner

import (
    "context"
    "fmt"
    "io"
    "strings"
    "time"

    "golang.org/x/crypto/ssh"
    "flowdeploy/internal/domain"
)

type SSHProvisioner struct {
    serverURL string
    logger    *slog.Logger
}

type ProvisionResult struct {
    Success bool
    Error   string
    Logs    []string
}

func NewSSHProvisioner(serverURL string) *SSHProvisioner {
    return &SSHProvisioner{
        serverURL: serverURL,
        logger:    slog.Default(),
    }
}

func (p *SSHProvisioner) Provision(
    ctx context.Context,
    input domain.ProvisionServerInput,
    agentToken string,
    progressChan chan<- domain.ProvisionProgress,
) (*ProvisionResult, error) {
    result := &ProvisionResult{Logs: []string{}}

    // 1. Conectar via SSH
    p.sendProgress(progressChan, "connect", "Connecting via SSH...", "running", "")

    client, err := p.connect(input)
    if err != nil {
        p.sendProgress(progressChan, "connect", "SSH connection failed", "error", err.Error())
        return nil, fmt.Errorf("SSH connection failed: %w", err)
    }
    defer client.Close()

    p.sendProgress(progressChan, "connect", "Connected via SSH", "done", "")

    // 2. Verificar requisitos
    p.sendProgress(progressChan, "verify", "Checking system requirements...", "running", "")

    if err := p.verifyRequirements(ctx, client, result); err != nil {
        p.sendProgress(progressChan, "verify", "System requirements not met", "error", err.Error())
        return result, err
    }

    p.sendProgress(progressChan, "verify", "System requirements verified", "done", "")

    // 3. Download do agent
    p.sendProgress(progressChan, "download", "Downloading FlowDeploy Agent...", "running", "")

    if err := p.downloadAgent(ctx, client, result); err != nil {
        p.sendProgress(progressChan, "download", "Download failed", "error", err.Error())
        return result, err
    }

    p.sendProgress(progressChan, "download", "Agent downloaded", "done", "")

    // 4. Instalar agent
    p.sendProgress(progressChan, "install", "Installing agent...", "running", "")

    if err := p.installAgent(ctx, client, agentToken, result); err != nil {
        p.sendProgress(progressChan, "install", "Installation failed", "error", err.Error())
        return result, err
    }

    p.sendProgress(progressChan, "install", "Agent installed", "done", "")

    // 5. Configurar systemd
    p.sendProgress(progressChan, "configure", "Configuring systemd service...", "running", "")

    if err := p.configureSystemd(ctx, client, agentToken, result); err != nil {
        p.sendProgress(progressChan, "configure", "Configuration failed", "error", err.Error())
        return result, err
    }

    p.sendProgress(progressChan, "configure", "Service configured", "done", "")

    // 6. Iniciar servico
    p.sendProgress(progressChan, "start", "Starting agent service...", "running", "")

    if err := p.startService(ctx, client, result); err != nil {
        p.sendProgress(progressChan, "start", "Failed to start service", "error", err.Error())
        return result, err
    }

    p.sendProgress(progressChan, "start", "Agent service started", "done", "")

    // 7. Aguardar conexao do agent
    p.sendProgress(progressChan, "wait", "Waiting for agent connection...", "running", "")

    result.Success = true
    return result, nil
}

func (p *SSHProvisioner) connect(input domain.ProvisionServerInput) (*ssh.Client, error) {
    var authMethods []ssh.AuthMethod

    if input.SSHPassword != nil && *input.SSHPassword != "" {
        authMethods = append(authMethods, ssh.Password(*input.SSHPassword))
    }

    if input.SSHKey != nil && *input.SSHKey != "" {
        signer, err := ssh.ParsePrivateKey([]byte(*input.SSHKey))
        if err != nil {
            return nil, fmt.Errorf("invalid SSH key: %w", err)
        }
        authMethods = append(authMethods, ssh.PublicKeys(signer))
    }

    if len(authMethods) == 0 {
        return nil, fmt.Errorf("no authentication method provided")
    }

    config := &ssh.ClientConfig{
        User:            input.SSHUser,
        Auth:            authMethods,
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Timeout:         30 * time.Second,
    }

    port := input.SSHPort
    if port == 0 {
        port = 22
    }

    addr := fmt.Sprintf("%s:%d", input.Hostname, port)
    return ssh.Dial("tcp", addr, config)
}

func (p *SSHProvisioner) verifyRequirements(ctx context.Context, client *ssh.Client, result *ProvisionResult) error {
    checks := []struct {
        name    string
        command string
    }{
        {"Docker", "docker --version"},
        {"Git", "git --version"},
        {"curl", "curl --version"},
    }

    for _, check := range checks {
        output, err := p.runCommand(client, check.command)
        result.Logs = append(result.Logs, fmt.Sprintf("$ %s\n%s", check.command, output))

        if err != nil {
            return fmt.Errorf("%s not found: %w", check.name, err)
        }
    }

    return nil
}

func (p *SSHProvisioner) downloadAgent(ctx context.Context, client *ssh.Client, result *ProvisionResult) error {
    commands := []string{
        "sudo mkdir -p /opt/flowdeploy",
        fmt.Sprintf("sudo curl -fsSL %s/downloads/agent-linux-amd64 -o /opt/flowdeploy/flowdeploy-agent", p.serverURL),
        "sudo chmod +x /opt/flowdeploy/flowdeploy-agent",
    }

    for _, cmd := range commands {
        output, err := p.runCommand(client, cmd)
        result.Logs = append(result.Logs, fmt.Sprintf("$ %s\n%s", cmd, output))

        if err != nil {
            return fmt.Errorf("command failed: %s: %w", cmd, err)
        }
    }

    return nil
}

func (p *SSHProvisioner) installAgent(ctx context.Context, client *ssh.Client, token string, result *ProvisionResult) error {
    configContent := fmt.Sprintf(`FLOWDEPLOY_SERVER=%s
FLOWDEPLOY_TOKEN=%s
`, p.serverURL, token)

    cmd := fmt.Sprintf("echo '%s' | sudo tee /opt/flowdeploy/config.env > /dev/null && sudo chmod 600 /opt/flowdeploy/config.env", configContent)

    output, err := p.runCommand(client, cmd)
    result.Logs = append(result.Logs, fmt.Sprintf("$ (creating config.env)\n%s", output))

    return err
}

func (p *SSHProvisioner) configureSystemd(ctx context.Context, client *ssh.Client, token string, result *ProvisionResult) error {
    serviceContent := `[Unit]
Description=FlowDeploy Agent
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
EnvironmentFile=/opt/flowdeploy/config.env
ExecStart=/opt/flowdeploy/flowdeploy-agent --server $FLOWDEPLOY_SERVER --token $FLOWDEPLOY_TOKEN
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`

    cmd := fmt.Sprintf("echo '%s' | sudo tee /etc/systemd/system/flowdeploy-agent.service > /dev/null", serviceContent)

    output, err := p.runCommand(client, cmd)
    result.Logs = append(result.Logs, fmt.Sprintf("$ (creating systemd service)\n%s", output))

    if err != nil {
        return err
    }

    output, err = p.runCommand(client, "sudo systemctl daemon-reload && sudo systemctl enable flowdeploy-agent")
    result.Logs = append(result.Logs, fmt.Sprintf("$ systemctl daemon-reload && systemctl enable\n%s", output))

    return err
}

func (p *SSHProvisioner) startService(ctx context.Context, client *ssh.Client, result *ProvisionResult) error {
    output, err := p.runCommand(client, "sudo systemctl start flowdeploy-agent")
    result.Logs = append(result.Logs, fmt.Sprintf("$ systemctl start flowdeploy-agent\n%s", output))
    return err
}

func (p *SSHProvisioner) runCommand(client *ssh.Client, cmd string) (string, error) {
    session, err := client.NewSession()
    if err != nil {
        return "", err
    }
    defer session.Close()

    output, err := session.CombinedOutput(cmd)
    return string(output), err
}

func (p *SSHProvisioner) sendProgress(ch chan<- domain.ProvisionProgress, step, message, status, log string) {
    if ch != nil {
        ch <- domain.ProvisionProgress{
            Step:    step,
            Message: message,
            Status:  status,
            Log:     log,
        }
    }
}

func (p *SSHProvisioner) TestConnection(input domain.ProvisionServerInput) (*domain.ServerMetadata, error) {
    client, err := p.connect(input)
    if err != nil {
        return nil, err
    }
    defer client.Close()

    metadata := &domain.ServerMetadata{}

    // Get Docker version
    if output, err := p.runCommand(client, "docker version --format '{{.Server.Version}}'"); err == nil {
        metadata.DockerVersion = strings.TrimSpace(output)
    }

    // Get OS info
    if output, err := p.runCommand(client, "cat /etc/os-release | grep PRETTY_NAME | cut -d'\"' -f2"); err == nil {
        metadata.OS = strings.TrimSpace(output)
    }

    // Get arch
    if output, err := p.runCommand(client, "uname -m"); err == nil {
        metadata.Architecture = strings.TrimSpace(output)
    }

    // Get CPU cores
    if output, err := p.runCommand(client, "nproc"); err == nil {
        fmt.Sscanf(strings.TrimSpace(output), "%d", &metadata.CPUCores)
    }

    return metadata, nil
}
```

### Server Handler

**Arquivo:** `apps/backend/internal/handler/server_handler.go`

```go
package handler

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "flowdeploy/internal/agent"
    "flowdeploy/internal/domain"
    "flowdeploy/internal/provisioner"
    "flowdeploy/internal/repository"
    "flowdeploy/internal/response"
)

type ServerHandler struct {
    repo        *repository.ServerRepository
    hub         *agent.Hub
    provisioner *provisioner.SSHProvisioner
}

func NewServerHandler(
    repo *repository.ServerRepository,
    hub *agent.Hub,
    provisioner *provisioner.SSHProvisioner,
) *ServerHandler {
    return &ServerHandler{
        repo:        repo,
        hub:         hub,
        provisioner: provisioner,
    }
}

// GET /api/servers
func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
    servers, err := h.repo.List(r.Context())
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to list servers")
        return
    }

    for i := range servers {
        if h.hub.IsServerOnline(servers[i].ID) {
            servers[i].Status = domain.ServerStatusOnline
        }
    }

    response.JSON(w, http.StatusOK, servers)
}

// GET /api/servers/:id
func (h *ServerHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    server, err := h.repo.GetByID(r.Context(), id)
    if err != nil {
        response.Error(w, http.StatusNotFound, "server not found")
        return
    }

    if h.hub.IsServerOnline(server.ID) {
        server.Status = domain.ServerStatusOnline
    }

    response.JSON(w, http.StatusOK, server)
}

// DELETE /api/servers/:id
func (h *ServerHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    if err := h.repo.Delete(r.Context(), id); err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to delete server")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// POST /api/servers/test-connection
func (h *ServerHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
    var input domain.ProvisionServerInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid request body")
        return
    }

    metadata, err := h.provisioner.TestConnection(input)
    if err != nil {
        response.JSON(w, http.StatusOK, map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    response.JSON(w, http.StatusOK, map[string]interface{}{
        "success":  true,
        "metadata": metadata,
    })
}

// POST /api/servers/provision (SSE endpoint)
func (h *ServerHandler) Provision(w http.ResponseWriter, r *http.Request) {
    var input domain.ProvisionServerInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // Gerar token para o agent
    token := generateToken()

    // Criar servidor no banco (status: provisioning)
    server, err := h.repo.Create(r.Context(), domain.CreateServerInput{
        Name:     input.Name,
        Hostname: input.Hostname,
        SSHPort:  input.SSHPort,
    }, token)
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to create server")
        return
    }

    _ = h.repo.UpdateStatus(r.Context(), server.ID, domain.ServerStatusProvisioning)

    // Setup SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        response.Error(w, http.StatusInternalServerError, "streaming not supported")
        return
    }

    // Canal para progresso
    progressChan := make(chan domain.ProvisionProgress, 10)

    // Executar provisionamento em goroutine
    go func() {
        defer close(progressChan)

        result, err := h.provisioner.Provision(r.Context(), input, token, progressChan)

        if err != nil {
            _ = h.repo.UpdateStatus(context.Background(), server.ID, domain.ServerStatusError)
            progressChan <- domain.ProvisionProgress{
                Step:    "error",
                Message: err.Error(),
                Status:  "error",
            }
            return
        }

        if result.Success {
            // Aguardar agent conectar (max 30s)
            for i := 0; i < 30; i++ {
                if h.hub.IsServerOnline(server.ID) {
                    progressChan <- domain.ProvisionProgress{
                        Step:    "complete",
                        Message: "Agent connected successfully",
                        Status:  "done",
                    }
                    return
                }
                time.Sleep(time.Second)
            }

            progressChan <- domain.ProvisionProgress{
                Step:    "timeout",
                Message: "Agent did not connect within 30 seconds",
                Status:  "warning",
            }
        }
    }()

    // Enviar eventos SSE
    for progress := range progressChan {
        data, _ := json.Marshal(progress)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }

    // Enviar resultado final
    finalServer, _ := h.repo.GetByID(r.Context(), server.ID)
    finalData, _ := json.Marshal(map[string]interface{}{
        "type":   "result",
        "server": finalServer,
    })
    fmt.Fprintf(w, "data: %s\n\n", finalData)
    flusher.Flush()
}

func generateToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

### Routes

**Modificar:** `apps/backend/internal/server/server.go`

```go
// Rotas de servidores
r.Route("/servers", func(r chi.Router) {
    r.Get("/", serverHandler.List)
    r.Get("/{id}", serverHandler.Get)
    r.Delete("/{id}", serverHandler.Delete)
    r.Post("/test-connection", serverHandler.TestConnection)
    r.Post("/provision", serverHandler.Provision)
})

// WebSocket para agents
r.Get("/agent/ws", agentHub.HandleWebSocket)
```

## Frontend Implementation

### Types

**Arquivo:** `apps/frontend/src/features/servers/types.ts`

```typescript
export type ServerStatus =
  | "pending"
  | "provisioning"
  | "online"
  | "offline"
  | "error";

export interface Server {
  id: string;
  name: string;
  hostname: string;
  ipAddress?: string;
  sshPort: number;
  agentVersion?: string;
  status: ServerStatus;
  lastHeartbeatAt?: string;
  metadata?: ServerMetadata;
  createdAt: string;
}

export interface ServerMetadata {
  os?: string;
  arch?: string;
  cpuCores?: number;
  memoryMb?: number;
  dockerVersion?: string;
}

export interface ProvisionServerInput {
  name: string;
  hostname: string;
  sshPort: number;
  sshUser: string;
  sshPassword?: string;
  sshKey?: string;
}

export interface TestConnectionResult {
  success: boolean;
  error?: string;
  metadata?: ServerMetadata;
}

export interface ProvisionProgress {
  step: string;
  message: string;
  status: "running" | "done" | "error" | "warning";
  log?: string;
}
```

### API Service

**Adicionar em:** `apps/frontend/src/services/api.ts`

```typescript
servers: {
  list: (): Promise<readonly Server[]> =>
    fetchApiList<Server>(`${API_BASE}/servers`),

  get: (id: string): Promise<Server> =>
    fetchApi<Server>(`${API_BASE}/servers/${id}`),

  delete: (id: string): Promise<void> =>
    fetchApi<void>(`${API_BASE}/servers/${id}`, {
      method: 'DELETE',
    }),

  testConnection: (input: ProvisionServerInput): Promise<TestConnectionResult> =>
    fetchApi<TestConnectionResult>(`${API_BASE}/servers/test-connection`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
}
```

### Hooks

**Arquivo:** `apps/frontend/src/features/servers/hooks/use-servers.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import type { ProvisionServerInput, ProvisionProgress, Server } from "../types";

export function useServers() {
  return useQuery({
    queryKey: ["servers"],
    queryFn: () => api.servers.list(),
  });
}

export function useServer(id: string) {
  return useQuery({
    queryKey: ["servers", id],
    queryFn: () => api.servers.get(id),
    enabled: !!id,
  });
}

export function useDeleteServer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.servers.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["servers"] });
    },
  });
}

export function useTestConnection() {
  return useMutation({
    mutationFn: (input: ProvisionServerInput) =>
      api.servers.testConnection(input),
  });
}

export function useProvisionServer() {
  const queryClient = useQueryClient();

  return {
    provision: async (
      input: ProvisionServerInput,
      onProgress: (progress: ProvisionProgress) => void,
    ): Promise<Server | null> => {
      const response = await fetch(`${API_BASE}/servers/provision`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      });

      const reader = response.body?.getReader();
      if (!reader) throw new Error("No response body");

      const decoder = new TextDecoder();
      let server: Server | null = null;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const text = decoder.decode(value);
        const lines = text.split("\n");

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = JSON.parse(line.slice(6));

            if (data.type === "result") {
              server = data.server;
            } else {
              onProgress(data as ProvisionProgress);
            }
          }
        }
      }

      queryClient.invalidateQueries({ queryKey: ["servers"] });
      return server;
    },
  };
}
```

### Add Server Dialog Component

**Arquivo:** `apps/frontend/src/features/servers/components/add-server-dialog.tsx`

```tsx
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTestConnection, useProvisionServer } from "../hooks/use-servers";
import type {
  ProvisionServerInput,
  ProvisionProgress,
  ServerMetadata,
} from "../types";

interface AddServerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onServerAdded: (serverId: string) => void;
}

type Step = "form" | "provisioning" | "complete";

export function AddServerDialog({
  open,
  onOpenChange,
  onServerAdded,
}: AddServerDialogProps) {
  const [step, setStep] = useState<Step>("form");
  const [authMethod, setAuthMethod] = useState<"password" | "key">("password");

  const [name, setName] = useState("");
  const [hostname, setHostname] = useState("");
  const [sshPort, setSshPort] = useState(22);
  const [sshUser, setSshUser] = useState("root");
  const [sshPassword, setSshPassword] = useState("");
  const [sshKey, setSshKey] = useState("");

  const [testResult, setTestResult] = useState<{
    success: boolean;
    metadata?: ServerMetadata;
    error?: string;
  } | null>(null);
  const [progressLogs, setProgressLogs] = useState<ProvisionProgress[]>([]);
  const [createdServerId, setCreatedServerId] = useState<string | null>(null);

  const testConnection = useTestConnection();
  const { provision } = useProvisionServer();

  const getInput = (): ProvisionServerInput => ({
    name,
    hostname,
    sshPort,
    sshUser,
    sshPassword: authMethod === "password" ? sshPassword : undefined,
    sshKey: authMethod === "key" ? sshKey : undefined,
  });

  const handleTestConnection = async () => {
    setTestResult(null);
    const result = await testConnection.mutateAsync(getInput());
    setTestResult(result);
  };

  const handleProvision = async () => {
    setStep("provisioning");
    setProgressLogs([]);

    const server = await provision(getInput(), (progress) => {
      setProgressLogs((prev) => [...prev, progress]);
    });

    if (server) {
      setCreatedServerId(server.id);
      setStep("complete");
    }
  };

  const handleComplete = () => {
    if (createdServerId) {
      onServerAdded(createdServerId);
    }
    onOpenChange(false);
    resetForm();
  };

  const resetForm = () => {
    setStep("form");
    setName("");
    setHostname("");
    setSshPort(22);
    setSshUser("root");
    setSshPassword("");
    setSshKey("");
    setTestResult(null);
    setProgressLogs([]);
    setCreatedServerId(null);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {step === "form" && "Add New Server"}
            {step === "provisioning" && "Installing FlowDeploy Agent"}
            {step === "complete" && "Server Added Successfully"}
          </DialogTitle>
        </DialogHeader>

        {step === "form" && (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Server Name</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="production-api"
              />
            </div>

            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2 space-y-2">
                <Label htmlFor="hostname">Host / IP</Label>
                <Input
                  id="hostname"
                  value={hostname}
                  onChange={(e) => setHostname(e.target.value)}
                  placeholder="192.168.1.100"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="sshPort">SSH Port</Label>
                <Input
                  id="sshPort"
                  type="number"
                  value={sshPort}
                  onChange={(e) => setSshPort(Number(e.target.value))}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label>SSH Authentication</Label>
              <p className="text-xs text-muted-foreground">
                Used only for agent installation. Credentials are NOT stored.
              </p>

              <Tabs
                value={authMethod}
                onValueChange={(v) => setAuthMethod(v as "password" | "key")}
              >
                <TabsList className="w-full">
                  <TabsTrigger value="password" className="flex-1">
                    Password
                  </TabsTrigger>
                  <TabsTrigger value="key" className="flex-1">
                    SSH Key
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="password" className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="sshUser">Username</Label>
                    <Input
                      id="sshUser"
                      value={sshUser}
                      onChange={(e) => setSshUser(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="sshPassword">Password</Label>
                    <Input
                      id="sshPassword"
                      type="password"
                      value={sshPassword}
                      onChange={(e) => setSshPassword(e.target.value)}
                    />
                  </div>
                </TabsContent>

                <TabsContent value="key" className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="sshUserKey">Username</Label>
                    <Input
                      id="sshUserKey"
                      value={sshUser}
                      onChange={(e) => setSshUser(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="sshKey">Private Key</Label>
                    <textarea
                      id="sshKey"
                      className="w-full h-32 p-2 text-sm font-mono border rounded-md"
                      value={sshKey}
                      onChange={(e) => setSshKey(e.target.value)}
                      placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                    />
                  </div>
                </TabsContent>
              </Tabs>
            </div>

            <Button
              variant="outline"
              className="w-full"
              onClick={handleTestConnection}
              disabled={!hostname || !sshUser || testConnection.isPending}
            >
              {testConnection.isPending ? "Testing..." : "Test Connection"}
            </Button>

            {testResult && (
              <div
                className={`p-3 rounded-md text-sm ${testResult.success ? "bg-green-50 text-green-800" : "bg-red-50 text-red-800"}`}
              >
                {testResult.success ? (
                  <div className="space-y-1">
                    <p className="font-medium">Connection successful</p>
                    {testResult.metadata && (
                      <>
                        <p>OS: {testResult.metadata.os}</p>
                        <p>Docker: {testResult.metadata.dockerVersion}</p>
                        <p>CPU: {testResult.metadata.cpuCores} cores</p>
                      </>
                    )}
                  </div>
                ) : (
                  <p>{testResult.error}</p>
                )}
              </div>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleProvision}
                disabled={!testResult?.success || !name}
              >
                Install Agent & Add Server
              </Button>
            </div>
          </div>
        )}

        {step === "provisioning" && (
          <div className="space-y-4">
            <div className="space-y-2">
              {progressLogs.map((log, i) => (
                <div key={i} className="flex items-center gap-2 text-sm">
                  {log.status === "running" && (
                    <span className="animate-spin">⏳</span>
                  )}
                  {log.status === "done" && (
                    <span className="text-green-600">✓</span>
                  )}
                  {log.status === "error" && (
                    <span className="text-red-600">✗</span>
                  )}
                  {log.status === "warning" && (
                    <span className="text-yellow-600">⚠</span>
                  )}
                  <span>{log.message}</span>
                </div>
              ))}
            </div>

            <div className="bg-muted p-2 rounded-md max-h-48 overflow-y-auto">
              <pre className="text-xs font-mono whitespace-pre-wrap">
                {progressLogs
                  .filter((l) => l.log)
                  .map((l) => l.log)
                  .join("\n")}
              </pre>
            </div>
          </div>
        )}

        {step === "complete" && (
          <div className="space-y-4">
            <div className="text-center py-4">
              <div className="text-4xl mb-2">✓</div>
              <p className="font-medium">Server added successfully!</p>
              <p className="text-sm text-muted-foreground">
                The FlowDeploy Agent is now running and connected.
              </p>
            </div>

            <p className="text-xs text-muted-foreground text-center">
              SSH credentials have been discarded. All future communication will
              be through the secure agent connection.
            </p>

            <Button className="w-full" onClick={handleComplete}>
              Continue
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
```

### Server Selector Component (for Onboarding)

**Arquivo:** `apps/frontend/src/features/servers/components/server-selector.tsx`

```tsx
import { useState } from "react";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { useServers } from "../hooks/use-servers";
import { AddServerDialog } from "./add-server-dialog";
import type { DeployMode } from "@/features/apps/types";

interface ServerSelectorProps {
  deployMode: DeployMode;
  serverId: string | null;
  onDeployModeChange: (mode: DeployMode) => void;
  onServerChange: (serverId: string | null) => void;
}

export function ServerSelector({
  deployMode,
  serverId,
  onDeployModeChange,
  onServerChange,
}: ServerSelectorProps) {
  const [showAddServer, setShowAddServer] = useState(false);
  const { data: servers, isLoading } = useServers();

  const handleServerAdded = (newServerId: string) => {
    onServerChange(newServerId);
    setShowAddServer(false);
  };

  return (
    <div className="space-y-4">
      <Label className="text-base font-medium">Deployment Target</Label>

      <RadioGroup
        value={deployMode}
        onValueChange={(v) => {
          onDeployModeChange(v as DeployMode);
          if (v === "local") {
            onServerChange(null);
          }
        }}
      >
        <div className="flex items-start space-x-3 p-4 border rounded-lg">
          <RadioGroupItem value="local" id="local" className="mt-1" />
          <div>
            <Label htmlFor="local" className="font-medium cursor-pointer">
              Deploy Local
            </Label>
            <p className="text-sm text-muted-foreground">
              Deploy on the same server where FlowDeploy is running
            </p>
          </div>
        </div>

        <div className="flex items-start space-x-3 p-4 border rounded-lg">
          <RadioGroupItem value="remote" id="remote" className="mt-1" />
          <div className="flex-1">
            <Label htmlFor="remote" className="font-medium cursor-pointer">
              Deploy Remote
            </Label>
            <p className="text-sm text-muted-foreground mb-3">
              Deploy on another server with FlowDeploy Agent
            </p>

            {deployMode === "remote" && (
              <div className="space-y-3">
                {isLoading ? (
                  <p className="text-sm text-muted-foreground">
                    Loading servers...
                  </p>
                ) : servers && servers.length > 0 ? (
                  <RadioGroup
                    value={serverId ?? ""}
                    onValueChange={onServerChange}
                    className="space-y-2"
                  >
                    {servers.map((server) => (
                      <div
                        key={server.id}
                        className="flex items-center justify-between p-3 border rounded-md"
                      >
                        <div className="flex items-center gap-3">
                          <RadioGroupItem value={server.id} id={server.id} />
                          <Label htmlFor={server.id} className="cursor-pointer">
                            <span className="font-medium">{server.name}</span>
                            <span className="text-muted-foreground ml-2">
                              {server.hostname}
                            </span>
                          </Label>
                        </div>
                        <span
                          className={`text-xs px-2 py-1 rounded-full ${
                            server.status === "online"
                              ? "bg-green-100 text-green-800"
                              : "bg-gray-100 text-gray-800"
                          }`}
                        >
                          {server.status}
                        </span>
                      </div>
                    ))}
                  </RadioGroup>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No servers configured
                  </p>
                )}

                <Button
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={() => setShowAddServer(true)}
                >
                  + Add New Server
                </Button>
              </div>
            )}
          </div>
        </div>
      </RadioGroup>

      <AddServerDialog
        open={showAddServer}
        onOpenChange={setShowAddServer}
        onServerAdded={handleServerAdded}
      />
    </div>
  );
}
```

## Integracao no Onboarding Wizard

**Modificar:** `apps/frontend/src/features/apps/components/onboarding/onboarding-wizard.tsx`

```tsx
import { ServerSelector } from "@/features/servers/components/server-selector";

// Adicionar estado
const [deployMode, setDeployMode] = useState<DeployMode>("local");
const [serverId, setServerId] = useState<string | null>(null);

// No step de Server Selection
<ServerSelector
  deployMode={deployMode}
  serverId={serverId}
  onDeployModeChange={setDeployMode}
  onServerChange={setServerId}
/>;

// Ao criar o app, incluir deployMode e serverId
const app = await createApp.mutateAsync({
  name,
  repositoryUrl,
  branch,
  workdir: workdir || undefined,
  deployMode,
  serverId: deployMode === "remote" ? serverId : undefined,
});
```

## Estrutura de Arquivos Final

```
apps/
├── agent/
│   ├── cmd/
│   │   └── agent/
│   │       └── main.go
│   ├── internal/
│   │   └── agent/
│   │       └── agent.go
│   ├── install.sh
│   ├── Dockerfile
│   └── go.mod
├── backend/
│   ├── internal/
│   │   ├── agent/
│   │   │   └── hub.go
│   │   ├── domain/
│   │   │   ├── app.go (modificado)
│   │   │   └── server.go (novo)
│   │   ├── handler/
│   │   │   └── server_handler.go (novo)
│   │   ├── provisioner/
│   │   │   └── ssh_provisioner.go (novo)
│   │   └── repository/
│   │       └── server_repository.go (novo)
│   └── migrations/
│       ├── 000006_add_servers.up.sql (novo)
│       └── 000006_add_servers.down.sql (novo)
└── frontend/
    └── src/
        └── features/
            ├── apps/
            │   └── components/
            │       └── onboarding/
            │           └── onboarding-wizard.tsx (modificado)
            └── servers/ (novo)
                ├── components/
                │   ├── add-server-dialog.tsx
                │   ├── server-card.tsx
                │   ├── server-list.tsx
                │   └── server-selector.tsx
                ├── hooks/
                │   └── use-servers.ts
                └── types.ts
```

## Resumo do Fluxo

1. **Usuario inicia onboarding** de um novo app
2. **Step 1:** Configura repositorio (URL, branch, workdir)
3. **Step 2:** Seleciona target de deploy
   - Se **Local**: continua normalmente
   - Se **Remote**: seleciona servidor existente OU adiciona novo
4. **Adicionar novo servidor:**
   - Informa nome, host, porta SSH
   - Escolhe autenticacao (senha OU chave SSH)
   - Testa conexao
   - Clica "Install Agent & Add Server"
5. **Provisionamento automatico:**
   - FlowDeploy conecta via SSH
   - Verifica Docker e Git
   - Download e instala o agent
   - Configura systemd
   - Inicia o agent
   - Agent conecta via WebSocket
6. **SSH encerrado**, credenciais descartadas
7. **Servidor aparece como Online**
8. **Step 3:** Configura variaveis de ambiente
9. **Step 4:** Review e deploy

## Consideracoes de Seguranca

1. **Credenciais SSH descartadas:** Nunca armazenadas, usadas apenas para provisionamento
2. **Token do agent:** Gerado aleatoriamente, unico por servidor
3. **Comunicacao agent-server:** Via WebSocket com autenticacao por token
4. **Conexao outbound:** Agent inicia conexao, nenhuma porta precisa ser aberta no servidor remoto
5. **Timeout:** Conexao SSH tem timeout de 30 segundos
6. **Validacao:** Host/IP validados antes da conexao

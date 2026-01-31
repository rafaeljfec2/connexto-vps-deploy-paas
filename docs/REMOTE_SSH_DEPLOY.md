# Arquitetura de Deploy Remoto com Agent

## Visao Geral

O FlowDeploy suporta deploy de aplicacoes em servidores remotos atraves de um **Agent leve** instalado em cada servidor. Esta arquitetura oferece maior seguranca, confiabilidade e visibilidade centralizada.

## Por que Agent ao inves de SSH Direto?

| Aspecto     | SSH Direto                    | Agent                              |
| ----------- | ----------------------------- | ---------------------------------- |
| Seguranca   | Chaves SSH no banco           | Token de autenticacao              |
| Conexao     | Inbound (requer porta aberta) | Outbound (agent conecta ao server) |
| Firewall    | Precisa abrir porta 22        | Nenhuma porta a abrir              |
| Logs        | Dificil de coletar            | Streaming em tempo real            |
| Status      | Sem visibilidade              | Heartbeat continuo                 |
| Atualizacao | Manual                        | Auto-update                        |

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

## Componentes

### 1. FlowDeploy Server (Backend)

Responsabilidades:

- Gerenciar registro de agents
- Enviar comandos de deploy para agents
- Receber logs e metricas em tempo real
- Armazenar estado dos servidores e apps

### 2. FlowDeploy Agent

Binario Go leve (~10MB) que roda em cada servidor remoto.

Responsabilidades:

- Manter conexao WebSocket com o server
- Executar comandos de deploy (git, docker)
- Enviar logs em tempo real
- Reportar metricas (CPU, memoria, containers)
- Executar health checks locais

## Banco de Dados

### Migration 000006 - Servidores

```sql
-- Tabela de servidores (hosts remotos)
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    agent_token VARCHAR(64) NOT NULL UNIQUE,
    agent_version VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_heartbeat_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE server_status AS ENUM ('pending', 'online', 'offline', 'error');

CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_agent_token ON servers(agent_token);

-- Adicionar referencia de servidor na tabela apps
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

## Domain Layer

### Server Domain

**Arquivo:** `apps/backend/internal/domain/server.go`

```go
package domain

import (
    "encoding/json"
    "time"
)

type ServerStatus string

const (
    ServerStatusPending ServerStatus = "pending"
    ServerStatusOnline  ServerStatus = "online"
    ServerStatusOffline ServerStatus = "offline"
    ServerStatusError   ServerStatus = "error"
)

type Server struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    Hostname        string          `json:"hostname"`
    IPAddress       *string         `json:"ipAddress,omitempty"`
    AgentToken      string          `json:"-"`
    AgentVersion    *string         `json:"agentVersion,omitempty"`
    Status          ServerStatus    `json:"status"`
    LastHeartbeatAt *time.Time      `json:"lastHeartbeatAt,omitempty"`
    Metadata        json.RawMessage `json:"metadata,omitempty"`
    CreatedAt       time.Time       `json:"createdAt"`
    UpdatedAt       time.Time       `json:"updatedAt"`
}

type ServerMetadata struct {
    OS           string `json:"os,omitempty"`
    Architecture string `json:"arch,omitempty"`
    CPUCores     int    `json:"cpuCores,omitempty"`
    MemoryMB     int    `json:"memoryMb,omitempty"`
    DockerVersion string `json:"dockerVersion,omitempty"`
}

type CreateServerInput struct {
    Name     string `json:"name" validate:"required,min=2,max=63"`
    Hostname string `json:"hostname" validate:"required"`
}

type ServerWithToken struct {
    Server
    AgentToken string `json:"agentToken"`
}
```

### App Domain (Atualizado)

**Arquivo:** `apps/backend/internal/domain/app.go`

```go
type DeployMode string

const (
    DeployModeLocal  DeployMode = "local"
    DeployModeRemote DeployMode = "remote"
)

type App struct {
    ID             string          `json:"id"`
    Name           string          `json:"name"`
    RepositoryURL  string          `json:"repositoryUrl"`
    Branch         string          `json:"branch"`
    Workdir        string          `json:"workdir"`
    Runtime        *string         `json:"runtime,omitempty"`
    Config         json.RawMessage `json:"config"`
    Status         AppStatus       `json:"status"`
    WebhookID      *int64          `json:"webhookId,omitempty"`
    DeployMode     DeployMode      `json:"deployMode"`
    ServerID       *string         `json:"serverId,omitempty"`
    LastDeployedAt *time.Time      `json:"lastDeployedAt,omitempty"`
    CreatedAt      time.Time       `json:"createdAt"`
    UpdatedAt      time.Time       `json:"updatedAt"`
}
```

## Agent Implementation

### Agent Main

**Arquivo:** `apps/agent/cmd/agent/main.go`

```go
package main

import (
    "context"
    "flag"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "flowdeploy/agent/internal/agent"
)

func main() {
    var (
        serverURL  = flag.String("server", "", "FlowDeploy server URL (required)")
        token      = flag.String("token", "", "Agent token (required)")
        dataDir    = flag.String("data-dir", "/opt/flowdeploy", "Data directory")
    )
    flag.Parse()

    if *serverURL == "" || *token == "" {
        slog.Error("server and token are required")
        os.Exit(1)
    }

    cfg := agent.Config{
        ServerURL: *serverURL,
        Token:     *token,
        DataDir:   *dataDir,
    }

    a, err := agent.New(cfg)
    if err != nil {
        slog.Error("failed to create agent", "error", err)
        os.Exit(1)
    }

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    if err := a.Run(ctx); err != nil {
        slog.Error("agent error", "error", err)
        os.Exit(1)
    }
}
```

### Agent Core

**Arquivo:** `apps/agent/internal/agent/agent.go`

```go
package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/exec"
    "runtime"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

type Config struct {
    ServerURL string
    Token     string
    DataDir   string
}

type Agent struct {
    cfg    Config
    conn   *websocket.Conn
    mu     sync.Mutex
    logger *slog.Logger
}

type Message struct {
    Type    string          `json:"type"`
    ID      string          `json:"id,omitempty"`
    Payload json.RawMessage `json:"payload,omitempty"`
}

type DeployCommand struct {
    AppID         string            `json:"appId"`
    DeploymentID  string            `json:"deploymentId"`
    RepositoryURL string            `json:"repositoryUrl"`
    Branch        string            `json:"branch"`
    Workdir       string            `json:"workdir"`
    CommitSHA     string            `json:"commitSha,omitempty"`
    EnvVars       map[string]string `json:"envVars,omitempty"`
}

func New(cfg Config) (*Agent, error) {
    if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create data dir: %w", err)
    }

    return &Agent{
        cfg:    cfg,
        logger: slog.Default(),
    }, nil
}

func (a *Agent) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        if err := a.connect(ctx); err != nil {
            a.logger.Error("connection failed", "error", err)
            time.Sleep(5 * time.Second)
            continue
        }

        if err := a.listen(ctx); err != nil {
            a.logger.Error("listen error", "error", err)
        }

        time.Sleep(time.Second)
    }
}

func (a *Agent) connect(ctx context.Context) error {
    wsURL := fmt.Sprintf("%s/agent/ws", a.cfg.ServerURL)

    header := http.Header{}
    header.Set("Authorization", "Bearer "+a.cfg.Token)

    conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
    if err != nil {
        return fmt.Errorf("websocket dial failed: %w", err)
    }

    a.mu.Lock()
    a.conn = conn
    a.mu.Unlock()

    // Enviar info do sistema
    if err := a.sendSystemInfo(); err != nil {
        return fmt.Errorf("failed to send system info: %w", err)
    }

    a.logger.Info("connected to server")
    return nil
}

func (a *Agent) listen(ctx context.Context) error {
    // Heartbeat goroutine
    go a.heartbeat(ctx)

    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        _, data, err := a.conn.ReadMessage()
        if err != nil {
            return fmt.Errorf("read message failed: %w", err)
        }

        var msg Message
        if err := json.Unmarshal(data, &msg); err != nil {
            a.logger.Error("failed to unmarshal message", "error", err)
            continue
        }

        go a.handleMessage(ctx, msg)
    }
}

func (a *Agent) handleMessage(ctx context.Context, msg Message) {
    switch msg.Type {
    case "deploy":
        var cmd DeployCommand
        if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
            a.logger.Error("failed to unmarshal deploy command", "error", err)
            return
        }
        a.handleDeploy(ctx, msg.ID, cmd)

    case "logs":
        a.handleLogs(ctx, msg)

    case "exec":
        a.handleExec(ctx, msg)

    default:
        a.logger.Warn("unknown message type", "type", msg.Type)
    }
}

func (a *Agent) handleDeploy(ctx context.Context, msgID string, cmd DeployCommand) {
    a.logger.Info("starting deploy", "appId", cmd.AppID, "deploymentId", cmd.DeploymentID)

    appDir := fmt.Sprintf("%s/apps/%s", a.cfg.DataDir, cmd.AppID)
    repoDir := fmt.Sprintf("%s/repo", appDir)

    // 1. Git clone/pull
    a.sendLog(cmd.DeploymentID, "info", "Syncing repository...")
    if err := a.syncRepo(ctx, cmd, repoDir); err != nil {
        a.sendDeployResult(msgID, cmd.DeploymentID, false, err.Error())
        return
    }

    // 2. Docker build
    a.sendLog(cmd.DeploymentID, "info", "Building Docker image...")
    imageName := fmt.Sprintf("flowdeploy-%s:latest", cmd.AppID)
    if err := a.dockerBuild(ctx, cmd, repoDir, imageName); err != nil {
        a.sendDeployResult(msgID, cmd.DeploymentID, false, err.Error())
        return
    }

    // 3. Docker run
    a.sendLog(cmd.DeploymentID, "info", "Starting container...")
    if err := a.dockerRun(ctx, cmd, imageName); err != nil {
        a.sendDeployResult(msgID, cmd.DeploymentID, false, err.Error())
        return
    }

    a.sendLog(cmd.DeploymentID, "info", "Deploy completed successfully")
    a.sendDeployResult(msgID, cmd.DeploymentID, true, "")
}

func (a *Agent) syncRepo(ctx context.Context, cmd DeployCommand, repoDir string) error {
    if _, err := os.Stat(repoDir + "/.git"); os.IsNotExist(err) {
        // Clone
        args := []string{"clone", "--branch", cmd.Branch, "--single-branch", cmd.RepositoryURL, repoDir}
        return a.runCommand(ctx, cmd.DeploymentID, "git", args...)
    }

    // Fetch and checkout
    if err := a.runCommand(ctx, cmd.DeploymentID, "git", "-C", repoDir, "fetch", "origin"); err != nil {
        return err
    }

    ref := cmd.CommitSHA
    if ref == "" {
        ref = "origin/" + cmd.Branch
    }

    return a.runCommand(ctx, cmd.DeploymentID, "git", "-C", repoDir, "checkout", ref)
}

func (a *Agent) dockerBuild(ctx context.Context, cmd DeployCommand, repoDir, imageName string) error {
    buildContext := repoDir
    if cmd.Workdir != "" && cmd.Workdir != "." {
        buildContext = fmt.Sprintf("%s/%s", repoDir, cmd.Workdir)
    }

    args := []string{"build", "-t", imageName, buildContext}
    return a.runCommand(ctx, cmd.DeploymentID, "docker", args...)
}

func (a *Agent) dockerRun(ctx context.Context, cmd DeployCommand, imageName string) error {
    containerName := fmt.Sprintf("flowdeploy-%s", cmd.AppID)

    // Parar container existente
    _ = a.runCommand(ctx, cmd.DeploymentID, "docker", "stop", containerName)
    _ = a.runCommand(ctx, cmd.DeploymentID, "docker", "rm", containerName)

    args := []string{"run", "-d", "--name", containerName, "--restart", "unless-stopped"}

    // Adicionar env vars
    for k, v := range cmd.EnvVars {
        args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
    }

    args = append(args, imageName)
    return a.runCommand(ctx, cmd.DeploymentID, "docker", args...)
}

func (a *Agent) runCommand(ctx context.Context, deploymentID, name string, args ...string) error {
    cmd := exec.CommandContext(ctx, name, args...)

    output, err := cmd.CombinedOutput()

    // Enviar output como log
    if len(output) > 0 {
        a.sendLog(deploymentID, "output", string(output))
    }

    if err != nil {
        return fmt.Errorf("%s failed: %w", name, err)
    }

    return nil
}

func (a *Agent) sendLog(deploymentID, level, message string) {
    a.send(Message{
        Type: "log",
        Payload: mustMarshal(map[string]string{
            "deploymentId": deploymentID,
            "level":        level,
            "message":      message,
            "timestamp":    time.Now().Format(time.RFC3339),
        }),
    })
}

func (a *Agent) sendDeployResult(msgID, deploymentID string, success bool, errorMsg string) {
    a.send(Message{
        Type: "deploy_result",
        ID:   msgID,
        Payload: mustMarshal(map[string]interface{}{
            "deploymentId": deploymentID,
            "success":      success,
            "error":        errorMsg,
        }),
    })
}

func (a *Agent) sendSystemInfo() error {
    info := map[string]interface{}{
        "os":           runtime.GOOS,
        "arch":         runtime.GOARCH,
        "cpuCores":     runtime.NumCPU(),
        "agentVersion": "1.0.0",
    }

    // Get Docker version
    if output, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output(); err == nil {
        info["dockerVersion"] = string(output)
    }

    return a.send(Message{
        Type:    "system_info",
        Payload: mustMarshal(info),
    })
}

func (a *Agent) heartbeat(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            a.send(Message{Type: "heartbeat"})
        }
    }
}

func (a *Agent) send(msg Message) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.conn == nil {
        return fmt.Errorf("not connected")
    }

    return a.conn.WriteJSON(msg)
}

func mustMarshal(v interface{}) json.RawMessage {
    data, _ := json.Marshal(v)
    return data
}
```

## Backend - Agent Hub

**Arquivo:** `apps/backend/internal/agent/hub.go`

```go
package agent

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "flowdeploy/internal/domain"
    "flowdeploy/internal/repository"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
    serverRepo *repository.ServerRepository
    agents     map[string]*AgentConn
    mu         sync.RWMutex
    logger     *slog.Logger
}

type AgentConn struct {
    ServerID string
    Conn     *websocket.Conn
    Send     chan []byte
}

type Message struct {
    Type    string          `json:"type"`
    ID      string          `json:"id,omitempty"`
    Payload json.RawMessage `json:"payload,omitempty"`
}

func NewHub(serverRepo *repository.ServerRepository) *Hub {
    return &Hub{
        serverRepo: serverRepo,
        agents:     make(map[string]*AgentConn),
        logger:     slog.Default(),
    }
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Extrair token do header
    token := r.Header.Get("Authorization")
    if token == "" {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Remover "Bearer " prefix
    if len(token) > 7 && token[:7] == "Bearer " {
        token = token[7:]
    }

    // Buscar servidor pelo token
    server, err := h.serverRepo.GetByToken(r.Context(), token)
    if err != nil {
        http.Error(w, "invalid token", http.StatusUnauthorized)
        return
    }

    // Upgrade para WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("websocket upgrade failed", "error", err)
        return
    }

    agent := &AgentConn{
        ServerID: server.ID,
        Conn:     conn,
        Send:     make(chan []byte, 256),
    }

    h.register(agent)
    defer h.unregister(agent)

    // Atualizar status do servidor
    _ = h.serverRepo.UpdateStatus(r.Context(), server.ID, domain.ServerStatusOnline)

    go h.writePump(agent)
    h.readPump(r.Context(), agent)
}

func (h *Hub) register(agent *AgentConn) {
    h.mu.Lock()
    defer h.mu.Unlock()

    // Fechar conexao anterior se existir
    if old, ok := h.agents[agent.ServerID]; ok {
        old.Conn.Close()
    }

    h.agents[agent.ServerID] = agent
    h.logger.Info("agent connected", "serverId", agent.ServerID)
}

func (h *Hub) unregister(agent *AgentConn) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if current, ok := h.agents[agent.ServerID]; ok && current == agent {
        delete(h.agents, agent.ServerID)
        close(agent.Send)
        agent.Conn.Close()

        // Atualizar status para offline
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = h.serverRepo.UpdateStatus(ctx, agent.ServerID, domain.ServerStatusOffline)

        h.logger.Info("agent disconnected", "serverId", agent.ServerID)
    }
}

func (h *Hub) readPump(ctx context.Context, agent *AgentConn) {
    defer agent.Conn.Close()

    for {
        _, data, err := agent.Conn.ReadMessage()
        if err != nil {
            return
        }

        var msg Message
        if err := json.Unmarshal(data, &msg); err != nil {
            continue
        }

        h.handleMessage(ctx, agent.ServerID, msg)
    }
}

func (h *Hub) writePump(agent *AgentConn) {
    for data := range agent.Send {
        if err := agent.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
            return
        }
    }
}

func (h *Hub) handleMessage(ctx context.Context, serverID string, msg Message) {
    switch msg.Type {
    case "heartbeat":
        _ = h.serverRepo.UpdateHeartbeat(ctx, serverID)

    case "system_info":
        var info domain.ServerMetadata
        _ = json.Unmarshal(msg.Payload, &info)
        _ = h.serverRepo.UpdateMetadata(ctx, serverID, info)

    case "log":
        // Encaminhar log para SSE
        h.logger.Debug("deploy log", "serverId", serverID, "payload", string(msg.Payload))

    case "deploy_result":
        // Processar resultado do deploy
        h.logger.Info("deploy result", "serverId", serverID, "payload", string(msg.Payload))
    }
}

func (h *Hub) SendToServer(serverID string, msg Message) error {
    h.mu.RLock()
    agent, ok := h.agents[serverID]
    h.mu.RUnlock()

    if !ok {
        return fmt.Errorf("server not connected: %s", serverID)
    }

    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    select {
    case agent.Send <- data:
        return nil
    default:
        return fmt.Errorf("send buffer full")
    }
}

func (h *Hub) IsServerOnline(serverID string) bool {
    h.mu.RLock()
    defer h.mu.RUnlock()
    _, ok := h.agents[serverID]
    return ok
}
```

## API Endpoints

### Server Handler

**Arquivo:** `apps/backend/internal/handler/server_handler.go`

```go
package handler

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"

    "github.com/go-chi/chi/v5"
    "flowdeploy/internal/domain"
    "flowdeploy/internal/repository"
    "flowdeploy/internal/response"
    "flowdeploy/internal/agent"
)

type ServerHandler struct {
    repo *repository.ServerRepository
    hub  *agent.Hub
}

// POST /api/servers - Criar servidor e gerar token
func (h *ServerHandler) Create(w http.ResponseWriter, r *http.Request) {
    var input domain.CreateServerInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // Gerar token unico
    token := generateToken()

    server, err := h.repo.Create(r.Context(), input, token)
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to create server")
        return
    }

    // Retornar servidor com token (unica vez que o token e exposto)
    result := domain.ServerWithToken{
        Server:     *server,
        AgentToken: token,
    }

    response.JSON(w, http.StatusCreated, result)
}

// GET /api/servers - Listar servidores
func (h *ServerHandler) List(w http.ResponseWriter, r *http.Request) {
    servers, err := h.repo.List(r.Context())
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to list servers")
        return
    }

    // Adicionar status online/offline baseado no hub
    for i := range servers {
        if h.hub.IsServerOnline(servers[i].ID) {
            servers[i].Status = domain.ServerStatusOnline
        }
    }

    response.JSON(w, http.StatusOK, servers)
}

// GET /api/servers/:id - Detalhes do servidor
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

// DELETE /api/servers/:id - Remover servidor
func (h *ServerHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    if err := h.repo.Delete(r.Context(), id); err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to delete server")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// POST /api/servers/:id/regenerate-token - Gerar novo token
func (h *ServerHandler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    newToken := generateToken()

    if err := h.repo.UpdateToken(r.Context(), id, newToken); err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to regenerate token")
        return
    }

    response.JSON(w, http.StatusOK, map[string]string{
        "token": newToken,
    })
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
// Adicionar rotas de servidores
r.Route("/servers", func(r chi.Router) {
    r.Get("/", serverHandler.List)
    r.Post("/", serverHandler.Create)
    r.Get("/{id}", serverHandler.Get)
    r.Delete("/{id}", serverHandler.Delete)
    r.Post("/{id}/regenerate-token", serverHandler.RegenerateToken)
})

// WebSocket para agents
r.Get("/agent/ws", agentHub.HandleWebSocket)
```

## Frontend

### Types

```typescript
export type ServerStatus = "pending" | "online" | "offline" | "error";

export interface Server {
  id: string;
  name: string;
  hostname: string;
  ipAddress?: string;
  agentVersion?: string;
  status: ServerStatus;
  lastHeartbeatAt?: string;
  metadata?: {
    os?: string;
    arch?: string;
    cpuCores?: number;
    memoryMb?: number;
    dockerVersion?: string;
  };
  createdAt: string;
}

export interface ServerWithToken extends Server {
  agentToken: string;
}

export interface CreateServerInput {
  name: string;
  hostname: string;
}

export type DeployMode = "local" | "remote";

export interface CreateAppInput {
  name: string;
  repositoryUrl: string;
  branch?: string;
  workdir?: string;
  deployMode?: DeployMode;
  serverId?: string;
}
```

### API Service

```typescript
servers: {
  list: (): Promise<readonly Server[]> =>
    fetchApiList<Server>(`${API_BASE}/servers`),

  create: (input: CreateServerInput): Promise<ServerWithToken> =>
    fetchApi<ServerWithToken>(`${API_BASE}/servers`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  get: (id: string): Promise<Server> =>
    fetchApi<Server>(`${API_BASE}/servers/${id}`),

  delete: (id: string): Promise<void> =>
    fetchApi<void>(`${API_BASE}/servers/${id}`, {
      method: 'DELETE',
    }),

  regenerateToken: (id: string): Promise<{ token: string }> =>
    fetchApi<{ token: string }>(`${API_BASE}/servers/${id}/regenerate-token`, {
      method: 'POST',
    }),
}
```

## Fluxo de Setup de Servidor

```
1. Usuario acessa FlowDeploy Dashboard
2. Usuario vai em "Servidores" > "Adicionar Servidor"
3. Usuario informa nome e hostname
4. Sistema gera token unico e exibe comando de instalacao:

   curl -fsSL https://flowdeploy.io/install.sh | sh -s -- \
     --server https://sua-instancia.flowdeploy.io \
     --token abc123xyz...

5. Usuario executa comando no servidor remoto
6. Agent conecta ao FlowDeploy Server via WebSocket
7. Servidor aparece como "Online" no dashboard
```

## Fluxo de Deploy Remoto

```
1. Usuario cria app com deployMode = "remote" e seleciona servidor
2. Usuario inicia deploy
3. Backend envia comando de deploy via WebSocket para o agent
4. Agent executa:
   a. git clone/pull
   b. docker build
   c. docker run
5. Agent envia logs em tempo real via WebSocket
6. Agent envia resultado final (success/failure)
7. Backend atualiza status do deployment
8. Frontend recebe atualizacao via SSE
```

## Instalacao do Agent

### Script de Instalacao

**Arquivo:** `apps/agent/install.sh`

```bash
#!/bin/bash
set -e

FLOWDEPLOY_SERVER=""
FLOWDEPLOY_TOKEN=""
INSTALL_DIR="/opt/flowdeploy"
SERVICE_NAME="flowdeploy-agent"

while [[ $# -gt 0 ]]; do
    case $1 in
        --server) FLOWDEPLOY_SERVER="$2"; shift 2 ;;
        --token) FLOWDEPLOY_TOKEN="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ -z "$FLOWDEPLOY_SERVER" || -z "$FLOWDEPLOY_TOKEN" ]]; then
    echo "Usage: install.sh --server <url> --token <token>"
    exit 1
fi

echo "Installing FlowDeploy Agent..."

# Criar diretorio
sudo mkdir -p "$INSTALL_DIR"

# Detectar arquitetura
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download do binario
DOWNLOAD_URL="${FLOWDEPLOY_SERVER}/downloads/agent-linux-${ARCH}"
sudo curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/flowdeploy-agent"
sudo chmod +x "$INSTALL_DIR/flowdeploy-agent"

# Criar arquivo de configuracao
sudo tee "$INSTALL_DIR/config.env" > /dev/null <<EOF
FLOWDEPLOY_SERVER=$FLOWDEPLOY_SERVER
FLOWDEPLOY_TOKEN=$FLOWDEPLOY_TOKEN
EOF
sudo chmod 600 "$INSTALL_DIR/config.env"

# Criar systemd service
sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" > /dev/null <<EOF
[Unit]
Description=FlowDeploy Agent
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
EnvironmentFile=$INSTALL_DIR/config.env
ExecStart=$INSTALL_DIR/flowdeploy-agent --server \$FLOWDEPLOY_SERVER --token \$FLOWDEPLOY_TOKEN
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Iniciar servico
sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE_NAME"
sudo systemctl start "$SERVICE_NAME"

echo "FlowDeploy Agent installed successfully!"
echo "Check status: sudo systemctl status $SERVICE_NAME"
```

## Pre-requisitos do Servidor Remoto

- Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)
- Docker Engine 20+
- Git
- Acesso a internet (outbound) para conectar ao FlowDeploy Server

## Estrutura de Arquivos

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
│   │   │   └── server.go
│   │   ├── handler/
│   │   │   └── server_handler.go
│   │   └── repository/
│   │       └── server_repository.go
│   └── migrations/
│       ├── 000006_add_servers.up.sql
│       └── 000006_add_servers.down.sql
└── frontend/
    └── src/
        └── features/
            └── servers/
                ├── components/
                │   ├── server-list.tsx
                │   ├── server-card.tsx
                │   └── add-server-dialog.tsx
                ├── hooks/
                │   └── use-servers.ts
                └── types.ts
```

## Proximos Passos

1. Criar migration para tabela servers
2. Implementar domain e repository de Server
3. Implementar Agent Hub (WebSocket server)
4. Implementar handlers de Server
5. Criar projeto do Agent (Go)
6. Implementar comunicacao WebSocket
7. Criar script de instalacao
8. Implementar frontend (servidores)
9. Integrar deploy remoto no worker
10. Testes de integracao

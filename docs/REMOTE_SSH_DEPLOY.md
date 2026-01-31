# Implementacao de Deploy Remoto via SSH

## Visao Geral

Esta feature permite que o FlowDeploy faca deploy de aplicacoes em servidores remotos via SSH, ao inves de apenas no servidor local onde o FlowDeploy esta rodando.

### Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                    FlowDeploy (Servidor Cloud)                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Frontend │  │ Backend  │  │PostgreSQL│  │ Traefik  │        │
│  │  React   │  │   Go     │  │          │  │          │        │
│  └────┬─────┘  └────┬─────┘  └──────────┘  └──────────┘        │
│       │             │                                           │
└───────┼─────────────┼───────────────────────────────────────────┘
        │             │
        │             │ SSH
        │             ▼
┌───────┼─────────────────────────────────────────────────────────┐
│       │        Servidor Remoto 1                                │
│       │   ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│       │   │ Docker   │  │  App A   │  │  App B   │             │
│       │   │ Engine   │  │          │  │          │             │
│       │   └──────────┘  └──────────┘  └──────────┘             │
└───────┼─────────────────────────────────────────────────────────┘
        │             │
        │             │ SSH
        │             ▼
┌───────┼─────────────────────────────────────────────────────────┐
│       │        Servidor Remoto 2                                │
│       │   ┌──────────┐  ┌──────────┐                           │
│       │   │ Docker   │  │  App C   │                           │
│       │   │ Engine   │  │          │                           │
│       │   └──────────┘  └──────────┘                           │
└─────────────────────────────────────────────────────────────────┘
```

## Mudancas no Banco de Dados

### Migration 000006

**Arquivo:** `apps/backend/migrations/000006_add_ssh_target.up.sql`

```sql
-- Tabela para armazenar chaves SSH
CREATE TABLE ssh_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    private_key TEXT NOT NULL,
    public_key TEXT,
    fingerprint VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ssh_keys_name ON ssh_keys(name);

-- Adicionar campos SSH na tabela apps
ALTER TABLE apps ADD COLUMN deploy_mode VARCHAR(20) NOT NULL DEFAULT 'local';
ALTER TABLE apps ADD COLUMN ssh_host VARCHAR(255);
ALTER TABLE apps ADD COLUMN ssh_port INTEGER DEFAULT 22;
ALTER TABLE apps ADD COLUMN ssh_user VARCHAR(100) DEFAULT 'root';
ALTER TABLE apps ADD COLUMN ssh_key_id UUID REFERENCES ssh_keys(id) ON DELETE SET NULL;

CREATE INDEX idx_apps_deploy_mode ON apps(deploy_mode);
```

**Arquivo:** `apps/backend/migrations/000006_add_ssh_target.down.sql`

```sql
ALTER TABLE apps DROP COLUMN IF EXISTS ssh_key_id;
ALTER TABLE apps DROP COLUMN IF EXISTS ssh_user;
ALTER TABLE apps DROP COLUMN IF EXISTS ssh_port;
ALTER TABLE apps DROP COLUMN IF EXISTS ssh_host;
ALTER TABLE apps DROP COLUMN IF EXISTS deploy_mode;

DROP TABLE IF EXISTS ssh_keys;
```

## Mudancas no Backend (Go)

### 1. Domain Layer

**Novo arquivo:** `apps/backend/internal/domain/ssh_key.go`

```go
package domain

import "time"

type SSHKey struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    PrivateKey  string    `json:"-"`
    PublicKey   *string   `json:"publicKey,omitempty"`
    Fingerprint *string   `json:"fingerprint,omitempty"`
    CreatedAt   time.Time `json:"createdAt"`
}

type CreateSSHKeyInput struct {
    Name       string `json:"name"`
    PrivateKey string `json:"privateKey"`
}
```

**Modificar:** `apps/backend/internal/domain/app.go`

Adicionar campos:

```go
type App struct {
    // ... campos existentes ...
    DeployMode string  `json:"deployMode"`
    SSHHost    *string `json:"sshHost,omitempty"`
    SSHPort    *int    `json:"sshPort,omitempty"`
    SSHUser    *string `json:"sshUser,omitempty"`
    SSHKeyID   *string `json:"sshKeyId,omitempty"`
}

type DeployMode string

const (
    DeployModeLocal DeployMode = "local"
    DeployModeSSH   DeployMode = "ssh"
)
```

### 2. Repository Layer

**Novo arquivo:** `apps/backend/internal/repository/ssh_key_repository.go`

```go
package repository

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "your-project/internal/domain"
)

type SSHKeyRepository struct {
    db *pgxpool.Pool
}

func NewSSHKeyRepository(db *pgxpool.Pool) *SSHKeyRepository {
    return &SSHKeyRepository{db: db}
}

func (r *SSHKeyRepository) Create(ctx context.Context, input domain.CreateSSHKeyInput) (*domain.SSHKey, error) {
    // INSERT INTO ssh_keys (name, private_key, public_key, fingerprint)
    // VALUES ($1, $2, $3, $4)
    // RETURNING ...
}

func (r *SSHKeyRepository) GetByID(ctx context.Context, id string) (*domain.SSHKey, error) {
    // SELECT * FROM ssh_keys WHERE id = $1
}

func (r *SSHKeyRepository) List(ctx context.Context) ([]domain.SSHKey, error) {
    // SELECT id, name, public_key, fingerprint, created_at FROM ssh_keys
    // Nota: NAO retornar private_key na listagem
}

func (r *SSHKeyRepository) Delete(ctx context.Context, id string) error {
    // DELETE FROM ssh_keys WHERE id = $1
}
```

**Modificar:** `apps/backend/internal/repository/app_repository.go`

Atualizar queries para incluir novos campos:

```go
// Create - adicionar campos SSH
INSERT INTO apps (name, repository_url, branch, workdir, config, deploy_mode, ssh_host, ssh_port, ssh_user, ssh_key_id, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'active', NOW(), NOW())

// GetByID - incluir campos SSH no SELECT
SELECT id, name, repository_url, branch, workdir, runtime, config, status,
       webhook_id, deploy_mode, ssh_host, ssh_port, ssh_user, ssh_key_id,
       last_deployed_at, created_at, updated_at
FROM apps WHERE id = $1 AND status != 'deleted'
```

### 3. Engine Layer

**Novo arquivo:** `apps/backend/internal/engine/ssh_executor.go`

```go
package engine

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/pkg/sftp"
    "golang.org/x/crypto/ssh"
)

type SSHExecutor struct {
    client *ssh.Client
    host   string
    port   int
    user   string
}

type SSHConfig struct {
    Host       string
    Port       int
    User       string
    PrivateKey string
    Timeout    time.Duration
}

func NewSSHExecutor(cfg SSHConfig) (*SSHExecutor, error) {
    signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
    if err != nil {
        return nil, fmt.Errorf("failed to parse private key: %w", err)
    }

    config := &ssh.ClientConfig{
        User: cfg.User,
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: implementar verificacao
        Timeout:         cfg.Timeout,
    }

    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    client, err := ssh.Dial("tcp", addr, config)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
    }

    return &SSHExecutor{
        client: client,
        host:   cfg.Host,
        port:   cfg.Port,
        user:   cfg.User,
    }, nil
}

func (e *SSHExecutor) Run(ctx context.Context, cmd string) (string, error) {
    session, err := e.client.NewSession()
    if err != nil {
        return "", fmt.Errorf("failed to create session: %w", err)
    }
    defer session.Close()

    output, err := session.CombinedOutput(cmd)
    if err != nil {
        return string(output), fmt.Errorf("command failed: %w, output: %s", err, output)
    }

    return string(output), nil
}

func (e *SSHExecutor) RunWithOutput(ctx context.Context, cmd string, stdout, stderr io.Writer) error {
    session, err := e.client.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    defer session.Close()

    session.Stdout = stdout
    session.Stderr = stderr

    return session.Run(cmd)
}

func (e *SSHExecutor) Upload(localPath, remotePath string) error {
    sftpClient, err := sftp.NewClient(e.client)
    if err != nil {
        return fmt.Errorf("failed to create SFTP client: %w", err)
    }
    defer sftpClient.Close()

    // Implementar upload de arquivo
    // ...
}

func (e *SSHExecutor) Close() error {
    return e.client.Close()
}

func (e *SSHExecutor) TestConnection() error {
    _, err := e.Run(context.Background(), "echo 'connection test'")
    return err
}
```

**Modificar:** `apps/backend/internal/engine/worker.go`

```go
func (w *Worker) Run(ctx context.Context, deployment *domain.Deployment, app *domain.App) error {
    // Determinar modo de deploy
    if app.DeployMode == string(domain.DeployModeSSH) {
        return w.runRemoteDeploy(ctx, deployment, app)
    }
    return w.runLocalDeploy(ctx, deployment, app)
}

func (w *Worker) runRemoteDeploy(ctx context.Context, deployment *domain.Deployment, app *domain.App) error {
    // 1. Criar conexao SSH
    sshKey, err := w.sshKeyRepo.GetByID(ctx, *app.SSHKeyID)
    if err != nil {
        return fmt.Errorf("failed to get SSH key: %w", err)
    }

    executor, err := NewSSHExecutor(SSHConfig{
        Host:       *app.SSHHost,
        Port:       *app.SSHPort,
        User:       *app.SSHUser,
        PrivateKey: sshKey.PrivateKey,
        Timeout:    30 * time.Second,
    })
    if err != nil {
        return fmt.Errorf("failed to connect via SSH: %w", err)
    }
    defer executor.Close()

    // 2. Criar diretorio no servidor remoto
    dataDir := fmt.Sprintf("/opt/paasdeploy/apps/%s", app.ID)
    if _, err := executor.Run(ctx, fmt.Sprintf("mkdir -p %s", dataDir)); err != nil {
        return fmt.Errorf("failed to create data dir: %w", err)
    }

    // 3. Clonar/atualizar repositorio
    if err := w.syncGitRemote(ctx, executor, app, dataDir); err != nil {
        return err
    }

    // 4. Carregar configuracao
    config, err := w.loadConfigRemote(ctx, executor, app, dataDir)
    if err != nil {
        return err
    }

    // 5. Build Docker
    if err := w.buildDockerRemote(ctx, executor, app, config, dataDir); err != nil {
        return err
    }

    // 6. Deploy container
    if err := w.deployContainerRemote(ctx, executor, app, config, dataDir); err != nil {
        return err
    }

    // 7. Health check (via HTTP para o IP do servidor remoto)
    if err := w.healthCheckRemote(ctx, app, config); err != nil {
        return err
    }

    return nil
}

func (w *Worker) syncGitRemote(ctx context.Context, executor *SSHExecutor, app *domain.App, dataDir string) error {
    repoDir := fmt.Sprintf("%s/repo", dataDir)

    // Verificar se repo ja existe
    _, err := executor.Run(ctx, fmt.Sprintf("test -d %s/.git", repoDir))
    if err != nil {
        // Clone
        cmd := fmt.Sprintf("git clone --branch %s %s %s", app.Branch, app.RepositoryURL, repoDir)
        if _, err := executor.Run(ctx, cmd); err != nil {
            return fmt.Errorf("git clone failed: %w", err)
        }
    } else {
        // Pull
        cmd := fmt.Sprintf("cd %s && git fetch origin && git checkout %s && git pull origin %s", repoDir, app.Branch, app.Branch)
        if _, err := executor.Run(ctx, cmd); err != nil {
            return fmt.Errorf("git pull failed: %w", err)
        }
    }

    return nil
}
```

### 4. Handler Layer

**Novo arquivo:** `apps/backend/internal/handler/ssh_key_handler.go`

```go
package handler

import (
    "net/http"
    "your-project/internal/domain"
    "your-project/internal/repository"
    "your-project/internal/response"

    "github.com/go-chi/chi/v5"
)

type SSHKeyHandler struct {
    repo *repository.SSHKeyRepository
}

func NewSSHKeyHandler(repo *repository.SSHKeyRepository) *SSHKeyHandler {
    return &SSHKeyHandler{repo: repo}
}

// POST /api/ssh-keys
func (h *SSHKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
    var input domain.CreateSSHKeyInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // Validar e extrair fingerprint da chave
    // ...

    key, err := h.repo.Create(r.Context(), input)
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to create SSH key")
        return
    }

    response.JSON(w, http.StatusCreated, key)
}

// GET /api/ssh-keys
func (h *SSHKeyHandler) List(w http.ResponseWriter, r *http.Request) {
    keys, err := h.repo.List(r.Context())
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to list SSH keys")
        return
    }

    response.JSON(w, http.StatusOK, keys)
}

// DELETE /api/ssh-keys/:id
func (h *SSHKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    if err := h.repo.Delete(r.Context(), id); err != nil {
        response.Error(w, http.StatusInternalServerError, "failed to delete SSH key")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// POST /api/ssh-keys/:id/test
func (h *SSHKeyHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    var input struct {
        Host string `json:"host"`
        Port int    `json:"port"`
        User string `json:"user"`
    }
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid request body")
        return
    }

    key, err := h.repo.GetByID(r.Context(), id)
    if err != nil {
        response.Error(w, http.StatusNotFound, "SSH key not found")
        return
    }

    executor, err := engine.NewSSHExecutor(engine.SSHConfig{
        Host:       input.Host,
        Port:       input.Port,
        User:       input.User,
        PrivateKey: key.PrivateKey,
        Timeout:    10 * time.Second,
    })
    if err != nil {
        response.JSON(w, http.StatusOK, map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        })
        return
    }
    defer executor.Close()

    if err := executor.TestConnection(); err != nil {
        response.JSON(w, http.StatusOK, map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    response.JSON(w, http.StatusOK, map[string]interface{}{
        "success": true,
    })
}
```

### 5. Server Routes

**Modificar:** `apps/backend/internal/server/server.go`

```go
// Adicionar rotas SSH
r.Route("/ssh-keys", func(r chi.Router) {
    r.Get("/", sshKeyHandler.List)
    r.Post("/", sshKeyHandler.Create)
    r.Delete("/{id}", sshKeyHandler.Delete)
    r.Post("/{id}/test", sshKeyHandler.TestConnection)
})
```

## Mudancas no Frontend (React)

### 1. Tipos

**Modificar:** `apps/frontend/src/features/apps/types.ts`

```typescript
export type DeployMode = "local" | "ssh";

export interface CreateAppInput {
  name: string;
  repositoryUrl: string;
  branch?: string;
  workdir?: string;
  deployMode?: DeployMode;
  sshHost?: string;
  sshPort?: number;
  sshUser?: string;
  sshKeyId?: string;
}

export interface SSHKey {
  id: string;
  name: string;
  publicKey?: string;
  fingerprint?: string;
  createdAt: string;
}

export interface CreateSSHKeyInput {
  name: string;
  privateKey: string;
}

export interface TestSSHConnectionInput {
  host: string;
  port: number;
  user: string;
}

export interface TestSSHConnectionResult {
  success: boolean;
  error?: string;
}
```

### 2. API Service

**Modificar:** `apps/frontend/src/services/api.ts`

```typescript
export const api = {
  // ... existentes ...

  sshKeys: {
    list: (): Promise<readonly SSHKey[]> =>
      fetchApiList<SSHKey>(`${API_BASE}/ssh-keys`),

    create: (input: CreateSSHKeyInput): Promise<SSHKey> =>
      fetchApi<SSHKey>(`${API_BASE}/ssh-keys`, {
        method: "POST",
        body: JSON.stringify(input),
      }),

    delete: (id: string): Promise<void> =>
      fetchApi<void>(`${API_BASE}/ssh-keys/${id}`, {
        method: "DELETE",
      }),

    testConnection: (
      id: string,
      input: TestSSHConnectionInput,
    ): Promise<TestSSHConnectionResult> =>
      fetchApi<TestSSHConnectionResult>(`${API_BASE}/ssh-keys/${id}/test`, {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },
};
```

### 3. Hooks

**Novo arquivo:** `apps/frontend/src/features/ssh-keys/hooks/use-ssh-keys.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import type { CreateSSHKeyInput, TestSSHConnectionInput } from "../types";

export function useSSHKeys() {
  return useQuery({
    queryKey: ["ssh-keys"],
    queryFn: () => api.sshKeys.list(),
  });
}

export function useCreateSSHKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateSSHKeyInput) => api.sshKeys.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ssh-keys"] });
    },
  });
}

export function useDeleteSSHKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.sshKeys.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ssh-keys"] });
    },
  });
}

export function useTestSSHConnection() {
  return useMutation({
    mutationFn: ({
      keyId,
      input,
    }: {
      keyId: string;
      input: TestSSHConnectionInput;
    }) => api.sshKeys.testConnection(keyId, input),
  });
}
```

### 4. Componentes

**Modificar:** `apps/frontend/src/features/apps/components/app-form.tsx`

Adicionar secao de configuracao SSH apos os campos de repositorio:

```tsx
// Estado adicional
const [deployMode, setDeployMode] = useState<DeployMode>("local");
const [sshHost, setSSHHost] = useState("");
const [sshPort, setSSHPort] = useState(22);
const [sshUser, setSSHUser] = useState("root");
const [sshKeyId, setSSHKeyId] = useState("");

// Buscar chaves SSH
const { data: sshKeys } = useSSHKeys();
const testConnection = useTestSSHConnection();

// Componente de configuracao SSH
{
  deployMode === "ssh" && (
    <div className="space-y-4 border-l-2 border-blue-500 pl-4">
      <div>
        <Label htmlFor="sshKey">Chave SSH</Label>
        <Select value={sshKeyId} onValueChange={setSSHKeyId}>
          <SelectTrigger>
            <SelectValue placeholder="Selecione uma chave SSH" />
          </SelectTrigger>
          <SelectContent>
            {sshKeys?.map((key) => (
              <SelectItem key={key.id} value={key.id}>
                {key.name} ({key.fingerprint})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="sshHost">Host (IP ou Dominio)</Label>
          <Input
            id="sshHost"
            value={sshHost}
            onChange={(e) => setSSHHost(e.target.value)}
            placeholder="192.168.1.100"
          />
        </div>
        <div>
          <Label htmlFor="sshPort">Porta</Label>
          <Input
            id="sshPort"
            type="number"
            value={sshPort}
            onChange={(e) => setSSHPort(Number(e.target.value))}
          />
        </div>
      </div>

      <div>
        <Label htmlFor="sshUser">Usuario</Label>
        <Input
          id="sshUser"
          value={sshUser}
          onChange={(e) => setSSHUser(e.target.value)}
          placeholder="root"
        />
      </div>

      <Button
        type="button"
        variant="outline"
        onClick={handleTestConnection}
        disabled={!sshKeyId || !sshHost}
      >
        Testar Conexao
      </Button>
    </div>
  );
}
```

## Fluxo de Deploy Remoto

```
1. Usuario cria app com deployMode = "ssh"
2. Usuario configura: host, porta, usuario, chave SSH
3. Usuario testa conexao
4. Usuario inicia deploy
5. Worker:
   a. Conecta via SSH ao servidor remoto
   b. Cria diretorio /opt/paasdeploy/apps/{appId}
   c. Clona repositorio no servidor remoto
   d. Le paasdeploy.json do servidor remoto
   e. Executa docker build no servidor remoto
   f. Executa docker compose up no servidor remoto
   g. Faz health check via HTTP para o IP do servidor remoto
6. Deploy concluido
```

## Consideracoes de Seguranca

1. **Armazenamento de Chaves:** Chaves privadas devem ser criptografadas no banco (AES-256-GCM)
2. **Validacao de Host:** Validar formato de IP/dominio antes de conectar
3. **Timeout:** Timeout de conexao SSH de 30 segundos
4. **Logs:** NUNCA logar chaves privadas
5. **Permissoes:** Usuario SSH deve ter permissoes minimas necessarias
6. **Host Key:** Implementar verificacao de host key para producao

## Pre-requisitos no Servidor Remoto

O servidor de destino precisa ter instalado:

- Docker Engine 24+
- Docker Compose v2
- Git
- Usuario com acesso SSH e permissoes para Docker

Comandos para preparar o servidor:

```bash
# Instalar Docker
curl -fsSL https://get.docker.com | sh

# Adicionar usuario ao grupo docker
sudo usermod -aG docker $USER

# Instalar Git
sudo apt-get install -y git

# Criar diretorio para apps
sudo mkdir -p /opt/paasdeploy/apps
sudo chown -R $USER:$USER /opt/paasdeploy
```

## Proximos Passos

1. Implementar migration e modelos
2. Implementar SSHExecutor
3. Modificar Worker para suportar deploy remoto
4. Implementar handlers e rotas
5. Implementar interface no frontend
6. Testes de integracao
7. Documentacao de uso

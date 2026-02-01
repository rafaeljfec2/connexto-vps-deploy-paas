# Arquitetura de Deploy Remoto com Agent (gRPC)

## Visao Geral

O FlowDeploy suporta deploy de aplicacoes em servidores remotos atraves de um **Agent leve** que se comunica via **gRPC** com mTLS. O agent pode ser instalado:

1. **Automaticamente** - Durante o onboarding, FlowDeploy conecta via SSH e instala o agent
2. **Manualmente** - Usuario executa script de instalacao no servidor

## Por que gRPC ao inves de WebSocket/REST?

| Aspecto         | REST/WebSocket     | gRPC                       |
| --------------- | ------------------ | -------------------------- |
| Protocolo       | JSON over HTTP     | Protocol Buffers (binario) |
| Performance     | Mais lento         | 10x mais rapido            |
| Contrato        | Informal (docs)    | Fortemente tipado (.proto) |
| Streaming       | WebSocket separado | Nativo (bidirecional)      |
| Seguranca       | TLS + Token        | mTLS (certificados mutuos) |
| Versionamento   | Headers/URL        | Package versioning         |
| Code generation | Manual             | Automatico                 |

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      FlowDeploy Server (Cloud)                          │
│                                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │ Frontend │  │ Backend  │  │PostgreSQL│  │  gRPC    │               │
│  │  React   │  │   Go     │  │          │  │  Server  │               │
│  └──────────┘  └────┬─────┘  └──────────┘  └────┬─────┘               │
│                     │                           │                       │
│                     │                           │ gRPC + mTLS           │
│                     │                           │ (porta 50051)         │
└─────────────────────┼───────────────────────────┼───────────────────────┘
                      │                           │
                      │                           ▼
┌─────────────────────┼───────────────────────────────────────────────────┐
│                     │      Servidor Remoto 1                            │
│                     │                                                   │
│                     │   ┌────────────────┐                             │
│                     │   │  FlowDeploy    │◄── gRPC Client              │
│                     │   │    Agent       │    com certificado          │
│                     │   └───────┬────────┘                             │
│                     │           │                                       │
│                     │           ▼                                       │
│                     │   ┌──────────┐  ┌──────────┐                     │
│                     │   │ Docker   │  │  Apps    │                     │
│                     │   │ Engine   │  │          │                     │
│                     │   └──────────┘  └──────────┘                     │
└─────────────────────────────────────────────────────────────────────────┘
```

## Contrato gRPC (.proto)

### Estrutura de Arquivos Proto

```
apps/
├── proto/
│   └── flowdeploy/
│       └── v1/
│           ├── agent.proto      # Service principal
│           ├── deploy.proto     # Mensagens de deploy
│           ├── server.proto     # Mensagens de servidor
│           └── common.proto     # Tipos compartilhados
```

### agent.proto - Service Principal

```protobuf
syntax = "proto3";

package flowdeploy.v1;

option go_package = "github.com/flowdeploy/flowdeploy/gen/go/flowdeploy/v1;flowdeployv1";

import "flowdeploy/v1/deploy.proto";
import "flowdeploy/v1/server.proto";
import "flowdeploy/v1/common.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";

// AgentService - Servico principal do Agent
// O Agent conecta ao Server e mantem conexao persistente
service AgentService {
  // =========================================
  // CONEXAO E HEARTBEAT
  // =========================================

  // Register - Agent se registra no server ao iniciar
  // Envia informacoes do sistema e recebe configuracao
  rpc Register(RegisterRequest) returns (RegisterResponse);

  // Heartbeat - Agent envia heartbeat periodico
  // Server responde com comandos pendentes (se houver)
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

  // =========================================
  // DEPLOY
  // =========================================

  // ExecuteDeploy - Server solicita deploy ao agent
  // Agent executa e retorna resultado
  rpc ExecuteDeploy(DeployRequest) returns (DeployResponse);

  // StreamDeployLogs - Stream bidirecional de logs durante deploy
  // Agent envia logs em tempo real, server pode cancelar
  rpc StreamDeployLogs(stream DeployLogEntry) returns (stream DeployLogControl);

  // =========================================
  // CONTAINER MANAGEMENT
  // =========================================

  // ListContainers - Lista containers no servidor
  rpc ListContainers(ListContainersRequest) returns (ListContainersResponse);

  // GetContainerLogs - Stream de logs de um container
  rpc GetContainerLogs(ContainerLogsRequest) returns (stream ContainerLogEntry);

  // GetContainerStats - Stream de metricas de um container
  rpc GetContainerStats(ContainerStatsRequest) returns (stream ContainerStats);

  // RestartContainer - Reinicia um container
  rpc RestartContainer(RestartContainerRequest) returns (RestartContainerResponse);

  // StopContainer - Para um container
  rpc StopContainer(StopContainerRequest) returns (StopContainerResponse);

  // =========================================
  // SYSTEM INFO
  // =========================================

  // GetSystemInfo - Informacoes do sistema (CPU, memoria, disco)
  rpc GetSystemInfo(google.protobuf.Empty) returns (SystemInfo);

  // GetDockerInfo - Informacoes do Docker
  rpc GetDockerInfo(google.protobuf.Empty) returns (DockerInfo);
}
```

### deploy.proto - Mensagens de Deploy

```protobuf
syntax = "proto3";

package flowdeploy.v1;

option go_package = "github.com/flowdeploy/flowdeploy/gen/go/flowdeploy/v1;flowdeployv1";

import "google/protobuf/timestamp.proto";
import "flowdeploy/v1/common.proto";

// =========================================
// DEPLOY REQUEST/RESPONSE
// =========================================

message DeployRequest {
  string deployment_id = 1;
  string app_id = 2;
  string app_name = 3;

  // Git configuration
  GitConfig git = 4;

  // Build configuration
  BuildConfig build = 5;

  // Runtime configuration
  RuntimeConfig runtime = 6;

  // Environment variables (encrypted in transit via mTLS)
  map<string, string> env_vars = 7;

  // Health check configuration
  HealthCheckConfig health_check = 8;

  // Rollback image (for rollback on failure)
  optional string rollback_image = 9;
}

message GitConfig {
  string repository_url = 1;
  string branch = 2;
  string commit_sha = 3;      // Specific commit (optional)
  string workdir = 4;         // Subdirectory for monorepos

  // Authentication
  oneof auth {
    string access_token = 5;  // GitHub PAT or similar
    SSHAuth ssh_auth = 6;     // SSH key authentication
  }
}

message SSHAuth {
  string private_key = 1;
  string passphrase = 2;      // Optional
}

message BuildConfig {
  string dockerfile = 1;      // Path to Dockerfile (default: ./Dockerfile)
  string context = 2;         // Build context (default: .)
  map<string, string> args = 3;  // Build arguments
  string target = 4;          // Multi-stage build target (optional)
  repeated string cache_from = 5; // Images to use as cache
}

message RuntimeConfig {
  int32 port = 1;             // Container port
  optional int32 host_port = 2; // Host port (optional, for direct mapping)

  ResourceLimits resources = 3;

  repeated string domains = 4; // Custom domains

  int32 replicas = 5;         // Number of replicas (default: 1)

  // Network configuration
  string network = 6;         // Docker network name

  // Restart policy
  RestartPolicy restart_policy = 7;
}

message ResourceLimits {
  string memory = 1;          // e.g., "512m", "1g"
  string cpu = 2;             // e.g., "0.5", "2"
  string memory_swap = 3;     // Optional swap limit
}

enum RestartPolicy {
  RESTART_POLICY_UNSPECIFIED = 0;
  RESTART_POLICY_NO = 1;
  RESTART_POLICY_ALWAYS = 2;
  RESTART_POLICY_ON_FAILURE = 3;
  RESTART_POLICY_UNLESS_STOPPED = 4;
}

message HealthCheckConfig {
  string path = 1;            // HTTP path (e.g., /health)
  string interval = 2;        // e.g., "30s"
  string timeout = 3;         // e.g., "5s"
  int32 retries = 4;          // Number of retries
  string start_period = 5;    // e.g., "10s"
}

message DeployResponse {
  bool success = 1;
  string message = 2;

  // Deploy result details
  optional DeployResult result = 3;

  // Error details (if failed)
  optional DeployError error = 4;
}

message DeployResult {
  string container_id = 1;
  string image_id = 2;
  string image_tag = 3;

  google.protobuf.Timestamp started_at = 4;
  google.protobuf.Timestamp completed_at = 5;

  int64 build_duration_ms = 6;
  int64 deploy_duration_ms = 7;

  // Container info
  int32 exposed_port = 8;
  string container_ip = 9;
}

message DeployError {
  DeployErrorCode code = 1;
  string message = 2;
  string stage = 3;           // git_sync, build, deploy, health_check
  string details = 4;         // Stack trace or detailed error
}

enum DeployErrorCode {
  DEPLOY_ERROR_UNSPECIFIED = 0;
  DEPLOY_ERROR_GIT_CLONE_FAILED = 1;
  DEPLOY_ERROR_GIT_CHECKOUT_FAILED = 2;
  DEPLOY_ERROR_DOCKERFILE_NOT_FOUND = 3;
  DEPLOY_ERROR_BUILD_FAILED = 4;
  DEPLOY_ERROR_CONTAINER_START_FAILED = 5;
  DEPLOY_ERROR_HEALTH_CHECK_FAILED = 6;
  DEPLOY_ERROR_ROLLBACK_FAILED = 7;
  DEPLOY_ERROR_TIMEOUT = 8;
  DEPLOY_ERROR_CANCELLED = 9;
  DEPLOY_ERROR_INTERNAL = 10;
}

// =========================================
// DEPLOY LOGS (STREAMING)
// =========================================

message DeployLogEntry {
  string deployment_id = 1;

  google.protobuf.Timestamp timestamp = 2;

  DeployLogLevel level = 3;
  DeployStage stage = 4;

  string message = 5;

  // Progress info (optional)
  optional DeployProgress progress = 6;
}

enum DeployLogLevel {
  DEPLOY_LOG_LEVEL_UNSPECIFIED = 0;
  DEPLOY_LOG_LEVEL_DEBUG = 1;
  DEPLOY_LOG_LEVEL_INFO = 2;
  DEPLOY_LOG_LEVEL_WARN = 3;
  DEPLOY_LOG_LEVEL_ERROR = 4;
}

enum DeployStage {
  DEPLOY_STAGE_UNSPECIFIED = 0;
  DEPLOY_STAGE_INITIALIZING = 1;
  DEPLOY_STAGE_GIT_SYNC = 2;
  DEPLOY_STAGE_BUILD = 3;
  DEPLOY_STAGE_PUSH = 4;
  DEPLOY_STAGE_DEPLOY = 5;
  DEPLOY_STAGE_HEALTH_CHECK = 6;
  DEPLOY_STAGE_CLEANUP = 7;
  DEPLOY_STAGE_ROLLBACK = 8;
  DEPLOY_STAGE_COMPLETE = 9;
}

message DeployProgress {
  int32 current_step = 1;
  int32 total_steps = 2;
  string step_name = 3;
  int32 percentage = 4;       // 0-100
}

message DeployLogControl {
  DeployLogControlAction action = 1;
  string reason = 2;
}

enum DeployLogControlAction {
  DEPLOY_LOG_CONTROL_UNSPECIFIED = 0;
  DEPLOY_LOG_CONTROL_CONTINUE = 1;    // Continue sending logs
  DEPLOY_LOG_CONTROL_CANCEL = 2;      // Cancel the deploy
  DEPLOY_LOG_CONTROL_ACK = 3;         // Acknowledge receipt
}
```

### server.proto - Mensagens de Servidor

```protobuf
syntax = "proto3";

package flowdeploy.v1;

option go_package = "github.com/flowdeploy/flowdeploy/gen/go/flowdeploy/v1;flowdeployv1";

import "google/protobuf/timestamp.proto";

// =========================================
// REGISTER
// =========================================

message RegisterRequest {
  string agent_id = 1;        // Unique agent identifier (from cert CN)
  string agent_version = 2;

  SystemInfo system_info = 3;
  DockerInfo docker_info = 4;
}

message RegisterResponse {
  bool accepted = 1;
  string message = 2;

  // Server configuration for agent
  AgentConfig config = 3;

  // Pending deploys (if any)
  repeated string pending_deployment_ids = 4;
}

message AgentConfig {
  int32 heartbeat_interval_seconds = 1;  // Default: 30
  int32 log_buffer_size = 2;             // Default: 1000
  int32 max_concurrent_deploys = 3;      // Default: 2

  // Feature flags
  bool enable_metrics = 4;
  bool enable_auto_update = 5;
}

// =========================================
// HEARTBEAT
// =========================================

message HeartbeatRequest {
  string agent_id = 1;
  google.protobuf.Timestamp timestamp = 2;

  // Current status
  AgentStatus status = 3;

  // Active deployments
  repeated ActiveDeployment active_deployments = 4;

  // System metrics (optional, if enabled)
  optional SystemMetrics metrics = 5;
}

message AgentStatus {
  AgentState state = 1;
  int32 active_deploy_count = 2;
  int32 container_count = 3;

  google.protobuf.Timestamp started_at = 4;
  int64 uptime_seconds = 5;
}

enum AgentState {
  AGENT_STATE_UNSPECIFIED = 0;
  AGENT_STATE_IDLE = 1;
  AGENT_STATE_DEPLOYING = 2;
  AGENT_STATE_ERROR = 3;
  AGENT_STATE_UPDATING = 4;
}

message ActiveDeployment {
  string deployment_id = 1;
  string app_id = 2;
  DeployStage stage = 3;
  google.protobuf.Timestamp started_at = 4;
}

message HeartbeatResponse {
  bool acknowledged = 1;

  // Commands from server (optional)
  repeated AgentCommand commands = 2;

  // Updated config (optional)
  optional AgentConfig updated_config = 3;
}

message AgentCommand {
  AgentCommandType type = 1;
  string payload = 2;         // JSON payload
}

enum AgentCommandType {
  AGENT_COMMAND_UNSPECIFIED = 0;
  AGENT_COMMAND_DEPLOY = 1;           // Start a deploy
  AGENT_COMMAND_CANCEL_DEPLOY = 2;    // Cancel active deploy
  AGENT_COMMAND_RESTART_CONTAINER = 3;
  AGENT_COMMAND_UPDATE_AGENT = 4;     // Update agent binary
  AGENT_COMMAND_SHUTDOWN = 5;         // Graceful shutdown
}

// =========================================
// SYSTEM INFO
// =========================================

message SystemInfo {
  string hostname = 1;
  string os = 2;              // e.g., "linux"
  string os_version = 3;      // e.g., "Ubuntu 22.04"
  string architecture = 4;    // e.g., "amd64"

  int32 cpu_cores = 5;
  int64 memory_total_bytes = 6;
  int64 disk_total_bytes = 7;

  string kernel_version = 8;
}

message DockerInfo {
  string version = 1;
  string api_version = 2;

  string storage_driver = 3;
  int64 images_count = 4;
  int64 containers_count = 5;

  bool swarm_active = 6;
}

message SystemMetrics {
  double cpu_usage_percent = 1;
  int64 memory_used_bytes = 2;
  int64 memory_available_bytes = 3;
  int64 disk_used_bytes = 4;
  int64 disk_available_bytes = 5;

  double load_average_1m = 6;
  double load_average_5m = 7;
  double load_average_15m = 8;

  int64 network_rx_bytes = 9;
  int64 network_tx_bytes = 10;
}

// =========================================
// CONTAINER MANAGEMENT
// =========================================

message ListContainersRequest {
  bool all = 1;               // Include stopped containers
  optional string app_id = 2; // Filter by app
}

message ListContainersResponse {
  repeated ContainerInfo containers = 1;
}

message ContainerInfo {
  string id = 1;
  string name = 2;
  string image = 3;
  string state = 4;           // running, stopped, etc.
  string status = 5;          // human readable status

  google.protobuf.Timestamp created_at = 6;

  map<string, string> labels = 7;

  repeated PortBinding ports = 8;
}

message PortBinding {
  int32 container_port = 1;
  int32 host_port = 2;
  string protocol = 3;        // tcp, udp
}

message ContainerLogsRequest {
  string container_id = 1;
  bool follow = 2;            // Stream logs
  int32 tail = 3;             // Number of lines from end (0 = all)
  bool timestamps = 4;
  optional google.protobuf.Timestamp since = 5;
}

message ContainerLogEntry {
  google.protobuf.Timestamp timestamp = 1;
  string stream = 2;          // stdout, stderr
  string message = 3;
}

message ContainerStatsRequest {
  string container_id = 1;
  bool stream = 2;            // Stream stats
}

message ContainerStats {
  google.protobuf.Timestamp timestamp = 1;

  double cpu_percent = 2;
  int64 memory_usage_bytes = 3;
  int64 memory_limit_bytes = 4;

  int64 network_rx_bytes = 5;
  int64 network_tx_bytes = 6;

  int64 block_read_bytes = 7;
  int64 block_write_bytes = 8;
}

message RestartContainerRequest {
  string container_id = 1;
  int32 timeout_seconds = 2;  // Default: 10
}

message RestartContainerResponse {
  bool success = 1;
  string message = 2;
}

message StopContainerRequest {
  string container_id = 1;
  int32 timeout_seconds = 2;  // Default: 10
}

message StopContainerResponse {
  bool success = 1;
  string message = 2;
}
```

### common.proto - Tipos Compartilhados

```protobuf
syntax = "proto3";

package flowdeploy.v1;

option go_package = "github.com/flowdeploy/flowdeploy/gen/go/flowdeploy/v1;flowdeployv1";

// Reexport commonly used types
// This file can be extended with shared enums, messages, etc.

message Pagination {
  int32 page = 1;
  int32 page_size = 2;
}

message PaginationInfo {
  int32 total = 1;
  int32 page = 2;
  int32 page_size = 3;
  int32 total_pages = 4;
}
```

## Arquitetura de Seguranca (mTLS)

### Visao Geral do mTLS

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           CERTIFICATE AUTHORITY                         │
│                                                                         │
│  FlowDeploy Server gera e gerencia certificados                         │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                         ROOT CA                                  │   │
│  │                                                                  │   │
│  │  - Gerado na instalacao do FlowDeploy                           │   │
│  │  - Armazenado de forma segura (encrypted)                       │   │
│  │  - Usado para assinar todos os certificados                     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│              ┌───────────────┴───────────────┐                         │
│              │                               │                         │
│              ▼                               ▼                         │
│  ┌─────────────────────┐       ┌─────────────────────┐                │
│  │   SERVER CERT       │       │   AGENT CERT        │                │
│  │                     │       │   (per server)      │                │
│  │   CN: flowdeploy    │       │                     │                │
│  │   server            │       │   CN: server-uuid   │                │
│  │                     │       │   OU: agent         │                │
│  │   Usado pelo gRPC   │       │                     │                │
│  │   server            │       │   Gerado durante    │                │
│  └─────────────────────┘       │   provisionamento   │                │
│                                └─────────────────────┘                │
└─────────────────────────────────────────────────────────────────────────┘
```

### Fluxo de Certificados

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │     │   Backend    │     │    Agent     │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       │ 1. Add Server      │                    │
       │ (with SSH creds)   │                    │
       │───────────────────>│                    │
       │                    │                    │
       │                    │ 2. Generate:       │
       │                    │    - Agent Key     │
       │                    │    - Agent CSR     │
       │                    │    - Sign with CA  │
       │                    │────┐               │
       │                    │<───┘               │
       │                    │                    │
       │                    │ 3. SSH: Install    │
       │                    │    - Agent binary  │
       │                    │    - CA cert       │
       │                    │    - Agent cert    │
       │                    │    - Agent key     │
       │                    │───────────────────>│
       │                    │                    │
       │                    │ 4. Close SSH       │
       │                    │       X            │
       │                    │                    │
       │                    │ 5. Agent connects  │
       │                    │    with mTLS       │
       │                    │<───────────────────│
       │                    │                    │
       │                    │ 6. Verify:         │
       │                    │    - Agent cert    │
       │                    │      signed by CA  │
       │                    │    - CN matches    │
       │                    │      server UUID   │
       │                    │────┐               │
       │                    │<───┘               │
       │                    │                    │
       │ 7. Server Online   │                    │
       │<───────────────────│                    │
       │                    │                    │
```

### Implementacao da PKI

**Arquivo:** `apps/backend/internal/pki/ca.go`

```go
package pki

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "math/big"
    "time"
)

type CertificateAuthority struct {
    cert       *x509.Certificate
    privateKey *ecdsa.PrivateKey
}

// NewCA creates a new Certificate Authority
func NewCA() (*CertificateAuthority, error) {
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, err
    }

    template := &x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject: pkix.Name{
            Organization: []string{"FlowDeploy"},
            CommonName:   "FlowDeploy Root CA",
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
        IsCA:                  true,
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        BasicConstraintsValid: true,
    }

    certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
    if err != nil {
        return nil, err
    }

    cert, err := x509.ParseCertificate(certDER)
    if err != nil {
        return nil, err
    }

    return &CertificateAuthority{
        cert:       cert,
        privateKey: privateKey,
    }, nil
}

// GenerateServerCert generates the gRPC server certificate
func (ca *CertificateAuthority) GenerateServerCert(hostname string) (*Certificate, error) {
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, err
    }

    template := &x509.Certificate{
        SerialNumber: generateSerial(),
        Subject: pkix.Name{
            Organization: []string{"FlowDeploy"},
            CommonName:   "flowdeploy-server",
        },
        DNSNames:    []string{hostname, "localhost"},
        NotBefore:   time.Now(),
        NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year
        KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    }

    certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &privateKey.PublicKey, ca.privateKey)
    if err != nil {
        return nil, err
    }

    return &Certificate{
        CertPEM: pemEncode("CERTIFICATE", certDER),
        KeyPEM:  pemEncodeKey(privateKey),
    }, nil
}

// GenerateAgentCert generates a certificate for an agent
func (ca *CertificateAuthority) GenerateAgentCert(serverID string) (*Certificate, error) {
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, err
    }

    template := &x509.Certificate{
        SerialNumber: generateSerial(),
        Subject: pkix.Name{
            Organization:       []string{"FlowDeploy"},
            OrganizationalUnit: []string{"agent"},
            CommonName:         serverID, // Server UUID as CN
        },
        NotBefore:   time.Now(),
        NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year
        KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
    }

    certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &privateKey.PublicKey, ca.privateKey)
    if err != nil {
        return nil, err
    }

    return &Certificate{
        CertPEM: pemEncode("CERTIFICATE", certDER),
        KeyPEM:  pemEncodeKey(privateKey),
    }, nil
}

// GetCACertPEM returns the CA certificate in PEM format
func (ca *CertificateAuthority) GetCACertPEM() []byte {
    return pemEncode("CERTIFICATE", ca.cert.Raw)
}

type Certificate struct {
    CertPEM []byte
    KeyPEM  []byte
}

func generateSerial() *big.Int {
    serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
    return serial
}

func pemEncode(blockType string, data []byte) []byte {
    return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: data})
}

func pemEncodeKey(key *ecdsa.PrivateKey) []byte {
    data, _ := x509.MarshalECPrivateKey(key)
    return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: data})
}
```

### Configuracao do gRPC Server com mTLS

**Arquivo:** `apps/backend/internal/grpc/server.go`

```go
package grpc

import (
    "crypto/tls"
    "crypto/x509"
    "net"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/peer"

    pb "flowdeploy/gen/go/flowdeploy/v1"
    "flowdeploy/internal/pki"
)

type Server struct {
    pb.UnimplementedAgentServiceServer

    grpcServer *grpc.Server
    ca         *pki.CertificateAuthority
    serverRepo ServerRepository
    hub        *AgentHub
}

func NewServer(ca *pki.CertificateAuthority, serverCert *pki.Certificate) (*Server, error) {
    // Load server certificate
    cert, err := tls.X509KeyPair(serverCert.CertPEM, serverCert.KeyPEM)
    if err != nil {
        return nil, err
    }

    // Create certificate pool with CA
    certPool := x509.NewCertPool()
    certPool.AppendCertsFromPEM(ca.GetCACertPEM())

    // Configure mTLS
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert,
        ClientCAs:    certPool,
        MinVersion:   tls.VersionTLS13,
    }

    // Create gRPC server with mTLS
    grpcServer := grpc.NewServer(
        grpc.Creds(credentials.NewTLS(tlsConfig)),
        grpc.UnaryInterceptor(authInterceptor),
        grpc.StreamInterceptor(streamAuthInterceptor),
    )

    s := &Server{
        grpcServer: grpcServer,
        ca:         ca,
        hub:        NewAgentHub(),
    }

    pb.RegisterAgentServiceServer(grpcServer, s)

    return s, nil
}

func (s *Server) Start(address string) error {
    lis, err := net.Listen("tcp", address)
    if err != nil {
        return err
    }

    return s.grpcServer.Serve(lis)
}

// authInterceptor extracts server ID from client certificate
func authInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    serverID, err := extractServerIDFromCert(ctx)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid certificate")
    }

    ctx = context.WithValue(ctx, "server_id", serverID)
    return handler(ctx, req)
}

func extractServerIDFromCert(ctx context.Context) (string, error) {
    p, ok := peer.FromContext(ctx)
    if !ok {
        return "", fmt.Errorf("no peer info")
    }

    tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
    if !ok {
        return "", fmt.Errorf("no TLS info")
    }

    if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
        return "", fmt.Errorf("no verified chains")
    }

    cert := tlsInfo.State.VerifiedChains[0][0]

    // Verify OU is "agent"
    if len(cert.Subject.OrganizationalUnit) == 0 || cert.Subject.OrganizationalUnit[0] != "agent" {
        return "", fmt.Errorf("invalid certificate OU")
    }

    return cert.Subject.CommonName, nil
}
```

### Configuracao do Agent com mTLS

**Arquivo:** `apps/agent/internal/grpc/client.go`

```go
package grpc

import (
    "crypto/tls"
    "crypto/x509"
    "os"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"

    pb "flowdeploy/gen/go/flowdeploy/v1"
)

type Client struct {
    conn   *grpc.ClientConn
    client pb.AgentServiceClient
}

func NewClient(serverAddr, caCertPath, certPath, keyPath string) (*Client, error) {
    // Load CA certificate
    caCert, err := os.ReadFile(caCertPath)
    if err != nil {
        return nil, err
    }

    certPool := x509.NewCertPool()
    certPool.AppendCertsFromPEM(caCert)

    // Load agent certificate
    cert, err := tls.LoadX509KeyPair(certPath, keyPath)
    if err != nil {
        return nil, err
    }

    // Configure mTLS
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      certPool,
        MinVersion:   tls.VersionTLS13,
    }

    // Connect with mTLS
    conn, err := grpc.Dial(
        serverAddr,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
    )
    if err != nil {
        return nil, err
    }

    return &Client{
        conn:   conn,
        client: pb.NewAgentServiceClient(conn),
    }, nil
}

func (c *Client) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    return c.client.Register(ctx, req)
}

func (c *Client) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
    return c.client.Heartbeat(ctx, req)
}

func (c *Client) StreamDeployLogs(ctx context.Context) (pb.AgentService_StreamDeployLogsClient, error) {
    return c.client.StreamDeployLogs(ctx)
}

func (c *Client) Close() error {
    return c.conn.Close()
}
```

## Estrategia de Versionamento gRPC

### Estrutura de Packages

```
proto/
└── flowdeploy/
    ├── v1/           # Versao estavel atual
    │   ├── agent.proto
    │   ├── deploy.proto
    │   └── server.proto
    └── v2/           # Proxima versao (quando necessario)
        ├── agent.proto
        └── ...
```

### Regras de Versionamento

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    REGRAS DE COMPATIBILIDADE                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  PERMITIDO (backward compatible):                                       │
│  ✓ Adicionar novos campos (com numero novo)                            │
│  ✓ Adicionar novos RPCs                                                │
│  ✓ Adicionar novos valores em enums                                    │
│  ✓ Deprecar campos (nao remover)                                       │
│                                                                         │
│  PROIBIDO (breaking changes):                                           │
│  ✗ Remover campos existentes                                           │
│  ✗ Mudar tipo de campos                                                │
│  ✗ Mudar numero de campos                                              │
│  ✗ Renomear campos                                                     │
│  ✗ Remover valores de enums                                            │
│                                                                         │
│  REQUER NOVA VERSAO (v2):                                              │
│  • Mudancas estruturais significativas                                  │
│  • Remocao de funcionalidades                                          │
│  • Mudanca de semantica de campos                                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Negociacao de Versao

```protobuf
// No RegisterRequest, agent informa versoes suportadas
message RegisterRequest {
  string agent_id = 1;
  string agent_version = 2;

  // Versoes do protocolo suportadas
  repeated string supported_protocol_versions = 10; // ["v1", "v1.1"]
  string preferred_protocol_version = 11;           // "v1.1"

  SystemInfo system_info = 3;
  DockerInfo docker_info = 4;
}

// No RegisterResponse, server confirma versao a usar
message RegisterResponse {
  bool accepted = 1;
  string message = 2;

  // Versao do protocolo selecionada
  string selected_protocol_version = 10;

  // Se agent precisa atualizar
  optional AgentUpdateInfo update_available = 11;

  AgentConfig config = 3;
  repeated string pending_deployment_ids = 4;
}

message AgentUpdateInfo {
  string latest_version = 1;
  string download_url = 2;
  string changelog = 3;
  bool required = 4;  // Se update e obrigatorio
}
```

### Suporte Multi-Versao no Server

```go
package grpc

import (
    "context"

    pb "flowdeploy/gen/go/flowdeploy/v1"
)

type VersionedAgentService struct {
    pb.UnimplementedAgentServiceServer

    // Handlers para diferentes versoes
    v1Handler *V1Handler
    v1_1Handler *V1_1Handler
}

func (s *VersionedAgentService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // Verificar versoes suportadas pelo agent
    selectedVersion := selectBestVersion(req.SupportedProtocolVersions)

    // Armazenar versao no contexto do agent para uso futuro
    ctx = context.WithValue(ctx, "protocol_version", selectedVersion)

    // Verificar se agent precisa atualizar
    var updateInfo *pb.AgentUpdateInfo
    if needsUpdate(req.AgentVersion) {
        updateInfo = &pb.AgentUpdateInfo{
            LatestVersion: "1.2.0",
            DownloadUrl:   "https://releases.flowdeploy.io/agent/1.2.0/linux-amd64",
            Required:      isUpdateRequired(req.AgentVersion),
        }
    }

    return &pb.RegisterResponse{
        Accepted:                true,
        SelectedProtocolVersion: selectedVersion,
        UpdateAvailable:         updateInfo,
        Config:                  getDefaultConfig(),
    }, nil
}

func selectBestVersion(supported []string) string {
    // Ordem de preferencia: mais recente primeiro
    preferenceOrder := []string{"v1.1", "v1"}

    for _, preferred := range preferenceOrder {
        for _, supported := range supported {
            if preferred == supported {
                return preferred
            }
        }
    }

    return "v1" // fallback
}
```

## Fluxo de Provisionamento com gRPC

### Diagrama de Sequencia

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Frontend │     │ Backend  │     │  Server  │     │  Agent   │
│          │     │          │     │ (Remote) │     │          │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ 1. Add Server  │                │                │
     │ (SSH creds)    │                │                │
     │───────────────>│                │                │
     │                │                │                │
     │                │ 2. Generate:   │                │
     │                │    - Agent key │                │
     │                │    - Agent cert│                │
     │                │    (signed CA) │                │
     │                │────┐           │                │
     │                │<───┘           │                │
     │                │                │                │
     │                │ 3. SSH Connect │                │
     │                │───────────────>│                │
     │                │                │                │
     │ 4. Progress    │                │                │
     │    (SSE)       │                │                │
     │<───────────────│                │                │
     │                │                │                │
     │                │ 5. Install:    │                │
     │                │    - Binary    │                │
     │                │    - CA cert   │                │
     │                │    - Agent cert│                │
     │                │    - Agent key │                │
     │                │───────────────>│                │
     │                │                │                │
     │                │ 6. Start agent │                │
     │                │───────────────>│────┐           │
     │                │                │    │ systemctl │
     │                │                │<───┘ start     │
     │                │                │                │
     │                │ 7. Close SSH   │                │
     │                │       X        │                │
     │                │                │                │
     │                │ 8. Agent       │                │
     │                │    connects    │                │
     │                │    with mTLS   │                │
     │                │<───────────────────────────────│
     │                │                │                │
     │                │ 9. Verify cert │                │
     │                │    CN = UUID   │                │
     │                │────┐           │                │
     │                │<───┘           │                │
     │                │                │                │
     │                │ 10. Register() │                │
     │                │<───────────────────────────────│
     │                │                │                │
     │ 11. Server     │                │                │
     │     Online     │                │                │
     │<───────────────│                │                │
     │                │                │                │
```

## Estrutura de Arquivos Final

```
apps/
├── proto/
│   └── flowdeploy/
│       └── v1/
│           ├── agent.proto
│           ├── deploy.proto
│           ├── server.proto
│           └── common.proto
├── agent/
│   ├── cmd/
│   │   └── agent/
│   │       └── main.go
│   ├── internal/
│   │   ├── agent/
│   │   │   └── agent.go
│   │   ├── deploy/
│   │   │   └── executor.go
│   │   └── grpc/
│   │       └── client.go
│   ├── install.sh
│   ├── Dockerfile
│   └── go.mod
├── backend/
│   ├── gen/
│   │   └── go/
│   │       └── flowdeploy/
│   │           └── v1/
│   │               ├── agent.pb.go
│   │               ├── agent_grpc.pb.go
│   │               ├── deploy.pb.go
│   │               └── server.pb.go
│   ├── internal/
│   │   ├── grpc/
│   │   │   ├── server.go
│   │   │   └── handlers.go
│   │   ├── pki/
│   │   │   └── ca.go
│   │   ├── domain/
│   │   │   └── server.go
│   │   ├── handler/
│   │   │   └── server_handler.go
│   │   ├── provisioner/
│   │   │   └── ssh_provisioner.go
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
                │   ├── add-server-dialog.tsx
                │   └── server-selector.tsx
                ├── hooks/
                │   └── use-servers.ts
                └── types.ts
```

## Makefile para Geracao de Codigo

```makefile
.PHONY: proto proto-go proto-lint

PROTO_DIR := apps/proto
GEN_GO_DIR := apps/backend/gen/go

proto: proto-lint proto-go

proto-lint:
	buf lint $(PROTO_DIR)

proto-go:
	buf generate $(PROTO_DIR) --template $(PROTO_DIR)/buf.gen.yaml

proto-breaking:
	buf breaking $(PROTO_DIR) --against '.git#branch=main'
```

### buf.gen.yaml

```yaml
version: v1
plugins:
  - plugin: go
    out: ../backend/gen/go
    opt:
      - paths=source_relative
  - plugin: go-grpc
    out: ../backend/gen/go
    opt:
      - paths=source_relative
```

### buf.yaml

```yaml
version: v1
name: buf.build/flowdeploy/flowdeploy
deps:
  - buf.build/googleapis/googleapis
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

## Maquina de Estados do Agent

### Diagrama de Estados Principal

```
                                    ┌─────────────────────────────────────────────────────────────┐
                                    │                     AGENT STATE MACHINE                     │
                                    └─────────────────────────────────────────────────────────────┘

     ┌──────────┐     load config      ┌──────────┐     connect ok       ┌──────────┐
     │          │────────────────────>│          │─────────────────────>│          │
     │  INIT    │                      │CONNECTING│                      │REGISTERING│
     │          │<────────────────────│          │<─────────────────────│          │
     └──────────┘     config error     └──────────┘     connect fail     └──────────┘
          │                                 │                                 │
          │                                 │ max retries                     │ register ok
          ▼                                 ▼                                 ▼
     ┌──────────┐                     ┌──────────┐                      ┌──────────┐
     │          │                     │          │                      │          │
     │  FATAL   │                     │  BACKOFF │────────────────────>│   IDLE   │◄────────┐
     │          │                     │          │      retry           │          │         │
     └──────────┘                     └──────────┘                      └──────────┘         │
                                           ▲                                 │               │
                                           │                                 │               │
                                           │ connection lost                 │ deploy cmd   │
                                           │                                 ▼               │
                                           │                           ┌──────────┐         │
                                           │                           │          │         │
                                           └───────────────────────────│ DEPLOYING│─────────┘
                                                                       │          │  deploy done
                                                                       └──────────┘
                                                                            │
                                                                            │ deploy error
                                                                            ▼
                                                                      ┌──────────┐
                                                                      │          │
                                                                      │ ROLLBACK │──────────┐
                                                                      │          │          │
                                                                      └──────────┘          │
                                                                                            │
                                                                                            ▼
                                                                                       back to IDLE
```

### Estados do Agent

```go
type AgentState string

const (
    // Estados de inicializacao
    StateInit         AgentState = "INIT"          // Carregando configuracao
    StateConnecting   AgentState = "CONNECTING"    // Conectando ao server
    StateRegistering  AgentState = "REGISTERING"   // Registrando no server

    // Estados operacionais
    StateIdle         AgentState = "IDLE"          // Aguardando comandos
    StateDeploying    AgentState = "DEPLOYING"     // Executando deploy
    StateRollback     AgentState = "ROLLBACK"      // Executando rollback

    // Estados de recuperacao
    StateBackoff      AgentState = "BACKOFF"       // Aguardando para reconectar
    StateReconnecting AgentState = "RECONNECTING"  // Reconectando apos perda

    // Estados de manutencao
    StateUpdating     AgentState = "UPDATING"      // Atualizando agent
    StateDraining     AgentState = "DRAINING"      // Finalizando deploys antes de shutdown

    // Estados terminais
    StateFatal        AgentState = "FATAL"         // Erro fatal, requer intervencao
    StateShutdown     AgentState = "SHUTDOWN"      // Agent encerrado
)
```

### Tabela de Transicoes

```
┌─────────────────┬────────────────────────┬─────────────────┬────────────────────────────────┐
│ Estado Atual    │ Evento                 │ Proximo Estado  │ Acao                           │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ INIT            │ config_loaded          │ CONNECTING      │ Iniciar conexao gRPC           │
│ INIT            │ config_error           │ FATAL           │ Log erro, exit(1)              │
│ INIT            │ cert_expired           │ FATAL           │ Log erro, solicitar reprovision│
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ CONNECTING      │ connected              │ REGISTERING     │ Enviar RegisterRequest         │
│ CONNECTING      │ connect_failed         │ BACKOFF         │ Incrementar retry counter      │
│ CONNECTING      │ cert_rejected          │ FATAL           │ Certificado invalido           │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ REGISTERING     │ registered             │ IDLE            │ Iniciar heartbeat loop         │
│ REGISTERING     │ rejected               │ FATAL           │ Server rejeitou agent          │
│ REGISTERING     │ version_mismatch       │ UPDATING        │ Baixar nova versao             │
│ REGISTERING     │ timeout                │ BACKOFF         │ Retry com backoff              │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ IDLE            │ deploy_command         │ DEPLOYING       │ Iniciar deploy                 │
│ IDLE            │ heartbeat_failed       │ RECONNECTING    │ Tentar reconectar              │
│ IDLE            │ shutdown_command       │ DRAINING        │ Iniciar graceful shutdown      │
│ IDLE            │ update_command         │ UPDATING        │ Baixar e aplicar update        │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ DEPLOYING       │ deploy_success         │ IDLE            │ Notificar server, cleanup      │
│ DEPLOYING       │ deploy_failed          │ ROLLBACK        │ Iniciar rollback               │
│ DEPLOYING       │ deploy_cancelled       │ IDLE            │ Cleanup parcial                │
│ DEPLOYING       │ connection_lost        │ DEPLOYING       │ Continuar deploy offline       │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ ROLLBACK        │ rollback_success       │ IDLE            │ Notificar server               │
│ ROLLBACK        │ rollback_failed        │ IDLE            │ Notificar server (erro critico)│
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ BACKOFF         │ backoff_complete       │ CONNECTING      │ Tentar reconectar              │
│ BACKOFF         │ max_retries            │ FATAL           │ Desistir, requer intervencao   │
│ BACKOFF         │ shutdown_signal        │ SHUTDOWN        │ Encerrar imediatamente         │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ RECONNECTING    │ reconnected            │ IDLE            │ Sincronizar estado             │
│ RECONNECTING    │ reconnect_failed       │ BACKOFF         │ Iniciar backoff                │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ UPDATING        │ update_success         │ SHUTDOWN        │ Reiniciar com nova versao      │
│ UPDATING        │ update_failed          │ IDLE            │ Continuar com versao atual     │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ DRAINING        │ all_deploys_done       │ SHUTDOWN        │ Encerrar gracefully            │
│ DRAINING        │ drain_timeout          │ SHUTDOWN        │ Forcar encerramento            │
├─────────────────┼────────────────────────┼─────────────────┼────────────────────────────────┤
│ FATAL           │ -                      │ -               │ Exit(1), requer intervencao    │
│ SHUTDOWN        │ -                      │ -               │ Exit(0), encerrado com sucesso │
└─────────────────┴────────────────────────┴─────────────────┴────────────────────────────────┘
```

### Maquina de Estados do Deploy

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              DEPLOY STATE MACHINE                                           │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

     ┌──────────┐                ┌──────────┐                ┌──────────┐
     │          │  clone ok      │          │  build ok      │          │
     │ GIT_SYNC │───────────────>│  BUILD   │───────────────>│  DEPLOY  │
     │          │                │          │                │          │
     └──────────┘                └──────────┘                └──────────┘
          │                           │                           │
          │ clone fail                │ build fail                │ deploy ok
          │                           │                           ▼
          │                           │                      ┌──────────┐
          │                           │                      │  HEALTH  │
          │                           │                      │  CHECK   │
          │                           │                      └──────────┘
          │                           │                           │
          │                           │                           ├── health ok ──> SUCCESS
          │                           │                           │
          ▼                           ▼                           ▼ health fail
     ┌─────────────────────────────────────────────────────────────────────┐
     │                            ROLLBACK                                 │
     │                                                                     │
     │  1. Stop new container                                              │
     │  2. Start previous container (if exists)                            │
     │  3. Verify rollback health                                          │
     │  4. Report failure with rollback status                             │
     │                                                                     │
     └─────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                                 ┌──────────┐
                                 │  FAILED  │
                                 └──────────┘
```

### Estados do Deploy

```go
type DeployState string

const (
    DeployStateInitializing DeployState = "INITIALIZING"  // Preparando ambiente
    DeployStateGitSync      DeployState = "GIT_SYNC"      // Clonando/atualizando repo
    DeployStateBuild        DeployState = "BUILD"         // Docker build
    DeployStateDeploy       DeployState = "DEPLOY"        // Docker run
    DeployStateHealthCheck  DeployState = "HEALTH_CHECK"  // Verificando saude
    DeployStateCleanup      DeployState = "CLEANUP"       // Limpando recursos antigos
    DeployStateSuccess      DeployState = "SUCCESS"       // Deploy concluido
    DeployStateRollback     DeployState = "ROLLBACK"      // Revertendo para versao anterior
    DeployStateFailed       DeployState = "FAILED"        // Deploy falhou
    DeployStateCancelled    DeployState = "CANCELLED"     // Deploy cancelado
)
```

### Estrategia de Backoff Exponencial

```go
type BackoffConfig struct {
    InitialInterval time.Duration // 1s
    MaxInterval     time.Duration // 5m
    Multiplier      float64       // 2.0
    MaxRetries      int           // 10
    Jitter          float64       // 0.1 (10%)
}

func DefaultBackoffConfig() BackoffConfig {
    return BackoffConfig{
        InitialInterval: 1 * time.Second,
        MaxInterval:     5 * time.Minute,
        Multiplier:      2.0,
        MaxRetries:      10,
        Jitter:          0.1,
    }
}

// Calculo do intervalo de backoff
// interval = min(initial * (multiplier ^ attempt), max) * (1 + random(-jitter, +jitter))
//
// Exemplo com config padrao:
// Attempt 1:  1s  * 2^0 = 1s   (+/- 10%) = 0.9s  - 1.1s
// Attempt 2:  1s  * 2^1 = 2s   (+/- 10%) = 1.8s  - 2.2s
// Attempt 3:  1s  * 2^2 = 4s   (+/- 10%) = 3.6s  - 4.4s
// Attempt 4:  1s  * 2^3 = 8s   (+/- 10%) = 7.2s  - 8.8s
// Attempt 5:  1s  * 2^4 = 16s  (+/- 10%) = 14.4s - 17.6s
// Attempt 6:  1s  * 2^5 = 32s  (+/- 10%) = 28.8s - 35.2s
// Attempt 7:  1s  * 2^6 = 64s  (+/- 10%) = 57.6s - 70.4s
// Attempt 8:  1s  * 2^7 = 128s (+/- 10%) = 115s  - 141s
// Attempt 9:  1s  * 2^8 = 256s (+/- 10%) = 230s  - 282s
// Attempt 10: 5m (max)         (+/- 10%) = 270s  - 330s
```

### Tratamento de Erros por Categoria

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              ERROR HANDLING MATRIX                                          │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────┬─────────────────┬────────────────────┬──────────────────────────────┐
│ Categoria de Erro      │ Retentavel?     │ Acao               │ Exemplos                     │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ TRANSIENT_NETWORK      │ Sim             │ Retry com backoff  │ Connection refused,          │
│                        │                 │                    │ Connection reset,            │
│                        │                 │                    │ DNS timeout                  │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ TRANSIENT_SERVER       │ Sim             │ Retry com backoff  │ gRPC UNAVAILABLE,            │
│                        │                 │                    │ gRPC RESOURCE_EXHAUSTED,     │
│                        │                 │                    │ HTTP 503, 502, 504           │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ TRANSIENT_TIMEOUT      │ Sim (limitado)  │ Retry 3x, depois   │ Context deadline exceeded,   │
│                        │                 │ reportar erro      │ gRPC DEADLINE_EXCEEDED       │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ AUTH_ERROR             │ Nao             │ Estado FATAL       │ gRPC UNAUTHENTICATED,        │
│                        │                 │                    │ Certificate expired,         │
│                        │                 │                    │ Certificate revoked          │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ PERMISSION_ERROR       │ Nao             │ Reportar erro      │ gRPC PERMISSION_DENIED,      │
│                        │                 │                    │ Docker permission denied     │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ VALIDATION_ERROR       │ Nao             │ Reportar erro      │ gRPC INVALID_ARGUMENT,       │
│                        │                 │                    │ Invalid config, Bad request  │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ NOT_FOUND              │ Nao             │ Reportar erro      │ gRPC NOT_FOUND,              │
│                        │                 │                    │ Image not found,             │
│                        │                 │                    │ Container not found          │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ RESOURCE_ERROR         │ Sim (limitado)  │ Retry 3x           │ Disk full,                   │
│                        │                 │                    │ Out of memory,               │
│                        │                 │                    │ No space left on device      │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ INTERNAL_ERROR         │ Depende         │ Log + Retry 1x     │ gRPC INTERNAL,               │
│                        │                 │                    │ Unexpected server error      │
├────────────────────────┼─────────────────┼────────────────────┼──────────────────────────────┤
│ CANCELLED              │ Nao             │ Cleanup + IDLE     │ gRPC CANCELLED,              │
│                        │                 │                    │ Context cancelled            │
└────────────────────────┴─────────────────┴────────────────────┴──────────────────────────────┘
```

### Protocolo de Reconexao

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              RECONNECTION PROTOCOL                                          │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

Cenario: Conexao perdida durante operacao normal (IDLE)

     Agent                                                           Server
       │                                                                │
       │ ─────── heartbeat ─────────────────────────────────────────>  │
       │                                                                │
       │ <───────── ack ───────────────────────────────────────────── │
       │                                                                │
       │                        [CONEXAO PERDIDA]                       │
       │                              X                                 │
       │                                                                │
       │ ─────── heartbeat ──────────X (timeout)                       │
       │                                                                │
       │  [Detecta perda: 3 heartbeats sem resposta]                   │
       │                                                                │
       │  Estado: IDLE -> RECONNECTING                                 │
       │                                                                │
       │  [Backoff: 1s]                                                │
       │                                                                │
       │ ─────── reconnect attempt 1 ───────X (fail)                   │
       │                                                                │
       │  [Backoff: 2s]                                                │
       │                                                                │
       │ ─────── reconnect attempt 2 ───────X (fail)                   │
       │                                                                │
       │  [Backoff: 4s]                                                │
       │                                                                │
       │ ─────── reconnect attempt 3 ─────────────────────────────>   │
       │                                                                │
       │ <───────── connected ──────────────────────────────────────  │
       │                                                                │
       │ ─────── RegisterRequest ─────────────────────────────────>   │
       │         (reconnect = true)                                    │
       │                                                                │
       │ <───────── RegisterResponse ────────────────────────────────  │
       │            (pending_deploys)                                  │
       │                                                                │
       │  Estado: RECONNECTING -> IDLE                                 │
       │                                                                │
       │  [Processar deploys pendentes se houver]                      │
       │                                                                │
```

### Protocolo de Deploy com Falha e Rollback

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              DEPLOY WITH ROLLBACK PROTOCOL                                  │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

Cenario: Deploy falha no health check

     Agent                                                           Server
       │                                                                │
       │ <───────── DeployCommand ─────────────────────────────────── │
       │            (app: my-api, image: v2)                           │
       │                                                                │
       │  Estado: IDLE -> DEPLOYING                                    │
       │  Deploy Estado: INITIALIZING                                  │
       │                                                                │
       │ ─────── DeployLog (stage: GIT_SYNC) ─────────────────────>   │
       │                                                                │
       │  [git clone/pull...]                                          │
       │                                                                │
       │ ─────── DeployLog (stage: BUILD) ─────────────────────────>   │
       │                                                                │
       │  [docker build...]                                            │
       │                                                                │
       │ ─────── DeployLog (stage: DEPLOY) ────────────────────────>   │
       │                                                                │
       │  [docker stop old container]                                  │
       │  [docker run new container]                                   │
       │                                                                │
       │ ─────── DeployLog (stage: HEALTH_CHECK) ──────────────────>   │
       │                                                                │
       │  [GET /health - timeout]                                      │
       │  [GET /health - 500]                                          │
       │  [GET /health - 500]                                          │
       │                                                                │
       │ ─────── DeployLog (level: ERROR, "health check failed") ──>   │
       │                                                                │
       │  Deploy Estado: HEALTH_CHECK -> ROLLBACK                      │
       │                                                                │
       │ ─────── DeployLog (stage: ROLLBACK, "starting rollback") ─>   │
       │                                                                │
       │  [docker stop v2 container]                                   │
       │  [docker start v1 container (previous)]                       │
       │  [verify v1 health - ok]                                      │
       │                                                                │
       │ ─────── DeployLog (stage: ROLLBACK, "rollback success") ──>   │
       │                                                                │
       │ ─────── DeployResponse ───────────────────────────────────>   │
       │         (success: false,                                      │
       │          error: HEALTH_CHECK_FAILED,                          │
       │          rollback_status: SUCCESS)                            │
       │                                                                │
       │  Estado: DEPLOYING -> IDLE                                    │
       │                                                                │
```

### Protocolo de Graceful Shutdown

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              GRACEFUL SHUTDOWN PROTOCOL                                     │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

Cenario: Agent recebe SIGTERM durante deploy

     Agent                                                           Server
       │                                                                │
       │  [Executando deploy...]                                       │
       │  Estado: DEPLOYING                                            │
       │                                                                │
       │  [SIGTERM recebido]                                           │
       │                                                                │
       │  Estado: DEPLOYING -> DRAINING                                │
       │  [Nao aceita novos deploys]                                   │
       │  [Continua deploy atual]                                      │
       │                                                                │
       │ ─────── Heartbeat (state: DRAINING) ──────────────────────>   │
       │                                                                │
       │ <───────── HeartbeatResponse (no new commands) ─────────────  │
       │                                                                │
       │  [Deploy continua...]                                         │
       │  [Deploy completo]                                            │
       │                                                                │
       │ ─────── DeployResponse (success) ─────────────────────────>   │
       │                                                                │
       │ ─────── Unregister (reason: SHUTDOWN) ────────────────────>   │
       │                                                                │
       │ <───────── UnregisterResponse ────────────────────────────── │
       │                                                                │
       │  [Fechar conexao gRPC]                                        │
       │  Estado: DRAINING -> SHUTDOWN                                 │
       │  [Exit 0]                                                     │
       │                                                                │


Cenario: Timeout no drain (deploy muito longo)

     Agent                                                           Server
       │                                                                │
       │  [SIGTERM recebido]                                           │
       │  Estado: DEPLOYING -> DRAINING                                │
       │  [Inicia drain timeout: 5 minutos]                            │
       │                                                                │
       │  [Deploy continua...]                                         │
       │  [...]                                                        │
       │  [Drain timeout expirado!]                                    │
       │                                                                │
       │ ─────── DeployLog (level: WARN, "drain timeout") ─────────>   │
       │                                                                │
       │ ─────── DeployResponse (success: false,                       │
       │                         error: CANCELLED,                     │
       │                         reason: "agent shutdown") ──────────> │
       │                                                                │
       │ ─────── Unregister (reason: SHUTDOWN_TIMEOUT) ────────────>   │
       │                                                                │
       │  Estado: DRAINING -> SHUTDOWN                                 │
       │  [Exit 0]                                                     │
       │                                                                │
```

### Protocolo de Auto-Update

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              AUTO-UPDATE PROTOCOL                                           │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

     Agent                                                           Server
       │                                                                │
       │ ─────── RegisterRequest (version: 1.0.0) ─────────────────>   │
       │                                                                │
       │ <───────── RegisterResponse ────────────────────────────────  │
       │            (update_available:                                 │
       │              version: 1.1.0,                                  │
       │              url: https://...,                                │
       │              required: false)                                 │
       │                                                                │
       │  [Agenda update para proximo periodo de inatividade]          │
       │                                                                │
       │  [...operacao normal...]                                      │
       │                                                                │
       │  [Periodo de inatividade detectado]                           │
       │  [Nenhum deploy ativo]                                        │
       │  Estado: IDLE -> UPDATING                                     │
       │                                                                │
       │ ─────── Heartbeat (state: UPDATING) ──────────────────────>   │
       │                                                                │
       │  [Download nova versao]                                       │
       │  [Verificar checksum]                                         │
       │  [Substituir binario]                                         │
       │                                                                │
       │ ─────── Unregister (reason: UPDATE) ──────────────────────>   │
       │                                                                │
       │  [Reiniciar via systemd]                                      │
       │  Estado: UPDATING -> SHUTDOWN                                 │
       │                                                                │
       │  [systemd reinicia agent]                                     │
       │                                                                │
       │ ─────── RegisterRequest (version: 1.1.0) ─────────────────>   │
       │                                                                │
       │ <───────── RegisterResponse (update_available: null) ───────  │
       │                                                                │
```

### Implementacao da State Machine

```go
package agent

import (
    "context"
    "sync"
    "time"
)

type StateMachine struct {
    mu           sync.RWMutex
    currentState AgentState
    prevState    AgentState

    // Channels para eventos
    events       chan Event
    transitions  chan Transition

    // Handlers por estado
    handlers     map[AgentState]StateHandler

    // Configuracao
    config       *Config

    // Metricas
    stateEnterTime time.Time
    transitionCount int64
}

type Event struct {
    Type    EventType
    Payload interface{}
    Error   error
}

type EventType string

const (
    EventConfigLoaded     EventType = "config_loaded"
    EventConfigError      EventType = "config_error"
    EventConnected        EventType = "connected"
    EventConnectFailed    EventType = "connect_failed"
    EventRegistered       EventType = "registered"
    EventRejected         EventType = "rejected"
    EventDeployCommand    EventType = "deploy_command"
    EventDeploySuccess    EventType = "deploy_success"
    EventDeployFailed     EventType = "deploy_failed"
    EventHeartbeatFailed  EventType = "heartbeat_failed"
    EventConnectionLost   EventType = "connection_lost"
    EventReconnected      EventType = "reconnected"
    EventBackoffComplete  EventType = "backoff_complete"
    EventMaxRetries       EventType = "max_retries"
    EventShutdownSignal   EventType = "shutdown_signal"
    EventUpdateCommand    EventType = "update_command"
    EventUpdateSuccess    EventType = "update_success"
    EventUpdateFailed     EventType = "update_failed"
)

type Transition struct {
    From      AgentState
    To        AgentState
    Event     EventType
    Timestamp time.Time
}

type StateHandler interface {
    Enter(ctx context.Context, sm *StateMachine) error
    Exit(ctx context.Context, sm *StateMachine) error
    HandleEvent(ctx context.Context, sm *StateMachine, event Event) (AgentState, error)
}

func NewStateMachine(config *Config) *StateMachine {
    sm := &StateMachine{
        currentState: StateInit,
        events:       make(chan Event, 100),
        transitions:  make(chan Transition, 100),
        handlers:     make(map[AgentState]StateHandler),
        config:       config,
    }

    // Registrar handlers
    sm.handlers[StateInit] = &InitHandler{}
    sm.handlers[StateConnecting] = &ConnectingHandler{}
    sm.handlers[StateRegistering] = &RegisteringHandler{}
    sm.handlers[StateIdle] = &IdleHandler{}
    sm.handlers[StateDeploying] = &DeployingHandler{}
    sm.handlers[StateRollback] = &RollbackHandler{}
    sm.handlers[StateBackoff] = &BackoffHandler{}
    sm.handlers[StateReconnecting] = &ReconnectingHandler{}
    sm.handlers[StateUpdating] = &UpdatingHandler{}
    sm.handlers[StateDraining] = &DrainingHandler{}

    return sm
}

func (sm *StateMachine) Run(ctx context.Context) error {
    // Entrar no estado inicial
    if handler, ok := sm.handlers[sm.currentState]; ok {
        if err := handler.Enter(ctx, sm); err != nil {
            return err
        }
    }

    sm.stateEnterTime = time.Now()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()

        case event := <-sm.events:
            if err := sm.handleEvent(ctx, event); err != nil {
                // Log error but continue
                sm.logError(err)
            }
        }
    }
}

func (sm *StateMachine) handleEvent(ctx context.Context, event Event) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    handler, ok := sm.handlers[sm.currentState]
    if !ok {
        return fmt.Errorf("no handler for state %s", sm.currentState)
    }

    nextState, err := handler.HandleEvent(ctx, sm, event)
    if err != nil {
        return err
    }

    if nextState != sm.currentState {
        return sm.transitionTo(ctx, nextState, event.Type)
    }

    return nil
}

func (sm *StateMachine) transitionTo(ctx context.Context, newState AgentState, eventType EventType) error {
    // Exit current state
    if handler, ok := sm.handlers[sm.currentState]; ok {
        if err := handler.Exit(ctx, sm); err != nil {
            return err
        }
    }

    // Record transition
    transition := Transition{
        From:      sm.currentState,
        To:        newState,
        Event:     eventType,
        Timestamp: time.Now(),
    }

    select {
    case sm.transitions <- transition:
    default:
        // Channel full, log and continue
    }

    // Update state
    sm.prevState = sm.currentState
    sm.currentState = newState
    sm.stateEnterTime = time.Now()
    sm.transitionCount++

    // Enter new state
    if handler, ok := sm.handlers[newState]; ok {
        if err := handler.Enter(ctx, sm); err != nil {
            return err
        }
    }

    return nil
}

func (sm *StateMachine) SendEvent(event Event) {
    select {
    case sm.events <- event:
    default:
        // Channel full, log warning
    }
}

func (sm *StateMachine) CurrentState() AgentState {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.currentState
}

func (sm *StateMachine) StateUptime() time.Duration {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return time.Since(sm.stateEnterTime)
}
```

### Exemplo de Handler: IdleHandler

```go
package agent

import (
    "context"
    "time"
)

type IdleHandler struct {
    heartbeatTicker *time.Ticker
    heartbeatFails  int
}

func (h *IdleHandler) Enter(ctx context.Context, sm *StateMachine) error {
    // Iniciar heartbeat loop
    h.heartbeatTicker = time.NewTicker(sm.config.HeartbeatInterval)
    h.heartbeatFails = 0

    go h.heartbeatLoop(ctx, sm)

    return nil
}

func (h *IdleHandler) Exit(ctx context.Context, sm *StateMachine) error {
    if h.heartbeatTicker != nil {
        h.heartbeatTicker.Stop()
    }
    return nil
}

func (h *IdleHandler) HandleEvent(ctx context.Context, sm *StateMachine, event Event) (AgentState, error) {
    switch event.Type {
    case EventDeployCommand:
        return StateDeploying, nil

    case EventHeartbeatFailed:
        h.heartbeatFails++
        if h.heartbeatFails >= sm.config.MaxHeartbeatFails {
            return StateReconnecting, nil
        }
        return StateIdle, nil

    case EventShutdownSignal:
        return StateDraining, nil

    case EventUpdateCommand:
        return StateUpdating, nil

    case EventConnectionLost:
        return StateReconnecting, nil

    default:
        return StateIdle, nil
    }
}

func (h *IdleHandler) heartbeatLoop(ctx context.Context, sm *StateMachine) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-h.heartbeatTicker.C:
            if err := sm.sendHeartbeat(ctx); err != nil {
                sm.SendEvent(Event{
                    Type:  EventHeartbeatFailed,
                    Error: err,
                })
            } else {
                h.heartbeatFails = 0 // Reset on success
            }
        }
    }
}
```

## Resumo da Arquitetura

| Componente        | Tecnologia              | Responsabilidade               |
| ----------------- | ----------------------- | ------------------------------ |
| **Protocolo**     | gRPC + Protocol Buffers | Comunicacao tipada e eficiente |
| **Seguranca**     | mTLS (TLS 1.3)          | Autenticacao mutua             |
| **Certificados**  | ECDSA P-256             | Leves e seguros                |
| **Streaming**     | gRPC Bidirectional      | Logs em tempo real             |
| **Versionamento** | Package versioning      | Compatibilidade backward       |
| **State Machine** | Event-driven FSM        | Gestao de ciclo de vida        |

## Proximos Passos

1. Criar arquivos .proto com buf
2. Gerar codigo Go (protoc/buf)
3. Implementar PKI (CA, certificados)
4. Implementar gRPC Server
5. Implementar Agent com gRPC client
6. Implementar SSH Provisioner
7. Integrar no frontend

# Deploy Remoto com Agent - Guia de Uso

> **Tutorial passo a passo:** Para um guia detalhado do início ao fim, consulte [REMOTE_SERVER_TUTORIAL.md](./REMOTE_SERVER_TUTORIAL.md).

## 1. Pré-requisitos

### 1.1 Instalar buf

```bash
go install github.com/bufbuild/buf/cmd/buf@v1.38.0
```

### 1.2 Gerar código dos protos

```bash
cd apps/proto
BUF_CACHE_DIR=$PWD/../.buf-cache buf generate
```

---

## 2. Configuração do Backend

### 2.1 Variáveis de ambiente

| Variável             | Obrigatório | Descrição                                                                   |
| -------------------- | ----------- | --------------------------------------------------------------------------- |
| TOKEN_ENCRYPTION_KEY | Sim         | Chave base64 32 bytes para criptografar chave SSH                           |
| GRPC_ENABLED         | Não         | true para habilitar gRPC (default: false)                                   |
| GRPC_PORT            | Não         | Porta gRPC (default: 50051)                                                 |
| GRPC_SERVER_ADDR     | Sim\*       | Endereço do backend acessível pelo agent (ex: paasdeploy.example.com:50051) |
| AGENT_BINARY_PATH    | Não         | Caminho do binário do agent para provisionamento                            |
| AGENT_GRPC_PORT      | Não         | Porta do agent para health check e controle (default: 50052)                |

\* Necessário quando agent está em outro host

### 2.2 Build

```bash
cd apps/backend && go build ./...
```

---

## 3. Build do Agent

```bash
cd apps/agent
go build -o ../../dist/agent ./cmd/agent
```

---

## 4. Fluxo de Uso

### 4.0 Pré-configuração no servidor remoto

1. Escolha um usuário (ex.: `deploy`) que terá acesso SSH.
2. Habilite **linger** para permitir systemd em modo usuário (mantém serviços ativos após logout):

```bash
sudo loginctl enable-linger deploy
```

3. Confira se o usuário consegue executar `systemctl --user status` sem erro.
4. Garanta espaço em disco no `$HOME` para instalar o agent em `~/paasdeploy-agent`.

### 4.1 Adicionar Servidor

1. Menu **Servers** > **Add Server**
2. Preencher: Nome, Host, SSH Port (22), SSH User e credenciais
   - Aceitamos **SSH Private Key** (recomendado) ou **SSH Password** (opcional).
   - As credenciais são armazenadas criptografadas e não ficam visíveis após o cadastro.

### 4.2 Provisionar

1. No card do servidor, clicar **Provision**
2. Backend conecta via SSH (usando a chave ou senha fornecida), instala certs e agent em `~/paasdeploy-agent`
3. Uma unidade `systemd --user` é criada em `~/.config/systemd/user/paasdeploy-agent.service`; o serviço é habilitado e iniciado (requer linger habilitado)
4. Agent inicia e conecta ao backend via mTLS

### 4.3 Associar App ao Servidor

1. Editar app > selecionar Servidor (em desenvolvimento)
2. Deploys futuros serão enviados ao agent remoto

### 4.4 Verificar no servidor remoto

```bash
systemctl --user status paasdeploy-agent
journalctl --user -u paasdeploy-agent -f
```

### 4.5 Teste de conectividade (health check)

Endpoint:

```bash
GET /paas-deploy/v1/servers/:id/health
```

Resposta esperada:

```json
{ "status": "ok", "latencyMs": 12 }
```

---

## 5. Limitações Atuais

- **Agent port**: Backend conecta ao agent em `host:<AGENT_GRPC_PORT>`.
- **ExecuteDeploy**: Agent ainda implementa stub; deploy real em desenvolvimento.

---

## 6. Notas

- **CA persistida**: A CA é salva no banco de dados; reinício do backend não invalida os certs.

## 7. Estrutura de Arquivos Instalados no Servidor

```
~/paasdeploy-agent/
├── agent          # binário
├── ca.pem         # certificado CA
├── cert.pem       # certificado do agent
└── key.pem        # chave privada
```

## 8. Executar Agent Manualmente

```bash
cd ~/paasdeploy-agent
./agent -server-addr=backend:50051 -server-id=<UUID_DO_SERVIDOR> \
  -ca-cert=ca.pem -cert=cert.pem -key=key.pem -agent-port=50052
```

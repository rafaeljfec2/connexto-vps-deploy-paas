# Deploy Remoto com Agent - Guia de Uso

> **Tutorial passo a passo:** Para um guia detalhado do inicio ao fim, consulte [REMOTE_SERVER_TUTORIAL.md](./REMOTE_SERVER_TUTORIAL.md).

## 1. Pre-requisitos

### 1.1 Instalar buf

```bash
go install github.com/bufbuild/buf/cmd/buf@v1.38.0
```

### 1.2 Gerar codigo dos protos

```bash
cd apps/proto
BUF_CACHE_DIR=$PWD/../.buf-cache buf generate
```

---

## 2. Configuracao do Backend

### 2.1 Variaveis de ambiente

| Variavel             | Obrigatorio | Descricao                                                                   |
| -------------------- | ----------- | --------------------------------------------------------------------------- |
| TOKEN_ENCRYPTION_KEY | Sim         | Chave base64 32 bytes para criptografar chave SSH                           |
| GRPC_ENABLED         | Nao         | true para habilitar gRPC (default: false)                                   |
| GRPC_PORT            | Nao         | Porta gRPC (default: 50051)                                                 |
| GRPC_SERVER_ADDR     | Sim\*       | Endereco do backend acessivel pelo agent (ex: paasdeploy.example.com:50051) |
| AGENT_BINARY_PATH    | Nao         | Caminho do binario do agent para provisionamento                            |
| AGENT_GRPC_PORT      | Nao         | Porta do agent para health check e controle (default: 50052)                |

\* Necessario quando agent esta em outro host

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

### 4.0 Pre-configuracao no servidor remoto

1. Escolha um usuario (ex.: `deploy`) que tera acesso SSH.
2. Habilite **linger** para permitir systemd em modo usuario (mantem servicos ativos apos logout):

```bash
sudo loginctl enable-linger deploy
```

3. Configure **sudo sem senha** se o usuario nao for root:

```bash
echo "deploy ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/deploy
```

4. Confira se o usuario consegue executar `systemctl --user status` sem erro.
5. Libere as portas necessarias no firewall: 22, 80, 443, 50052.

### 4.1 Adicionar Servidor

1. Menu **Servers** > **Add Server**
2. Preencher: Nome, Host, SSH Port (22), SSH User e credenciais
   - Aceitamos **SSH Private Key** (recomendado) ou **SSH Password** (opcional).
   - As credenciais sao armazenadas criptografadas e nao ficam visiveis apos o cadastro.
3. Preencher **ACME Email** para habilitar TLS automatico via Traefik (Let's Encrypt)

Cada usuario so ve e gerencia seus proprios servidores.

### 4.2 Provisionar

1. No card do servidor, clicar **Provision**
2. Backend conecta via SSH e executa automaticamente:
   - **Docker**: verifica se instalado; se nao, instala via `get.docker.com`
   - **Rede Docker**: cria rede `paasdeploy` para comunicacao entre containers
   - **Traefik**: instala e configura reverse proxy com TLS automatico (se ACME Email fornecido)
   - **Agent**: copia binario, certificados mTLS e configura servico systemd
3. Agent inicia e conecta ao backend via gRPC + mTLS
4. Status muda para **Online** quando o agent envia o primeiro heartbeat

O provisionamento e idempotente: rodar novamente nao duplica recursos.

### 4.3 Associar App ao Servidor

1. Na criacao da app, selecione o servidor remoto no step "Deploy Target"
2. Deploys futuros serao enviados ao agent remoto

### 4.4 Verificar no servidor remoto

```bash
systemctl --user status paasdeploy-agent
journalctl --user -u paasdeploy-agent -f
docker ps  # mostra Traefik e apps deployadas
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

## 5. O que o provisionamento instala automaticamente

| Componente  | Acao                                                    | Condicao                         |
| ----------- | ------------------------------------------------------- | -------------------------------- |
| Docker      | Instala via get.docker.com + habilita daemon             | Se nao encontrado                |
| Rede Docker | Cria rede `paasdeploy`                                  | Se nao existir                   |
| Traefik     | Container com portas 80, 443, 50051, 8081 + Let's Encrypt | Se ACME Email fornecido        |
| Agent       | Binario + certificados mTLS + servico systemd           | Sempre (sobrescreve se existir)  |

Cada componente tem timeout individual:
- Docker install: 10 minutos
- Docker start: 30 segundos
- Rede Docker: 30 segundos
- Traefik setup: 5 minutos

Se o Traefik ja estiver rodando com versao diferente da esperada (traefik:v3.2), o provisionamento faz upgrade automatico (pull + recreate).

---

## 6. Notas

- **CA persistida**: A CA e salva no banco de dados; reinicio do backend nao invalida os certs.
- **Multi-tenancy**: Servidores sao isolados por usuario. Cada usuario so ve seus proprios servidores. O agent e o gRPC operam no nivel do sistema (sem conceito de usuario).
- **Sudo**: Se conectado como usuario nao-root, comandos privilegiados usam `sudo -n` (sem prompt de senha).

## 7. Estrutura de Arquivos Instalados no Servidor

```
~/paasdeploy-agent/
  agent          # binario
  ca.pem         # certificado CA
  cert.pem       # certificado do agent
  key.pem        # chave privada

/opt/traefik/
  traefik.yml    # configuracao estatica
  letsencrypt/
    acme.json    # certificados TLS (auto-gerado)
```

## 8. Executar Agent Manualmente

```bash
cd ~/paasdeploy-agent
./agent -server-addr=backend:50051 -server-id=<UUID_DO_SERVIDOR> \
  -ca-cert=ca.pem -cert=cert.pem -key=key.pem -agent-port=50052
```

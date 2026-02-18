# Tutorial: Configurar Servidor Remoto para Deploy

Passo a passo para preparar um servidor (VPS) e provisionar o agent do FlowDeploy automaticamente.

---

## Visao geral

1. Preparar o servidor remoto (VPS)
2. Configurar o backend do FlowDeploy
3. Cadastrar o servidor no painel
4. Provisionar e verificar

---

## Como o provisionamento funciona

Quando voce clica em **Provision** no painel, o backend executa as seguintes etapas automaticamente via SSH:

```
Backend (SSH) ──────────────────────────────> Servidor Remoto
  |
  |  1. Conecta via SSH (chave ou senha)
  |  2. Detecta ambiente (home dir, UID, root ou sudo)
  |
  |  3. Verifica Docker
  |     - Se nao instalado: instala via get.docker.com (timeout: 10 min)
  |     - Se nao rodando: inicia e habilita via systemctl
  |     - Loga versao encontrada
  |
  |  4. Cria rede Docker "paasdeploy" (se nao existir)
  |
  |  5. Configura Traefik (se ACME Email fornecido)
  |     - Cria /opt/traefik/ e /opt/traefik/letsencrypt/
  |     - Gera traefik.yml (entrypoints: 80, 443, 50051, 8081)
  |     - Inicia container Traefik com Let's Encrypt
  |     - Se ja rodando com versao correta: pula
  |     - Se rodando com versao antiga: faz upgrade automatico
  |
  |  6. Copia binario do agent + certificados mTLS
  |     - ~/paasdeploy-agent/agent (binario)
  |     - ~/paasdeploy-agent/ca.pem, cert.pem, key.pem
  |
  |  7. Cria servico systemd (user-level)
  |     - ~/.config/systemd/user/paasdeploy-agent.service
  |     - Habilita e inicia o agent
  |
  |  8. Agent conecta ao backend via gRPC + mTLS
  |     - Envia Register + Heartbeats periodicos
  |     - Status muda para "Online" no painel
```

Cada etapa tem timeout individual: Docker install (10 min), Docker start (30s), Traefik setup (5 min), network (30s).

Se o usuario SSH nao for root, o provisionador usa `sudo -n` automaticamente para comandos privilegiados (instalar Docker, configurar Traefik em /opt).

---

## Parte 1: Preparar o servidor remoto (VPS)

Execute estes comandos **no servidor remoto** (via SSH com usuario root ou com sudo).

### 1.1 Criar usuario para o agent (se ainda nao existir)

```bash
sudo adduser deploy
```

Se preferir usar um usuario existente (ex.: `oab-api`), pule para o passo 1.2.

### 1.2 Habilitar linger para o usuario

O linger permite que servicos `systemd --user` continuem rodando apos logout. **Obrigatorio** para o agent.

```bash
sudo loginctl enable-linger <USUARIO>
```

Exemplo para usuario `deploy`:

```bash
sudo loginctl enable-linger deploy
```

### 1.3 Configurar sudo sem senha (se nao for root)

Se voce conecta como usuario nao-root, o provisionamento precisa de `sudo` para instalar Docker e Traefik. Configure sudo sem senha para o usuario:

```bash
echo "deploy ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/deploy
```

Alternativa mais restrita (apenas Docker e systemctl):

```bash
cat << 'EOF' | sudo tee /etc/sudoers.d/deploy
deploy ALL=(ALL) NOPASSWD: /usr/bin/sh -c *get.docker.com*
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl start docker
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl enable docker
deploy ALL=(ALL) NOPASSWD: /usr/sbin/usermod -aG docker *
deploy ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /opt/traefik*
deploy ALL=(ALL) NOPASSWD: /usr/bin/tee /opt/traefik/*
EOF
```

Se voce conecta como **root**, o sudo nao e necessario.

### 1.4 Verificar acesso SSH

O usuario precisa conseguir fazer login via SSH com **chave privada** ou **senha**.

**Opcao A - Chave privada (recomendado):**

- Adicione a chave publica em `~/.ssh/authorized_keys` do usuario.
- Teste: `ssh deploy@IP_DO_SERVIDOR -p PORTA` (deve conectar sem pedir senha).

**Opcao B - Senha:**

- Certifique-se de que o usuario tem senha definida.
- O FlowDeploy aceita senha; ela sera armazenada criptografada.

### 1.5 Confirmar systemd user

Entre no servidor com o usuario escolhido e teste:

```bash
ssh deploy@IP_DO_SERVIDOR -p PORTA
systemctl --user status
```

Se aparecer "Failed to connect to bus", o linger nao esta habilitado. Volte ao passo 1.2.

### 1.6 Liberar portas no firewall

O Traefik e o agent precisam das seguintes portas acessiveis:

| Porta | Protocolo | Uso                        | Obrigatorio |
| ----- | --------- | -------------------------- | ----------- |
| 22    | TCP       | SSH (provisionamento)      | Sim         |
| 80    | TCP       | HTTP (Traefik)             | Sim         |
| 443   | TCP       | HTTPS (Traefik + TLS)      | Sim         |
| 50051 | TCP       | gRPC (Traefik -> backend)  | Se usar gRPC via Traefik |
| 50052 | TCP       | gRPC (agent health check)  | Sim         |
| 8081  | TCP       | Traefik Dashboard          | Opcional    |

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 50052/tcp
sudo ufw allow 8081/tcp
```

---

## Parte 2: Configurar o backend do FlowDeploy

Execute no **host onde o backend esta rodando**.

### 2.1 Gerar chave de criptografia

```bash
openssl rand -base64 32
```

Use o resultado em `TOKEN_ENCRYPTION_KEY` no `.env`.

### 2.2 Gerar binario do agent

**Desenvolvimento local:**

```bash
cd apps/agent
go build -o ../../dist/agent ./cmd/agent
```

**Producao (Docker):** A imagem do backend ja inclui o agent em `/app/agent`. Configure `AGENT_BINARY_PATH=/app/agent` no container.

### 2.3 Configurar variaveis no `.env`

| Variavel             | Valor exemplo                            | Obrigatorio |
| -------------------- | ---------------------------------------- | ----------- |
| TOKEN_ENCRYPTION_KEY | (saida do `openssl rand -base64 32`)     | Sim         |
| GRPC_ENABLED         | true                                     | Sim         |
| GRPC_PORT            | 50051                                    | Sim         |
| GRPC_SERVER_ADDR     | host:50051 (endereco acessivel pela VPS) | Sim         |
| AGENT_BINARY_PATH    | /caminho/absoluto/para/dist/agent        | Sim         |
| AGENT_GRPC_PORT      | 50052                                    | Sim         |

### 2.4 Reiniciar o backend

Apos alterar o `.env`, reinicie o backend para carregar as novas variaveis.

---

## Parte 3: Cadastrar o servidor no painel

### 3.1 Acessar o FlowDeploy

Abra o painel (ex.: `http://localhost:3000`), faca login e va em **Servers** > **Add Server**.

Cada usuario so ve seus proprios servidores. Servidores cadastrados por um usuario nao aparecem para outros usuarios.

### 3.2 Preencher o formulario

| Campo                       | Descricao                                             |
| --------------------------- | ----------------------------------------------------- |
| Name                        | Nome amigavel (ex.: production, staging)              |
| Host                        | IP ou hostname da VPS                                 |
| SSH Port                    | Porta SSH (padrao 22)                                 |
| SSH User                    | Usuario do passo 1.1 (ex.: deploy, root)              |
| SSH Key                     | Chave privada completa (opcional se usar senha)       |
| SSH Password                | Senha do usuario (opcional se usar chave)             |
| ACME Email (Let's Encrypt)  | Email para certificados TLS automaticos via Traefik   |

- E necessario informar **chave** ou **senha**; os dois podem ser usados juntos.
- O **ACME Email** e necessario para que o Traefik configure TLS automatico. Sem ele, o Traefik nao sera instalado e suas aplicacoes nao terao HTTPS.

### 3.3 Salvar

Apos clicar em **Add Server**, o servidor aparece com status **Pending**.

---

## Parte 4: Provisionar e verificar

### 4.1 Provisionar

No card do servidor, clique em **Provision**.

O provisionamento acontece em tempo real com feedback via SSE (Server-Sent Events). Voce vera o progresso de cada etapa no painel:

1. `ssh_connect` - Conectando via SSH
2. `remote_env` - Verificando ambiente (home dir, UID)
3. `docker_check` / `docker_install` - Verificando/instalando Docker
4. `docker_start` - Verificando/iniciando Docker daemon
5. `docker_network` - Criando rede Docker
6. `traefik_check` / `traefik_install` - Verificando/instalando Traefik
7. `sftp_client` - Conectando SFTP
8. `install_dir` - Criando diretorios
9. `agent_certs` - Gerando e instalando certificados mTLS
10. `agent_binary` - Copiando binario do agent
11. `systemd_unit` - Configurando servico systemd
12. `start_agent` - Iniciando agent

O status muda para **Online** quando o agent se registrar e enviar heartbeat.

### 4.2 Verificar no servidor remoto

Entre na VPS e confira:

```bash
ls ~/paasdeploy-agent
# agent  ca.pem  cert.pem  key.pem

systemctl --user status paasdeploy-agent

journalctl --user -u paasdeploy-agent -f
```

Verificar Docker:

```bash
docker --version
docker ps  # deve mostrar Traefik rodando
docker network ls  # deve mostrar rede "paasdeploy"
```

Verificar Traefik:

```bash
docker inspect traefik --format '{{.Config.Image}}'
# deve mostrar traefik:v3.2

curl -s http://localhost:8081/api/overview | head -c 200
# deve retornar JSON do dashboard
```

### 4.3 Testar conectividade

No painel ou via API:

```
GET /paas-deploy/v1/servers/:id/health
```

Resposta esperada: `{ "status": "ok", "latencyMs": ... }`.

---

## Resolucao de problemas

### Erros de conexao SSH

| Problema                   | Causa provavel                 | Solucao                                       |
| -------------------------- | ------------------------------ | --------------------------------------------- |
| `ssh connect: i/o timeout` | Firewall bloqueando porta SSH  | Liberar porta SSH (22 ou a configurada)       |
| `unable to authenticate`   | Chave/senha incorretos         | Conferir credenciais; testar `ssh` manualmente |
| `provision failed: EOF`    | Conexao caiu durante transfer  | Reprovisionar; verificar estabilidade da rede |

### Erros de Docker

| Problema                                      | Causa provavel                | Solucao                                              |
| --------------------------------------------- | ----------------------------- | ---------------------------------------------------- |
| `install docker: command timed out after 10m`  | Internet lenta ou sem acesso  | Verificar acesso a internet; instalar Docker manualmente |
| `start docker daemon: ... sudo -n`            | Sudo sem senha nao configurado | Configurar sudoers (passo 1.3)                       |
| `create docker network: permission denied`    | Usuario sem acesso ao Docker  | `sudo usermod -aG docker <usuario>` e reconectar     |

### Erros de Traefik

| Problema                                  | Causa provavel                   | Solucao                                           |
| ----------------------------------------- | -------------------------------- | ------------------------------------------------- |
| Traefik nao instalado                     | ACME Email nao informado         | Editar servidor e adicionar ACME Email             |
| `write traefik config: ... sudo -n`       | Sudo sem senha nao configurado   | Configurar sudoers (passo 1.3)                    |
| Porta 80/443 ja em uso                    | Outro servico (nginx, apache)    | Parar servico conflitante: `sudo systemctl stop nginx` |
| TLS nao funciona                          | DNS nao aponta para o servidor   | Configurar DNS A record apontando para o IP do servidor |

### Erros de Agent

| Problema                        | Causa provavel                    | Solucao                                          |
| ------------------------------- | --------------------------------- | ------------------------------------------------ |
| `Unit ... could not be found`   | Linger nao habilitado             | Executar `sudo loginctl enable-linger <usuario>` |
| `Failed to connect to bus`      | Linger nao habilitado             | Idem acima                                       |
| Status permanece "Provisioning" | Processo travado ou em andamento  | Ver logs do backend; conferir se SSH nao caiu     |
| Status permanece "Pending"      | Provision nao foi executado       | Clicar em Provision no card do servidor          |
| Agent nao conecta ao backend    | Porta 50051 bloqueada no firewall | Liberar porta 50051 no firewall do backend       |
| Agent nao conecta ao backend    | GRPC_SERVER_ADDR incorreto        | Verificar se o endereco e acessivel da VPS       |

### Erros de permissao (multi-tenancy)

| Problema                         | Causa provavel                   | Solucao                                    |
| -------------------------------- | -------------------------------- | ------------------------------------------ |
| Servidor nao aparece na lista    | Servidor pertence a outro usuario | Cada usuario so ve seus proprios servidores |
| `server not found` ao acessar    | Tentando acessar servidor alheio  | Verificar se o servidor e seu              |
| Servidor sumiu apos atualizacao  | Migracao atribuiu a outro usuario | Verificar user_id no banco de dados        |

---

## Reprovisionar um servidor

Se algo deu errado, voce pode reprovisionar:

1. No painel, va em **Servers** e clique no servidor
2. Clique em **Provision** novamente

O provisionamento e **idempotente**:
- Docker: se ja instalado e rodando, pula instalacao
- Rede Docker: se ja existe, pula criacao
- Traefik: se ja rodando com versao correta, pula; se versao antiga, faz upgrade
- Agent: sobrescreve binario e certificados, reinicia servico

---

## Checklist rapido

**No servidor remoto (uma vez):**

- [ ] Usuario criado ou existente
- [ ] `sudo loginctl enable-linger <usuario>` executado
- [ ] SSH funcionando (chave ou senha)
- [ ] `systemctl --user status` funciona sem erro
- [ ] Sudo sem senha configurado (se nao for root)
- [ ] Portas liberadas no firewall (22, 80, 443, 50052)

**No backend:**

- [ ] `TOKEN_ENCRYPTION_KEY` definida
- [ ] `AGENT_BINARY_PATH` apontando para o binario do agent
- [ ] `GRPC_ENABLED=true` e `GRPC_SERVER_ADDR` configurado
- [ ] Binario do agent gerado

**No painel:**

- [ ] Servidor cadastrado (Nome, Host, Porta, User, Chave ou Senha)
- [ ] ACME Email preenchido (para TLS automatico)
- [ ] Botao **Provision** clicado
- [ ] Status mudou para **Online**

---

## Estrutura de arquivos no servidor remoto

Apos provisionamento completo:

```
~/paasdeploy-agent/
  agent          # binario do agent
  ca.pem         # certificado CA
  cert.pem       # certificado do agent (mTLS)
  key.pem        # chave privada do agent

~/.config/systemd/user/
  paasdeploy-agent.service  # servico systemd

/opt/traefik/              # (se ACME Email configurado)
  traefik.yml              # configuracao do Traefik
  letsencrypt/
    acme.json              # certificados TLS (gerado automaticamente)
```

Containers Docker criados:

```
traefik    # reverse proxy (portas 80, 443, 50051, 8081)
```

Rede Docker:

```
paasdeploy  # rede compartilhada entre Traefik e apps
```

# Tutorial: Configurar Servidor Remoto para Deploy

Passo a passo para preparar um servidor (VPS) e provisionar o agent do FlowDeploy automaticamente.

---

## Visão geral

1. Preparar o servidor remoto (VPS)
2. Configurar o backend do FlowDeploy
3. Cadastrar o servidor no painel
4. Provisionar e verificar

---

## Parte 1: Preparar o servidor remoto (VPS)

Execute estes comandos **no servidor remoto** (via SSH com usuário root ou com sudo).

### 1.1 Criar usuário para o agent (se ainda não existir)

```bash
sudo adduser deploy
```

Se preferir usar um usuário existente (ex.: `oab-api`), pule para o passo 1.2.

### 1.2 Habilitar linger para o usuário

O linger permite que serviços `systemd --user` continuem rodando após logout. **Obrigatório** para o agent.

```bash
sudo loginctl enable-linger <USUARIO>
```

Exemplo para usuário `oab-api`:

```bash
sudo loginctl enable-linger oab-api
```

### 1.3 Verificar acesso SSH

O usuário precisa conseguir fazer login via SSH com **chave privada** ou **senha**.

**Opção A – Chave privada:**

- Adicione a chave pública em `~/.ssh/authorized_keys` do usuário.
- Teste: `ssh deploy@IP_DO_SERVIDOR -p PORTA` (deve conectar sem pedir senha).

**Opção B – Senha:**

- Certifique-se de que o usuário tem senha definida.
- O FlowDeploy aceita senha; ela será armazenada criptografada.

### 1.4 Confirmar systemd user

Entre no servidor com o usuário escolhido e teste:

```bash
ssh deploy@IP_DO_SERVIDOR -p PORTA
systemctl --user status
```

Se aparecer "Failed to connect to bus", o linger não está habilitado. Volte ao passo 1.2.

---

## Parte 2: Configurar o backend do FlowDeploy

Execute no **host onde o backend está rodando**.

### 2.1 Gerar chave de criptografia

```bash
openssl rand -base64 32
```

Use o resultado em `TOKEN_ENCRYPTION_KEY` no `.env`.

### 2.2 Gerar binário do agent

**Desenvolvimento local:**

```bash
cd apps/agent
go build -o ../../dist/agent ./cmd/agent
```

**Produção (Docker):** A imagem do backend já inclui o agent em `/app/agent`. Configure `AGENT_BINARY_PATH=/app/agent` no container.

**Build manual do agent (standalone):**

```bash
docker build -f apps/agent/Dockerfile -t paasdeploy-agent .
docker run --rm paasdeploy-agent cat /agent > dist/agent
```

### 2.3 Configurar variáveis no `.env`

| Variável             | Valor exemplo                            |
| -------------------- | ---------------------------------------- |
| TOKEN_ENCRYPTION_KEY | (saída do `openssl rand -base64 32`)     |
| GRPC_ENABLED         | true                                     |
| GRPC_PORT            | 50051                                    |
| GRPC_SERVER_ADDR     | host:50051 (endereço acessível pela VPS) |
| AGENT_BINARY_PATH    | /caminho/absoluto/para/dist/agent        |
| AGENT_GRPC_PORT      | 50052                                    |

### 2.4 Reiniciar o backend

Após alterar o `.env`, reinicie o backend para carregar as novas variáveis.

---

## Parte 3: Cadastrar o servidor no painel

### 3.1 Acessar o FlowDeploy

Abra o painel (ex.: `http://localhost:3000`), faça login e vá em **Servers** > **Add Server**.

### 3.2 Preencher o formulário

| Campo        | Descrição                                       |
| ------------ | ----------------------------------------------- |
| Name         | Nome amigável (ex.: production, staging)        |
| Host         | IP ou hostname da VPS                           |
| SSH Port     | Porta SSH (padrão 22)                           |
| SSH User     | Usuário do passo 1.1/1.2 (ex.: deploy, oab-api) |
| SSH Key      | Chave privada completa (opcional se usar senha) |
| SSH Password | Senha do usuário (opcional se usar chave)       |

É necessário informar **chave** ou **senha**; os dois podem ser usados juntos.

### 3.3 Salvar

Após clicar em **Add Server**, o servidor aparece com status **Pending**.

---

## Parte 4: Provisionar e verificar

### 4.1 Provisionar

No card do servidor, clique em **Provision**.

O backend irá:

1. Conectar via SSH na VPS
2. Criar `~/paasdeploy-agent/` e copiar certs + binário
3. Criar serviço systemd em `~/.config/systemd/user/paasdeploy-agent.service`
4. Iniciar o agent

O status muda para **Online** quando o agent se registrar e enviar heartbeat.

### 4.2 Acompanhar logs (backend)

No terminal onde o backend está rodando, aparecerá:

- `provision completed ...` em caso de sucesso
- Mensagem de erro em caso de falha

### 4.3 Verificar no servidor remoto

Entre na VPS e confira:

```bash
# Conteúdo da pasta do agent
ls ~/paasdeploy-agent

# Status do serviço
systemctl --user status paasdeploy-agent

# Logs em tempo real
journalctl --user -u paasdeploy-agent -f
```

### 4.4 Testar conectividade

No painel ou via API:

```
GET /paas-deploy/v1/servers/:id/health
```

Resposta esperada: `{ "status": "ok", "latencyMs": ... }`.

---

## Resolução de problemas

| Problema                        | Possível causa                    | Solução                                          |
| ------------------------------- | --------------------------------- | ------------------------------------------------ |
| `ssh connect: i/o timeout`      | Firewall bloqueando porta SSH     | Liberar porta SSH (22 ou a configurada)          |
| `unable to authenticate`        | Chave/senha incorretos            | Conferir credenciais; testar `ssh` manualmente   |
| `provision failed: EOF`         | Binário grande / comando truncado | Atualizar backend (usa SFTP) e reprovisionar     |
| `Unit ... could not be found`   | Linger não habilitado             | Executar `sudo loginctl enable-linger <usuario>` |
| `Failed to connect to bus`      | Linger não habilitado             | Idem acima                                       |
| Status permanece "Provisioning" | Processo travado ou em andamento  | Ver logs do backend; conferir se SSH não caiu    |

---

## Resumo rápido (checklist)

**No servidor remoto (uma vez):**

- [ ] Usuário criado ou existente
- [ ] `sudo loginctl enable-linger <usuario>` executado
- [ ] SSH funcionando (chave ou senha)
- [ ] `systemctl --user status` funciona sem erro

**No backend:**

- [ ] `TOKEN_ENCRYPTION_KEY` definida
- [ ] `AGENT_BINARY_PATH` apontando para `dist/agent`
- [ ] `GRPC_ENABLED=true` e `GRPC_SERVER_ADDR` configurado
- [ ] Binário do agent gerado (`go build -o dist/agent ...`)

**No painel:**

- [ ] Servidor cadastrado (Nome, Host, Porta, User, Chave ou Senha)
- [ ] Botão **Provision** clicado
- [ ] Status mudou para **Online**

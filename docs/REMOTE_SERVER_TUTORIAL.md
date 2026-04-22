# Tutorial: Cadastrar e Provisionar um Servidor Remoto

Passo a passo para preparar um VPS, cadastra-lo no FlowDeploy e provisionar
o agente automaticamente. Para a arquitetura por tras do que acontece em cada
etapa, consulte [`REMOTE_SSH_DEPLOY.md`](./REMOTE_SSH_DEPLOY.md).

## 1. Visao geral

Voce vai:

1. Preparar o VPS de destino (sistema operacional, rede, usuario).
2. Garantir que o backend do FlowDeploy esta pronto para emitir certificados
   e expor a porta gRPC.
3. Cadastrar o servidor no painel.
4. Disparar o **Provision** e acompanhar os logs.
5. Validar que o agente esta `online` e usar a maquina para deploys.

> Tempo estimado: 5 a 10 minutos por servidor.

## 2. Pre-requisitos no VPS

Sistema operacional homologado: **Ubuntu Server 22.04+** (qualquer Linux com
`systemd` e Docker disponiveis tambem funciona).

| Item                                   | Por que                                                        |
| -------------------------------------- | -------------------------------------------------------------- |
| Acesso SSH (chave **ou** senha)        | provisionamento usa SSH                                         |
| Usuario com `sudo` (ou root)           | instalar Docker e criar a unit `systemd`                        |
| Conexao a internet                     | baixar Docker, Traefik e o binario do agente                    |
| Porta TCP 22 aberta                    | SSH durante o provisionamento                                   |
| Portas TCP 80/443 abertas              | Traefik atender HTTP/HTTPS dos apps                             |
| Porta TCP 50052 (ou customizada)       | porta gRPC do agente, **so o backend precisa alcancar**         |
| Hostname publico ou IP fixo            | aparecerá no painel e nos certificados                          |

> O backend nao precisa de IP fixo, mas precisa estar acessivel pelo VPS
> via HTTPS (para `Register`/`Heartbeat`) e via gRPC (para deploy).

## 3. Pre-requisitos no backend

Antes do primeiro provisionamento, garanta que:

1. As variaveis abaixo estao no `.env` do backend:
   - `GRPC_PORT` (default `50051`) — porta interna do gRPC do backend.
   - `GRPC_AGENT_PORT` (default `50052`) — porta exposta para os agentes.
   - `BACKEND_PUBLIC_HOST` (ou equivalente) — hostname publico do backend.
2. Traefik esta com **TCP route SNI** publicando `GRPC_AGENT_PORT` para o
   mesmo hostname (ver `deploy/traefik/`).
3. A migracao `000014_pki_ca` ja foi aplicada (gera a CA interna
   automaticamente no startup).
4. O binario do agente correspondente ao `AGENT_VERSION` esta buildado e
   acessivel pelo `agentdownload`.

> Em ambiente de desenvolvimento, voce pode rodar o agente apontando direto
> para `localhost:50051` sem Traefik, conforme o `README.md`.

## 4. Cadastrar o servidor no painel

1. Logue no painel como `admin`.
2. Acesse **Servers → New Server**.
3. Preencha:
   - **Name**: nome amigavel (ex: `prod-eu-1`).
   - **Host**: IP ou hostname publico do VPS.
   - **SSH Port**: `22` (default).
   - **SSH User**: usuario com `sudo` ou `root`.
   - **Authentication**: cole a chave privada (recomendado) **ou** informe a
     senha temporaria.
   - **ACME Email**: email para o Let's Encrypt nesse VPS (opcional, mas
     necessario para SSL automatico nos apps).
   - **Agent Update Mode**: `auto` (recomendado) ou `manual`.
4. Salve. O servidor aparece com status `pending`.

## 5. Disparar o provisionamento

1. Na lista de servidores, clique em **Provision**.
2. Uma janela com **logs em tempo real** abre. Voce vera as etapas:

```
[01/10] Conectando via SSH a 203.0.113.10
[02/10] Detectando privilegios e configurando ambiente remoto
[03/10] Instalando Docker Engine e Compose v2
[04/10] Criando rede Docker paasdeploy_net
[05/10] Instalando Traefik (acme_email configurado)
[06/10] Gerando certificado mTLS do agente
[07/10] Enviando binario do agente (paasdeploy-agent vX.Y.Z)
[08/10] Criando unidade systemd paasdeploy-agent.service
[09/10] Iniciando servico e aguardando Register
[10/10] Servidor online
```

3. Em caso de erro, a etapa fica vermelha e o painel mostra a saida
   completa do comando que falhou. Os logs tambem ficam em
   `apps/backend` (com `slog` correlacionado por `serverId`).

> O fluxo e idempotente: rodar **Provision** novamente em um servidor ja
> provisionado nao reinstala o que ja existe; apenas garante que tudo esta
> em estado consistente.

## 6. Validacao pos-provisionamento

Depois do status `online`, valide rapidamente:

```bash
ssh <user>@<host>
sudo systemctl status paasdeploy-agent     # active (running)
sudo journalctl -u paasdeploy-agent -n 50  # logs recentes
sudo docker ps                              # paasdeploy-agent + traefik (se ACME)
sudo docker network ls | grep paasdeploy_net
```

E no painel:

- O cartao do servidor mostra metricas de CPU/RAM/disco recebidas via SSE.
- A aba **Containers** lista os containers existentes (deve aparecer
  `paasdeploy-agent` e o `traefik` se foi instalado).
- A aba **Certificates** lista os certificados do Traefik daquele VPS.

## 7. Cadastrar e fazer deploy de um app neste servidor

1. **Apps → New App**.
2. Preencha:
   - **Repository URL**: `https://github.com/org/repo.git` (ou via GitHub App).
   - **Branch**: `main` (default).
   - **Workdir**: caminho relativo no monorepo, se aplicavel.
   - **Server**: selecione o VPS recem-provisionado.
   - **Domain**: dominio que sera atendido pelo Traefik desse VPS.
3. Confirme. O backend cria webhook no GitHub (via GitHub App, se conectado),
   enfileira o primeiro deploy e exibe os logs em tempo real.

## 8. Operacoes do dia a dia

### Re-deploy manual

Botao **Redeploy** (ou `POST /api/apps/:id/redeploy`). Usa o ultimo SHA
conhecido da branch.

### Rollback

Botao **Rollback** (ou `POST /api/apps/:id/rollback`). Volta para a versao
anterior **e** revalida o health check antes de dar sucesso.

### Terminal web

Aba **Terminal** dentro de um container. Usa `ExecContainer` (gRPC
bidirecional) com PTY de verdade — voce pode rodar `htop`, `vim`, etc.

### Atualizar o agente

- **Auto** (recomendado): backend faz `PushUpdate` ao detectar versao mais
  nova via heartbeat.
- **Manual**: na pagina do servidor, clique em **Push Update** apos buildar
  uma nova versao do binario.

### Limpeza programada

A scheduler de limpeza roda dentro do agente (`apps/agent/internal/cleanup`).
Voce pode disparar uma execucao agora pela UI ou via
`POST /api/cleanup/run?serverId=...`.

## 9. Solucao de problemas

| Sintoma                                         | O que checar                                                               |
| ----------------------------------------------- | -------------------------------------------------------------------------- |
| Provisionamento para em **Conectando via SSH**  | porta 22 aberta? chave correta? `sudo` exige senha?                         |
| Erro **Docker install failed**                  | distro suportada? Internet liberada? `apt`/`dnf` disponivel?                |
| Servidor fica em **registering**                | porta gRPC do backend acessivel pelo VPS? CA presente em `pki_ca`?          |
| Agente reinicia em loop                         | `journalctl -u paasdeploy-agent` mostra o erro real (ex: porta 50052 ocupada) |
| Deploy demora a sair de **pending**             | `DEPLOY_WORKERS` zerado? dispatcher logando algo?                           |
| Healthcheck sempre falha                        | `paasdeploy.json` com `port` e `healthcheck.path` corretos?                 |
| TLS dos apps nao funciona                       | `acme_email` configurado no servidor? portas 80/443 abertas?                |

## 10. Desprovisionar um servidor

1. **Servers → <servidor> → Delete**. Voce escolhe se quer:
   - **Apenas remover do painel** (deixa o agente rodando, util para
     migracao manual).
   - **Remover completamente** (executa rotina remota: para o servico, remove
     unit `systemd`, apaga binario, remove certificados).
2. Apps que estavam atrelados ao servidor ficam orfaos e devem ser
   reatribuidos antes da exclusao.

## 11. Boas praticas

- Use **um servidor por ambiente logico** (ex: `prod`, `staging`) e nao
  misture cargas concorrentes em um unico VPS pequeno.
- Configure `acme_email` em todos os servidores publicos para SSL automatico.
- Monitore o cartao do servidor regularmente; metricas em vermelho aparecem
  via SSE em segundos.
- Mantenha o `AGENT_VERSION` atualizado no repositorio para que o auto-update
  mantenha a frota homogenea.
- Faca rotacao periodica dos certificados mTLS (script previsto no roadmap).

## 12. Comandos uteis no VPS

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

> Em caso de problema persistente, anexe a saida de `journalctl -u
> paasdeploy-agent --no-pager -n 200` ao abrir uma issue.

# FlowDeploy — Mapa de Features e Roadmap

Este documento mostra **o que ja existe na plataforma hoje** e **o que esta
planejado para os proximos ciclos**. Use como fonte unica para responder a
pergunta "o FlowDeploy ja faz X?".

> Convencoes:
>
> - ✅ implementado e disponivel na branch principal
> - 🚧 parcial / experimental (existe codigo mas ainda nao foi promovido)
> - 📋 planejado (existe um roadmap claro mas nao foi iniciado)
> - ❌ fora de escopo no momento

## 1. Comparativo com plataformas similares

| Feature                                       | Coolify | FlowDeploy | Status                                    |
| --------------------------------------------- | ------- | ---------- | ----------------------------------------- |
| Deploy de aplicacoes Docker                   | ✅      | ✅         | `engine` + `shared/pkg/docker`            |
| Push to deploy via Git                        | ✅      | ✅         | webhooks GitHub + dedup (`000028`)        |
| Integracao GitHub (OAuth + App)               | ✅      | ✅         | `internal/github`, `internal/ghclient`    |
| Multi-servidor remoto                         | ✅      | ✅         | agente gRPC + mTLS (`internal/pki`)       |
| SSL automatico (Let's Encrypt + Traefik)      | ✅      | ✅         | Traefik 3.x + ACME                        |
| Variaveis de ambiente criptografadas          | ✅      | ✅         | `internal/crypto`                         |
| Logs em tempo real                            | ✅      | ✅         | SSE em `/events/deploys`                  |
| Monitoramento (CPU, RAM, disco, rede)         | ✅      | ✅         | `system_stats_monitor` + `server_stats`   |
| Health checks com retry e rollback            | ✅      | ✅         | configuravel via `paasdeploy.json`        |
| Rollback                                      | ✅      | ✅         | rota dedicada `/api/apps/:id/rollback`    |
| DNS automatico (Cloudflare)                   | ❌      | ✅         | OAuth Cloudflare + `internal/cloudflare`  |
| Notificacoes (Slack / Discord / Email)        | ✅      | ✅         | `internal/notification`                   |
| Templates one-click (Postgres, Redis, Nginx…) | ✅      | ✅         | `template_handler` + `template_data`      |
| Auditoria de eventos                          | ✅      | ✅         | `internal/service/audit_service.go`       |
| Auto-update do agente                         | parcial | ✅         | `agentdownload` + `handlers_update`       |
| Provisionamento automatico via SSH            | ✅      | ✅         | `internal/provisioner`                    |
| Terminal web (docker exec interativo)         | ✅      | ✅         | gRPC bidirecional + `creack/pty`          |
| Backups automaticos de banco                  | ✅      | 📋         | roadmap (Q3)                              |
| Runtime K3s (rolling, replicas, ingress)      | parcial | 📋         | ver `docs/K3S_DEPLOY_ROADMAP.md`          |
| Marketplace de templates                      | parcial | 📋         | catalogo atual e estatico                 |
| Multi-tenant com cobranca                     | ❌      | ❌         | fora de escopo                            |

## 2. Features implementadas

### 2.1 Pipeline de deploy

- ✅ Fila persistida em PostgreSQL com `SELECT ... FOR UPDATE SKIP LOCKED`.
- ✅ Pool de workers configuravel (`DEPLOY_WORKERS`).
- ✅ Suporte a Dockerfile e a `docker-compose.yml` por app.
- ✅ Suporte a monorepos via campo `workdir` em `paasdeploy.json`.
- ✅ Health check apos build, com retries e timeout configuraveis.
- ✅ Rollback automatico em caso de falha de health.
- ✅ Rollback manual por endpoint REST.
- ✅ Deduplicacao de deploys (`migration 000028_deploy_dedup`).
- ✅ Logs em tempo real via SSE, sem polling.

### 2.2 Multi-servidor

- ✅ Agente Go leve por VPS, comunicando via gRPC.
- ✅ mTLS com CA interna (`internal/pki`) e certificados por agente.
- ✅ Traefik com TCP route SNI para expor a porta gRPC do backend.
- ✅ Provisionamento automatico via SSH (chaves geradas pelo backend).
- ✅ Auto-update do binario do agente: backend serve binario, agente baixa,
  valida hash e reinicia.
- ✅ Heartbeat com metricas e versao reportada.
- ✅ `RequireAdminForLocal` impede usuarios comuns de fazer deploy no host.

### 2.3 Integracoes

- ✅ GitHub OAuth para login e GitHub App para clone de repositorios privados.
- ✅ Webhooks GitHub com verificacao HMAC-SHA256 em tempo constante.
- ✅ Cloudflare via OAuth: cria CNAMEs automaticamente em zonas do usuario.
- ✅ Slack, Discord e Email (SMTP) como canais de notificacao com regras
  configuraveis por app/evento.

### 2.4 Operacao e seguranca

- ✅ Login com GitHub OAuth ou email/senha (bcrypt).
- ✅ Sessoes opacas em cookies HttpOnly + Secure + SameSite=Lax.
- ✅ Roles `admin` e `user`.
- ✅ Variaveis de ambiente criptografadas no banco.
- ✅ Tokens GitHub e Cloudflare criptografados antes de persistir.
- ✅ Auditoria de eventos com payloads de webhook armazenados para replay.
- ✅ Migracoes embutidas via `golang-migrate` (28+ migracoes).

### 2.5 Frontend

- ✅ React 18 + Vite 6 + Tailwind 3.4 + shadcn/ui.
- ✅ Mobile-first em todas as telas.
- ✅ TanStack Query 5 + SSE para zero polling.
- ✅ Tema light/dark com persistencia.
- ✅ Editor de variaveis de ambiente com mascara para valores sensiveis.

## 3. Em desenvolvimento (🚧)

| Item                                                | Status                                                   |
| --------------------------------------------------- | -------------------------------------------------------- |
| Dashboard de auditoria com filtros avancados        | UI parcial, falta paginacao e exportacao                 |
| Alertas inteligentes (anomaly detection em metrica) | prototipo so no backend, sem regra ainda                 |
| Multi-region failover (active-passive)              | discussao tecnica iniciada                               |

## 4. Roadmap (📋)

### Q2

- 📋 **Backups automaticos de banco** — agendamento de `pg_dump` e
  envio para storage S3-compativel; ja existe modelo de `cleanup_logs`
  como inspiracao.
- 📋 **Templates dinamicos** — permitir que o usuario cadastre seus
  proprios templates (compose + manifesto) sem alterar codigo.
- 📋 **Webhooks de saida configuraveis** — disparar HTTP arbitrario para
  eventos de deploy, similar ao Slack/Discord mas generico.

### Q3

- 📋 **Runtime K3s opcional** — adicionar K3s como alternativa ao Docker em
  servidores remotos, com rolling updates, replicas e ingress automatico.
  Ver [`K3S_DEPLOY_ROADMAP.md`](./K3S_DEPLOY_ROADMAP.md) para detalhes.
- 📋 **Pipelines compostos** — declarar etapas extras (lint, testes, smoke
  tests) que rodam dentro do worker antes do `docker build`.
- 📋 **Marketplace de templates** — catalogo curado, com versionamento e
  preview de variaveis.

### Q4

- 📋 **Suporte a GitLab e Bitbucket** — abstrair `ghclient` como `gitclient`
  e adicionar implementacoes alternativas; o `webhook.Manager` ja esta
  desenhado para isso.
- 📋 **Painel multi-cluster** — agrupar servidores por ambiente (dev /
  staging / prod) e exibir deploys agregados.
- 📋 **CLI oficial** — comando `flowdeploy` para listar apps, abrir terminal
  remoto e disparar deploys via gRPC.

## 5. Itens explicitamente fora de escopo

- ❌ Multi-tenant SaaS com cobranca embutida.
- ❌ Construir nosso proprio orquestrador de containers (continuamos
  apostando em Docker e, no futuro, K3s).
- ❌ Substituir Traefik por outro proxy.
- ❌ Suporte a Kubernetes generico (somente K3s opt-in para a feature
  planejada).

## 6. Como propor novas features

1. Abra uma issue descrevendo o problema antes da solucao.
2. Aponte qual area do `.cursor/rules/` seria afetada.
3. Liste alternativas e impactos em seguranca, performance e UX.
4. Para mudancas grandes, escreva uma RFC em `docs/` (ver
   `.cursor/skills/create-rfc/` para template).
5. Valide o roadmap antes de implementar — features fora do escopo
   declarado devem virar RFC primeiro.

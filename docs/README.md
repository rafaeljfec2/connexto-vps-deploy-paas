# FlowDeploy Documentation

Welcome to the FlowDeploy documentation hub. This folder is the single source
of truth for **architecture, integrations and operational guides**. The
project root keeps higher-level entry points:

- [`README.md`](../README.md) — project overview, features and quick start
- [`AGENTS.md`](../AGENTS.md) — onboarding for AI agents and new humans
- [`CHANGELOG.md`](../CHANGELOG.md) — release notes
- [`.cursor/rules/`](../.cursor/rules/) — coding conventions enforced in review

## Index

| Document                                                       | Audience              | Language | What it covers                                                                          |
| -------------------------------------------------------------- | --------------------- | -------- | --------------------------------------------------------------------------------------- |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md)                         | engineers             | EN       | Runtime architecture: backend, deploy engine, agent, frontend, real-time events         |
| [`CONTRIBUTING.md`](./CONTRIBUTING.md)                         | engineers             | EN       | Local setup, daily commands, workflow, coding standards, testing, release process       |
| [`GITHUB_INTEGRATION.md`](./GITHUB_INTEGRATION.md)             | engineers             | EN       | OAuth sign-in, GitHub App, webhook architecture, signature verification                 |
| [`AUTO_DNS_CONFIGURATION.md`](./AUTO_DNS_CONFIGURATION.md)     | engineers + operators | PT-BR    | Cloudflare OAuth, automatic CNAME creation, custom domains lifecycle                    |
| [`REMOTE_SSH_DEPLOY.md`](./REMOTE_SSH_DEPLOY.md)               | engineers             | PT-BR    | Remote agent architecture (gRPC + mTLS), provisioning, deploy lifecycle, auto-update    |
| [`REMOTE_SERVER_TUTORIAL.md`](./REMOTE_SERVER_TUTORIAL.md)     | operators             | PT-BR    | Step-by-step tutorial: prepare a VPS, register it, provision the agent, deploy an app   |
| [`SECURITY.md`](./SECURITY.md)                                 | engineers + operators | EN       | Threat model, trust boundaries, mTLS, secrets handling, hardening checklist, reporting  |
| [`FEATURES_ROADMAP.md`](./FEATURES_ROADMAP.md)                 | product + engineers   | PT-BR    | What is implemented today vs. roadmap, comparison with similar platforms                |
| [`K3S_DEPLOY_ROADMAP.md`](./K3S_DEPLOY_ROADMAP.md)             | engineers             | EN       | Planned K3s runtime as an opt-in alternative to Docker                                  |

## How to contribute to the docs

1. Match the language already used in the file you're editing
   (English or PT-BR; the project intentionally mixes both).
2. Keep diagrams and code references in sync with the source — when
   architecture changes, update the doc in the same PR.
3. Reference files using relative paths from the document
   (e.g. `apps/backend/internal/engine/worker.go`) so that links keep working
   when files move.
4. For new architectural decisions, prefer adding an ADR or RFC in the
   appropriate skill folder before writing a long doc here.
5. Avoid creating new top-level documents unless the topic does not fit any
   existing file. Prefer extending an existing one to reduce fragmentation.

## Conventions

- Files use UPPER_SNAKE_CASE for historical reasons; keep the convention
  for any new doc.
- Each document opens with a short summary and a link list to related docs.
- Use Mermaid diagrams when sequencing or flow needs to be explicit; use
  ASCII boxes for high-level component diagrams (no rendering dependency).
- Versions and feature flags must reference reality. Run `git grep` against
  the source before publishing a change.

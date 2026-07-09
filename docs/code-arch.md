# Fern Server — Code & Architecture

Developer guide for running, configuring, and navigating this repo. For product context see [README](../README.md). For roadmap and API contracts see [PLAN.md](PLAN.md) and [MVP.md](MVP.md).

## Stack

- **Go 1.25** — chi router, WebSocket streaming, JWT auth
- **Postgres 17 + pgvector** — relational data now; embeddings/RAG reserved for later
- **LLM providers** — OpenAI or Anthropic via `internal/llm.Provider`

## Getting started

```bash
# 1. Start Postgres
docker compose up -d

# 2. Configure
cp .env.example .env
# Set JWT_SECRET and at least one of OPENAI_API_KEY / ANTHROPIC_API_KEY

# 3. Run (migrations apply on boot)
go run ./cmd/fern
```

Server listens on `:8080` by default. Health check: `GET /health`.

### Environment variables

| Variable | Required | Notes |
|----------|----------|-------|
| `JWT_SECRET` | yes | Signs access tokens |
| `DATABASE_URL` | no | Defaults to local docker-compose Postgres |
| `LLM_PROVIDER` | no | `openai` or `anthropic` (default: `openai`) |
| `LLM_MODEL` | no | Provider model id (default: `gpt-4o-mini`) |
| `OPENAI_API_KEY` | one of | Required when `LLM_PROVIDER=openai` |
| `ANTHROPIC_API_KEY` | one of | Required when `LLM_PROVIDER=anthropic` |
| `ENCRYPTION_KEY` | prod | Base64-encoded 32-byte key; omit for plaintext dev storage |
| `ALLOWED_ORIGINS` | no | CORS + WebSocket origins (comma-separated) |
| `PORT` | no | Default `8080` |

### Manual WebSocket testing

```bash
go run ./cmd/wstest
```

## Repository layout

```
cmd/
  fern/              Entrypoint — config, migrations, router, server
  wstest/            Dev helper for WebSocket chat
internal/
  api/               REST handlers, router, rate limiting
  auth/              JWT, argon2 passwords, refresh tokens, middleware
  ws/                WebSocket chat — streaming, wrap-up, memory injection
  llm/               Provider interface (anthropic.go, openai.go)
  prompts/           Persona, modalities, summary/report builders
  safety/            Crisis keyword detection
  crypto/            AES-256-GCM transcript encryption at rest
  store/             Postgres access (pgx)
  config/            Env loading
migrations/          SQL migrations (applied on startup)
docs/                Product and engineering docs
```

## API surface

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
GET    /api/v1/chat/ws              WebSocket — session streaming + wrap-up
GET    /api/v1/journal              List past sessions
GET    /api/v1/journal/{id}         Session detail (transcript + summary)
GET    /api/v1/mood/trends
GET    /api/v1/recovery
PUT    /api/v1/recovery/profile
DELETE /api/v1/recovery/profile
POST   /api/v1/recovery/craving
POST   /api/v1/reports/weekly       ?force=true to regenerate
GET    /api/v1/reports
GET    /api/v1/me
PATCH  /api/v1/me
GET    /api/v1/me/export
DELETE /api/v1/me
```

Full WS protocol and safety notes: [PLAN.md](PLAN.md#api-surface).

## Key design points

- **Session lifecycle** — Messages stream over WS; `wrap_up` triggers summary generation and persistence. Abandoned sessions are not saved.
- **Memory v1** — Last 8 session summaries injected into the system prompt (`memoryDepth` in `internal/ws/chat.go`).
- **Modality prompts** — CBT, ACT, DBT, MI, IFS, and freeform live in `internal/prompts/modalities.go`.
- **Encryption** — Transcripts encrypted at rest when `ENCRYPTION_KEY` is set; legacy plaintext rows still readable.

## Related repos

- Desktop client: [FernDesktop](https://github.com/DiableroTech/FernDesktop)

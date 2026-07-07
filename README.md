# Fern Server

Go backend for **Fern** — a privacy-first AI journaling companion trained in psychology, psychotherapy, and addictions counselling.

Fern pairs a conversational journal with therapeutic frameworks (CBT, ACT, DBT, Motivational Interviewing, IFS), mood tracking, and a dedicated recovery module. The AI runs on hosted LLMs (OpenAI / Anthropic) behind a provider abstraction; everything else — accounts, journal history, analysis — lives here.

The desktop client lives in [FernDesktop](../FernDesktop).

## Stack

- **Go 1.25** — chi router, WebSocket streaming, JWT auth
- **Postgres 17 + pgvector** — relational data now, embeddings/RAG later
- **LLM providers** — OpenAI (`gpt-4o-mini` default) and Anthropic, swappable via config

## Getting started

```bash
# 1. Start Postgres
docker compose up -d

# 2. Configure
cp .env.example .env
# set JWT_SECRET and at least one of OPENAI_API_KEY / ANTHROPIC_API_KEY

# 3. Run
go run ./cmd/fern
```

Server listens on `:8080` by default. Health check: `GET /health`.

## Layout

```
cmd/fern/          entrypoint
internal/
  api/             REST handlers + router
  auth/            JWT, argon2 password hashing, middleware
  ws/              WebSocket hub, token streaming
  llm/             Provider interface (anthropic.go, openai.go)
  prompts/         Therapeutic persona + modality system prompts
  safety/          Crisis detection pipeline
  journal/         Entry CRUD, encryption at rest
  mood/            Mood tracking
  recovery/        Addictions module: cravings, triggers, streaks
  store/           Postgres access (pgx)
  config/          Env config
migrations/        SQL migrations
docs/              Project plan and design docs
```

See [docs/PLAN.md](docs/PLAN.md) for the full roadmap.

## Safety note

Fern is a self-help journaling tool, **not therapy** and not a substitute for professional mental health care. Crisis detection surfaces helpline information (988 in US/Canada) and the AI is instructed never to provide harmful content.

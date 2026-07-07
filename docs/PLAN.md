# Fern — Project Plan

Fern is a self-hostable, privacy-first AI journaling companion inspired by Rosebud, with a sharper focus on therapeutic modalities and addictions recovery. Multi-user from day one.

## Product pillars

1. **Conversational journaling** — chat-style entries with adaptive follow-up questions, streamed token-by-token over WebSocket.
2. **Modality selector** — each session runs under a therapeutic framework: CBT, ACT, DBT, MI (Motivational Interviewing), IFS, or freeform. Each maps to a distinct system prompt (`internal/prompts/`).
3. **Recovery module** — craving log, trigger mapping, sobriety streaks/milestones, MI-driven check-ins. This is the differentiator vs. Rosebud.
4. **Crisis safety** — keyword + model-flagged risk detection that breaks the conversational pattern and surfaces helplines.
5. **Memory & insight (v2)** — RAG over journal history ("Ask Fern"), weekly AI reports, cross-entry pattern recognition, therapist-export PDF.

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────────┐
│ FernDesktop │────▶│   FernServer     │────▶│  OpenAI API      │
│ (Electron)  │◀────│   (Go)           │◀────│  or Claude API   │
└─────────────┘     │  auth / journal  │     └──────────────────┘
                    │  ws streaming    │     ┌──────────────────┐
                    │  prompts/safety  │────▶│  Postgres 17     │
                    │  recovery        │◀────│  + pgvector      │
                    └──────────────────┘     └──────────────────┘
```

**Key decisions:**
- **Hosted LLMs, not self-hosted.** GPT-4o mini is ~$0.01/session; quality far exceeds any self-hostable 8B model. Provider interface (`internal/llm.Provider`) keeps OpenAI and Anthropic swappable via `LLM_PROVIDER` env.
- **System prompts, not fine-tuning (for now).** A detailed therapeutic persona + per-modality instructions gets 80% of the value. Fine-tuning is a Phase 5 project once real usage data exists.
- **Postgres + pgvector** — one database for relational data and future embeddings.
- **JWT auth** — access + refresh tokens, argon2id password hashing.
- **Journal encryption at rest** — entry transcripts stored as encrypted BYTEA.
- **Deployment target** — any cheap VPS (server + Postgres in Docker, Caddy for TLS). No GPU needed.

## Roadmap

| Phase | Scope | Milestone |
|-------|-------|-----------|
| **0** | Repos, scaffolding, deps, docker-compose, prompt drafts | "Projects exist" ✅ |
| **1** | Auth (register/login/refresh), WS hub, streaming chat against provider | "It's alive" |
| **2** | Journal persistence + encryption, timeline, mood capture, modality selection end-to-end | "It's a journal" |
| **3** | Recovery module (cravings/triggers/streaks), crisis detection pipeline | "It's therapeutic" |
| **4** | VPS deploy, Caddy + TLS, domain | "It's live" |
| **5** | pgvector embeddings, Ask Fern (RAG), weekly reports, therapist export, fine-tuning | "It's smart" |

## API surface (planned)

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
GET    /api/v1/chat/ws            WebSocket: streaming sessions
GET    /api/v1/journal            list entries (paginated)
GET    /api/v1/journal/{id}
DELETE /api/v1/journal/{id}
GET    /api/v1/mood/trends
POST   /api/v1/recovery/craving
POST   /api/v1/recovery/trigger
GET    /api/v1/recovery/streaks
```

## Safety posture

- Fern never claims to be a therapist; system prompt enforces boundaries.
- Crisis detection: server-side keyword screen + model self-flagging; client shows helpline banner.
- No data sold, no third-party analytics on journal content. API providers receive conversation text (documented clearly to users); entries at rest are encrypted.

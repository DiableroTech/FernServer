# Fern — Project Plan

Fern is a self-hostable, privacy-first AI journaling companion inspired by Rosebud, with a sharper focus on therapeutic modalities and addictions recovery. Multi-user from day one.

> **See also:** [MVP.md](MVP.md) — the working blueprint: full architecture walkthrough, data/privacy position, MVP scope, milestone-by-milestone build order, and feature specs. This doc stays high-level; MVP.md is where the detail lives.

## Product pillars

1. **Session-based journaling** — journaling happens in discrete *sessions*, not one endless chat. A session is a guided conversation: the person writes, Fern responds (streamed over WebSocket), and when they're done they **wrap up** — Fern generates a summary, insights, and a mood label, and the session is saved.
2. **Memory** — Fern remembers. Recent session summaries are injected into the system prompt so every new session opens with context, like a therapist who read their notes. Full RAG over transcripts ("Ask Fern") comes in Phase 5.
3. **Modality selector** — each session runs under a therapeutic framework: CBT, ACT, DBT, MI (Motivational Interviewing), IFS, or freeform. Locked per-session once it starts.
4. **Recovery module** — craving log, trigger mapping, sobriety streaks/milestones, MI-driven check-ins. This is the differentiator vs. Rosebud.
5. **Insights & reports** — a summary card after every session (done) and a Rosebud-style weekly report that synthesizes the last 7 days of sessions, moods, and recovery events into themes, wins, and a gentle challenge.
6. **Crisis safety** — keyword + model-flagged risk detection that breaks the conversational pattern and surfaces helplines.
7. **Freeform notepad** — a classic open journal page (no AI), for people who just want to write. Planned, not yet scheduled.

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────────┐
│ FernDesktop │────▶│   FernServer     │────▶│  Claude API      │
│ (Electron)  │◀────│   (Go)           │◀────│  or OpenAI API   │
└─────────────┘     │  auth / sessions │     └──────────────────┘
                    │  ws streaming    │     ┌──────────────────┐
                    │  prompts/memory  │────▶│  Postgres 17     │
                    │  safety/recovery │◀────│  + pgvector      │
                    └──────────────────┘     └──────────────────┘
```

**Key decisions:**
- **Hosted LLMs, not self-hosted.** Provider interface (`internal/llm.Provider`) keeps Anthropic and OpenAI swappable via `LLM_PROVIDER` env. Currently running Claude.
- **System prompts, not fine-tuning (for now).** Persona + modality prompts live in `internal/prompts`. Fine-tuning is a Phase 5 project.
- **Session lifecycle over WS**: `message` → streamed `delta`/`done`; `wrap_up` → summary generation → session persisted → `summary` event. Sessions abandoned without wrap-up are not saved.
- **Memory v1**: last 8 session summaries formatted into the system prompt (`prompts.MemoryContext`). Cheap, effective; upgraded to embeddings-based RAG in Phase 5.
- **Postgres + pgvector** — `journal_sessions` (transcript JSONB, summary, insights, mood label) plus `session_embeddings` reserved for Phase 5.
- **JWT auth + "remember me"** — 1h access tokens, 30d single-use rotating refresh tokens, argon2id password hashing. The desktop app persists the refresh token and silently restores the session on launch, so users stay logged in.
- **Data lives server-side, encrypted.** Journal data is stored in *our* Postgres (never a third party's), per-user, with app-level AES-GCM encryption at rest coming in the hardening milestone. Users can export and delete everything. Fully-local "vault mode" storage is a post-MVP research item — memory injection requires the server to read past summaries, so local-only storage breaks the core product. Full reasoning in [MVP.md](MVP.md).
- **Server first, then frontend.** Every feature ships as: data model → endpoint → curl/wstest verification → screen. The API contract drives the UI, not the other way around.
- **Deployment target** — AWS EC2 (server + Postgres in Docker Compose, Caddy for TLS), Elastic IP + domain. RDS and S3 backups later. No GPU needed.

## Roadmap

Phases 0–2 are complete. The remaining MVP work is broken into milestones **M1–M6**, each fully specced in [MVP.md](MVP.md):

| Phase | Scope | Milestone |
|-------|-------|-----------|
| **0** | Repos, scaffolding, deps, docker-compose, prompt drafts | "Projects exist" ✅ |
| **1** | Auth (register/login/refresh + persistent login), WS hub, streaming chat | "It's alive" ✅ |
| **2** | Session model: wrap-up + summaries + insights, timeline, memory v1 | "It's a journal" ✅ |
| **M1** | Mood: score capture at wrap-up, trends endpoint, Mood screen | "It's rounded" ✅ |
| **M2** | Recovery: profile, craving logs, streaks, Recovery screen | "It's therapeutic" ✅ |
| **M3** | Weekly reports: 7-day synthesis generation, Reports screen | "It's insightful" ✅ |
| **M4** | Settings (profile, export, delete) + modality education UI | "It's respectful" ✅ |
| **M5** | Safety + hardening: crisis pipeline, encryption at rest, rate limits | "It's safe" ✅ |
| **M6** | AWS deploy: EC2 + Compose + Caddy, domain, TLS | "It's live" |
| **Post-MVP** | Strengths Mirror, Ask Fern (RAG), notepad, voice, therapist export, fine-tuning | "It's smart" |

## API surface

```
POST   /api/v1/auth/register        ✅
POST   /api/v1/auth/login           ✅
POST   /api/v1/auth/refresh         ✅
GET    /api/v1/chat/ws              ✅  WebSocket: session streaming + wrap-up
GET    /api/v1/journal              ✅  list sessions (summaries)
GET    /api/v1/journal/{id}         ✅  session detail incl. transcript
GET    /api/v1/mood/trends          ✅  daily averages, streak, top labels
GET    /api/v1/recovery             ✅  profile + stats + craving logs in one call
PUT    /api/v1/recovery/profile     ✅  set/replace recovery focus + sober date
DELETE /api/v1/recovery/profile     ✅
POST   /api/v1/recovery/craving     ✅  intensity, trigger, note, lapsed flag
POST   /api/v1/reports/weekly       ✅  generate trailing-7-day report (?force=true to regen)
GET    /api/v1/reports              ✅  list past reports
GET    /api/v1/me                   ✅
PATCH  /api/v1/me                   ✅  display name, default modality
GET    /api/v1/me/export            ✅  full JSON data export
DELETE /api/v1/me                   ✅  hard delete, cascades everything
```

## WS protocol

Client → server:
- `{"type":"message","text":"...","modality":"cbt"}`
- `{"type":"wrap_up"}`

Server → client:
- `{"type":"delta","text":"..."}` — streamed token(s)
- `{"type":"done"}` — assistant turn complete
- `{"type":"summary","sessionId":"...","summary":"...","insights":[...],"moodLabel":"...","moodScore":7}` — session saved
- `{"type":"crisis"}` — high-risk language detected; client shows helpline banner
- `{"type":"error","message":"..."}`

## Safety posture

- Fern never claims to be a therapist; system prompt enforces boundaries.
- Crisis detection ✅ — server-side phrase screen on every inbound message (`internal/safety`); client shows a dismissible helpline banner (988, findahelpline.com) without interrupting the conversation.
- Transcripts encrypted at rest ✅ — AES-256-GCM (`internal/crypto`), key from `ENCRYPTION_KEY` env; legacy plaintext rows pass through readable.
- Rate limiting ✅ — per-IP token buckets: tight on auth endpoints (brute-force), generous globally.
- No data sold, no third-party analytics on journal content. API providers receive conversation text (documented clearly to users).

# Fern — MVP Plan & Architecture Guide

This is the master planning doc: what Fern is, where every piece lives, what the MVP includes, and the order we build it in. The high-level pillars live in [PLAN.md](PLAN.md); this doc is the working blueprint.

---

## Part 1 — Understanding what you're building

### The four pieces

```
┌────────────────────┐         ┌─────────────────────┐
│    FernDesktop     │  HTTPS  │     FernServer      │
│  (Electron app on  │ ──────▶ │   (Go binary, one   │
│   the user's Mac/  │ ◀────── │  process, any VPS)  │
│      PC/Linux)     │   WS    │                     │
└────────────────────┘         └──────┬───────┬──────┘
                                      │       │
                               SQL    │       │  HTTPS
                                      ▼       ▼
                            ┌──────────┐   ┌──────────────┐
                            │ Postgres │   │ Anthropic /  │
                            │(+pgvector)│  │ OpenAI API   │
                            └──────────┘   └──────────────┘
```

1. **FernDesktop** — the Electron app. Renders UI, holds the user's login tokens, streams tokens over WebSocket. Stores *nothing else* — it's a window into the server.
2. **FernServer** — the Go binary. Owns auth, sessions, memory, prompts, safety, and is the *only* thing that talks to the database and the LLM. All intelligence-shaping (system prompts, modality logic, memory injection) lives here.
3. **Postgres** — the single source of truth. Users, sessions, transcripts, summaries, moods, recovery data. Today: a Docker container on your Mac (`fern-db`, volume `fernserver_fern-pgdata`). In production: same container running on the VPS/AWS next to the server.
4. **The LLM API** — Anthropic (or OpenAI). Stateless from our perspective: we send the system prompt + conversation each turn, it streams a response. It stores nothing for us.

### What happens when a user sends a message (the full loop)

1. User types and hits **Continue** in the desktop app.
2. Renderer sends `{"type":"message","text":...,"modality":"cbt"}` over the authenticated WebSocket.
3. Server verifies the JWT, loads the user's last 8 session summaries from Postgres (**this is the memory**).
4. Server assembles the system prompt: base persona + memory context + modality instructions + session arc + safety rules.
5. Server calls the Anthropic API with the system prompt + this session's message history, streaming.
6. Tokens stream back → server relays each as `{"type":"delta"}` → renderer appends to the bubble in real time.
7. On **Wrap up**: server sends the transcript to the LLM with the summary prompt, gets back summary/insights/mood, saves the whole session to Postgres, returns the summary card.
8. Next session, step 3 now includes this session. That's the flywheel.

### Where data lives (and the privacy position)

| Data | Where | Notes |
|------|-------|-------|
| Password | Postgres, argon2id hashed | never recoverable, only verifiable |
| Login session | Refresh token in app localStorage + hashed copy in Postgres | this is why you stay logged in between launches ("remember me" — already built) |
| Transcripts, summaries, insights | Postgres, per-user rows | encryption at rest is an MVP-hardening task (AES-GCM, key from server env) |
| Mood / recovery data | Postgres, per-user rows | same |
| Conversation text in flight | Anthropic API during the session | required for the AI to respond; covered by their no-training API terms; disclosed to users plainly |

**Why server-side storage and not on-device:** memory injection (step 3) happens server-side before the LLM call; the server must read past summaries. Local-only storage would also mean no multi-device access and total loss if the laptop dies. The honest privacy promise is: *your data lives on Fern's server, encrypted at rest, tied to your account, exportable and deletable at any time, never sold, never used to train models.* A fully-local "vault mode" is a post-MVP research item, not an MVP blocker.

---

## Part 2 — MVP definition

**The MVP is:** a deployed, multi-user desktop journaling app where sessions feel therapeutic, Fern remembers you, moods are tracked and visible, recovery tracking works, weekly reports generate, and the basics (auth, settings, safety) are solid.

### In the MVP

- [x] Auth: register / login / refresh / persistent sessions
- [x] Streaming session chat with 6 modalities
- [x] Wrap-up → summary + insights + mood label, saved
- [x] Memory v1: recent summaries injected into every new session
- [x] Timeline of past sessions
- [x] Modality education: in-app explainers for choosing a modality (guide modal in the selector)
- [x] Mood: numeric capture per session (1–10, inferred at wrap-up) + trends screen
- [x] Recovery: profile, cravings, triggers, streaks + screen; recovery context injected into sessions
- [x] Reports: weekly insight report (the Rosebud-style 7-day synthesis)
- [x] Settings: profile, default modality, data export, delete account
- [x] Crisis safety net: server-side detection + helpline banner
- [x] Hardening: AES-GCM transcript encryption at rest, per-IP rate limiting, configurable origins
- [ ] Deployed on AWS, real domain, TLS (M6 — the last MVP milestone)

### NOT in the MVP (Full-app ideas, in rough order)

1. **Strengths Mirror** — see [STRENGTHS_MIRROR.md](STRENGTHS_MIRROR.md)
2. **Ask Fern** — RAG over full journal history (pgvector is already in the schema)
3. Freeform notepad (no-AI open journal)
4. Voice journaling (Whisper transcription)
5. Guided journal "experiences" (structured multi-step exercises)
6. Therapist export (clean PDF for a real therapist)
7. Mobile app / web app
8. Fine-tuned model
9. Billing / subscriptions
10. Local "vault mode" storage research

---

## Part 3 — Build order (server first, then frontend)

Agreed: **server → frontend** for every feature. The server work defines the data model and API contract; the frontend then has something real to build against. Rhythm: ship the endpoint, test it with curl/wstest, then build the screen.

### M1 — Mood (server → UI)
- Server: extend wrap-up to also produce `mood_score` (1–10) alongside the label; `GET /api/v1/mood/trends` (daily averages, label frequencies, streak of journaling days).
- UI: Mood screen — line/area chart of score over time, recent mood chips, simple stats (avg this week vs last).

### M2 — Recovery (server → UI)
- Server: `recovery_profiles` (what they're recovering from, sobriety date), `craving_logs` (intensity 1–10, trigger, what happened, timestamp). Endpoints: log craving, list, streak computation. MI modality gets recovery context injected (streak, recent cravings) when a profile exists.
- UI: Recovery screen — streak counter (days), "log a craving" quick action, craving history with intensity dots, trigger patterns list.

### M3 — Weekly reports (server → UI)
- Server: `reports` table; generation endpoint `POST /api/v1/reports/weekly` (MVP: generated on demand when the user opens Reports and the week is complete; cron later). Input: the week's session summaries + moods + recovery events → LLM produces: themes, emotional landscape, wins, growth edges, a gentle challenge for next week.
- UI: Reports screen — report cards by week, latest expanded.

### M4 — Settings + modality education (server → UI)
- Server: `PATCH /api/v1/me` (display name, default modality), `GET /api/v1/me/export` (full JSON dump), `DELETE /api/v1/me` (hard delete). 
- UI: Settings screen — profile, default modality, export button, danger zone. Modality selector gets an "About" panel per modality: what it is, when to pick it, what a session feels like (copy written per-modality, sourced from the prompt design).

### M5 — Safety + hardening
- Server: crisis keyword screen on inbound messages + model-flag instruction in prompts; `{"type":"crisis"}` WS event. AES-GCM encryption at rest for transcripts. Rate limiting (per-user, per-IP). CORS/WS origin tightening.
- UI: crisis banner component (helplines: 988 US/CA, findahelpline.com), shown without killing the conversation.

### M6 — AWS deployment
- Dockerfile for the server; docker-compose (server + Postgres + Caddy) on a single **EC2** instance (t3.small class), Elastic IP, domain + TLS via Caddy.
- Why EC2-with-compose first: it's the same compose file you run locally — least new concepts, most learning per hour. Postgres moves to **RDS** later when uptime matters; S3 for backups is step two (nightly `pg_dump`).
- Learning path this gives you: EC2, security groups, Elastic IPs, IAM basics, S3, and later RDS — the practical 80% of AWS.

Frontend polish (chat affordances, transitions, empty states) rides along inside each milestone, after its server piece lands.

---

## Part 4 — Feature specs (first pass)

### Mood
Rosebud-style: mood is captured *from the journaling itself* (the wrap-up already infers it), not a chore. Numeric score joins the label. Trends screen answers: "how have I actually been?" Chart, weekly average deltas, most-frequent feelings. Later: correlate with recovery events.

### Recovery
The differentiator. One active recovery profile per user (substance or behavior + start date). Craving logs are fast to enter (two taps + optional note). Streak math respects honesty — a logged lapse resets the counter without shame language; the MI modality knows about it next session. Trigger field builds a personal trigger map over time.

### Weekly report (the Rosebud "insights" you liked)
Per-session insights already exist (the summary card). The weekly report is the zoom-out: what themes kept surfacing, how mood moved, what you worked through, one thing to sit with next week. Generated from *summaries*, not raw transcripts — cheaper, and it's already-distilled signal.

### Settings
Profile + preferences + data rights. Export and delete are non-negotiable for a journaling app — they're the proof behind the privacy promise.

### Modality education
Each modality gets: a one-line pitch (already in the selector), a "when to choose this" paragraph, and a "what a session feels like" example exchange. Surfaced via an info icon in the selector and a first-run explainer.

---

## Part 5 — Where we go from here

Recommended immediate next step: **M1 (Mood)** — it's the smallest server slice, it makes the Mood tab real, and it starts building the data that makes M3's weekly reports good. Then M2 → M3 → M4 → M5 → M6 in order.

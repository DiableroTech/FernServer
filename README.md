# Fern Server

Backend for **[Fern](https://github.com/DiableroTech/FernDesktop)** — a privacy-first AI journaling companion inspired by Rosebud, with a sharper focus on therapeutic frameworks and addictions recovery.

Fern is not a chatbot you dump thoughts into once. It's **session-based journaling**: you have a guided conversation, wrap up when you're done, and Fern saves a summary with insights and a mood read. Next time you open the app, Fern remembers — recent sessions inform how it shows up for you.

The desktop app lives in [FernDesktop](https://github.com/DiableroTech/FernDesktop). This repo is the server that powers it.

## What Fern does

**Guided journal sessions** — Write freely; Fern responds in real time with a voice shaped by evidence-based therapeutic approaches. When you're finished, wrap up and get a summary card: key themes, insights, and how you seemed to be feeling.

**Therapeutic modalities** — Each session runs under a framework you choose (or a default): CBT, ACT, DBT, Motivational Interviewing, IFS, or freeform. Fern adapts its style to match — structured when you want structure, spacious when you need space.

**Memory** — Fern carries context forward. Recent session summaries are woven into new conversations so it feels like continuity, not starting from zero every night.

**Mood tracking** — Mood is captured at wrap-up and tracked over time so you can see patterns, not just individual entries.

**Recovery support** — A dedicated module for cravings, triggers, and sobriety streaks. Recovery context can inform sessions when you're working on substance use or behavioral change.

**Weekly reports** — A Rosebud-style synthesis of your last seven days: themes, wins, and a gentle challenge to carry forward.

**Your data, your control** — Export everything as JSON or delete your account entirely. Journal transcripts are encrypted at rest on the server.

## Privacy & safety

- Your journal lives on **your Fern server** (self-hosted or deployed), not in a third-party app's database.
- Passwords are hashed with argon2id; sessions use short-lived JWTs with rotating refresh tokens.
- Transcripts are encrypted at rest (AES-256-GCM) when configured for production.
- Conversation text is sent to a hosted LLM (OpenAI or Anthropic) only to generate responses — covered by API no-training terms, not used to train models.
- **Fern is self-help journaling, not therapy.** It does not diagnose, prescribe, or replace professional care. Crisis language triggers a safety response with helpline resources (988 in US/Canada, [findahelpline.com](https://findahelpline.com) internationally).

## Status

MVP features (auth, sessions, memory, mood, recovery, reports, settings, safety) are implemented. Hosted deployment (AWS + TLS) is the remaining milestone before a public launch.

## For developers

Setup, environment variables, repo layout, and API reference:

**[docs/code-arch.md](docs/code-arch.md)**

Roadmap and detailed architecture: [docs/PLAN.md](docs/PLAN.md) · [docs/MVP.md](docs/MVP.md)

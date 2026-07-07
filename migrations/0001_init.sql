-- Initial schema: users, sessions, journal entries, moods.
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE journal_entries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    modality   TEXT NOT NULL DEFAULT 'freeform',
    -- Full conversation transcript, encrypted at rest by the application.
    content    BYTEA NOT NULL,
    mood       SMALLINT,           -- 1-10, nullable if not captured
    mood_label TEXT,               -- e.g. "anxious", "hopeful"
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_journal_entries_user_created ON journal_entries (user_id, created_at DESC);

-- Phase 5: embeddings for RAG / "Ask Fern"
CREATE TABLE entry_embeddings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Replace the unused journal_entries model with session-based journaling.
DROP TABLE IF EXISTS entry_embeddings;
DROP TABLE IF EXISTS journal_entries;

CREATE TABLE journal_sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    modality   TEXT NOT NULL DEFAULT 'freeform',
    transcript JSONB NOT NULL DEFAULT '[]',
    summary    TEXT NOT NULL DEFAULT '',
    insights   JSONB NOT NULL DEFAULT '[]',
    mood_label TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_journal_sessions_user ON journal_sessions (user_id, ended_at DESC);

-- Phase 5: embeddings for RAG / "Ask Fern", now keyed to sessions.
CREATE TABLE session_embeddings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES journal_sessions(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

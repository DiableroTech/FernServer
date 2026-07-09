-- One active recovery focus per user for MVP.
CREATE TABLE recovery_profiles (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    focus       TEXT NOT NULL,            -- e.g. "alcohol", "gambling"
    sober_since DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE craving_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    intensity  SMALLINT NOT NULL CHECK (intensity BETWEEN 1 AND 10),
    trigger    TEXT NOT NULL DEFAULT '',
    note       TEXT NOT NULL DEFAULT '',
    lapsed     BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_craving_logs_user ON craving_logs (user_id, created_at DESC);

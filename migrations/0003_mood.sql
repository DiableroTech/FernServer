-- Numeric mood alongside the label; 0 = not scored.
ALTER TABLE journal_sessions ADD COLUMN mood_score SMALLINT NOT NULL DEFAULT 0
    CHECK (mood_score BETWEEN 0 AND 10);

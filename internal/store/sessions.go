package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Session struct {
	ID         string          `json:"id"`
	Modality   string          `json:"modality"`
	Summary    string          `json:"summary"`
	Insights   []string        `json:"insights"`
	MoodLabel  string          `json:"moodLabel"`
	MoodScore  int             `json:"moodScore"`
	StartedAt  time.Time       `json:"startedAt"`
	EndedAt    time.Time       `json:"endedAt"`
	Transcript json.RawMessage `json:"transcript,omitempty"`
}

func (s *Store) CreateSession(ctx context.Context, userID, modality string, transcript json.RawMessage, summary string, insights []string, moodLabel string, moodScore int, startedAt time.Time) (string, error) {
	insightsJSON, err := json.Marshal(insights)
	if err != nil {
		return "", err
	}
	if moodScore < 0 || moodScore > 10 {
		moodScore = 0
	}
	if s.Enc != nil {
		if sealed, err := s.Enc.EncryptJSON(transcript); err == nil {
			transcript = sealed
		} else {
			return "", fmt.Errorf("encrypt transcript: %w", err)
		}
	}
	var id string
	err = s.Pool.QueryRow(ctx,
		`INSERT INTO journal_sessions (user_id, modality, transcript, summary, insights, mood_label, mood_score, started_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		userID, modality, transcript, summary, insightsJSON, moodLabel, moodScore, startedAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

func (s *Store) ListSessions(ctx context.Context, userID string, limit int) ([]Session, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, modality, summary, insights, mood_label, mood_score, started_at, ended_at
		 FROM journal_sessions
		 WHERE user_id = $1
		 ORDER BY ended_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []Session{}
	for rows.Next() {
		var sess Session
		var insights []byte
		if err := rows.Scan(&sess.ID, &sess.Modality, &sess.Summary, &insights, &sess.MoodLabel, &sess.MoodScore, &sess.StartedAt, &sess.EndedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(insights, &sess.Insights)
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) GetSession(ctx context.Context, userID, sessionID string) (*Session, error) {
	sess := &Session{}
	var insights []byte
	err := s.Pool.QueryRow(ctx,
		`SELECT id, modality, transcript, summary, insights, mood_label, mood_score, started_at, ended_at
		 FROM journal_sessions
		 WHERE id = $1 AND user_id = $2`,
		sessionID, userID,
	).Scan(&sess.ID, &sess.Modality, &sess.Transcript, &sess.Summary, &insights, &sess.MoodLabel, &sess.MoodScore, &sess.StartedAt, &sess.EndedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	_ = json.Unmarshal(insights, &sess.Insights)
	if err := s.decryptTranscript(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Store) decryptTranscript(sess *Session) error {
	if s.Enc == nil || len(sess.Transcript) == 0 {
		return nil
	}
	raw, err := s.Enc.DecryptJSON(sess.Transcript)
	if err != nil {
		return fmt.Errorf("session %s: %w", sess.ID, err)
	}
	sess.Transcript = raw
	return nil
}

// ExportSessions returns every session with full transcript for data export.
func (s *Store) ExportSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, modality, transcript, summary, insights, mood_label, mood_score, started_at, ended_at
		 FROM journal_sessions
		 WHERE user_id = $1
		 ORDER BY ended_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("export sessions: %w", err)
	}
	defer rows.Close()

	sessions := []Session{}
	for rows.Next() {
		var sess Session
		var insights []byte
		if err := rows.Scan(&sess.ID, &sess.Modality, &sess.Transcript, &sess.Summary, &insights, &sess.MoodLabel, &sess.MoodScore, &sess.StartedAt, &sess.EndedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(insights, &sess.Insights)
		if err := s.decryptTranscript(&sess); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

type SessionSummary struct {
	EndedAt   time.Time
	Modality  string
	Summary   string
	MoodLabel string
}

// RecentSummaries powers memory injection: what Fern "remembers" going
// into a new session.
func (s *Store) RecentSummaries(ctx context.Context, userID string, limit int) ([]SessionSummary, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT ended_at, modality, summary, mood_label
		 FROM journal_sessions
		 WHERE user_id = $1 AND summary <> ''
		 ORDER BY ended_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent summaries: %w", err)
	}
	defer rows.Close()

	var out []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		if err := rows.Scan(&ss.EndedAt, &ss.Modality, &ss.Summary, &ss.MoodLabel); err != nil {
			return nil, err
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

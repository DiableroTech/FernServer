package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Report struct {
	ID          string          `json:"id"`
	PeriodStart string          `json:"periodStart"`
	PeriodEnd   string          `json:"periodEnd"`
	Content     json.RawMessage `json:"content"`
	CreatedAt   time.Time       `json:"createdAt"`
}

func (s *Store) UpsertReport(ctx context.Context, userID string, periodStart, periodEnd time.Time, content json.RawMessage) (*Report, error) {
	r := &Report{
		PeriodStart: periodStart.Format("2006-01-02"),
		PeriodEnd:   periodEnd.Format("2006-01-02"),
		Content:     content,
	}
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO reports (user_id, period_start, period_end, content)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, period_end) DO UPDATE
		 SET content = EXCLUDED.content, period_start = EXCLUDED.period_start, created_at = now()
		 RETURNING id, created_at`,
		userID, periodStart, periodEnd, content,
	).Scan(&r.ID, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert report: %w", err)
	}
	return r, nil
}

func (s *Store) GetReportByPeriodEnd(ctx context.Context, userID string, periodEnd time.Time) (*Report, error) {
	r := &Report{}
	var start, end time.Time
	err := s.Pool.QueryRow(ctx,
		`SELECT id, period_start, period_end, content, created_at
		 FROM reports WHERE user_id = $1 AND period_end = $2`,
		userID, periodEnd,
	).Scan(&r.ID, &start, &end, &r.Content, &r.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	r.PeriodStart = start.Format("2006-01-02")
	r.PeriodEnd = end.Format("2006-01-02")
	return r, nil
}

func (s *Store) ListReports(ctx context.Context, userID string, limit int) ([]Report, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, period_start, period_end, content, created_at
		 FROM reports
		 WHERE user_id = $1
		 ORDER BY period_end DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	reports := []Report{}
	for rows.Next() {
		var r Report
		var start, end time.Time
		if err := rows.Scan(&r.ID, &start, &end, &r.Content, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.PeriodStart = start.Format("2006-01-02")
		r.PeriodEnd = end.Format("2006-01-02")
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

type WeekSession struct {
	EndedAt   time.Time
	Modality  string
	Summary   string
	Insights  []string
	MoodLabel string
	MoodScore int
}

// SessionsBetween returns wrapped-up sessions in [from, to) for report digests.
func (s *Store) SessionsBetween(ctx context.Context, userID string, from, to time.Time) ([]WeekSession, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT ended_at, modality, summary, insights, mood_label, mood_score
		 FROM journal_sessions
		 WHERE user_id = $1 AND ended_at >= $2 AND ended_at < $3
		 ORDER BY ended_at ASC`,
		userID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("sessions between: %w", err)
	}
	defer rows.Close()

	var out []WeekSession
	for rows.Next() {
		var ws WeekSession
		var insights []byte
		if err := rows.Scan(&ws.EndedAt, &ws.Modality, &ws.Summary, &insights, &ws.MoodLabel, &ws.MoodScore); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(insights, &ws.Insights)
		out = append(out, ws)
	}
	return out, rows.Err()
}

// CravingsBetween returns craving logs in [from, to) for report digests.
func (s *Store) CravingsBetween(ctx context.Context, userID string, from, to time.Time) ([]CravingLog, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, intensity, trigger, note, lapsed, created_at
		 FROM craving_logs
		 WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		 ORDER BY created_at ASC`,
		userID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("cravings between: %w", err)
	}
	defer rows.Close()

	logs := []CravingLog{}
	for rows.Next() {
		var c CravingLog
		if err := rows.Scan(&c.ID, &c.Intensity, &c.Trigger, &c.Note, &c.Lapsed, &c.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, c)
	}
	return logs, rows.Err()
}

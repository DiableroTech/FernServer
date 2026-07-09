package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type RecoveryProfile struct {
	Focus      string    `json:"focus"`
	SoberSince time.Time `json:"soberSince"`
	CreatedAt  time.Time `json:"createdAt"`
}

type CravingLog struct {
	ID        string    `json:"id"`
	Intensity int       `json:"intensity"`
	Trigger   string    `json:"trigger"`
	Note      string    `json:"note"`
	Lapsed    bool      `json:"lapsed"`
	CreatedAt time.Time `json:"createdAt"`
}

type TriggerCount struct {
	Trigger string `json:"trigger"`
	Count   int    `json:"count"`
}

type RecoveryStats struct {
	StreakDays    int            `json:"streakDays"`
	CravingsWeek  int            `json:"cravingsWeek"`
	AvgIntensity  float64        `json:"avgIntensity"` // last 30 days, 0 if none
	TopTriggers   []TriggerCount `json:"topTriggers"`
	TotalCravings int            `json:"totalCravings"`
}

func (s *Store) UpsertRecoveryProfile(ctx context.Context, userID, focus string, soberSince time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO recovery_profiles (user_id, focus, sober_since)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		 SET focus = EXCLUDED.focus, sober_since = EXCLUDED.sober_since, updated_at = now()`,
		userID, focus, soberSince,
	)
	if err != nil {
		return fmt.Errorf("upsert recovery profile: %w", err)
	}
	return nil
}

func (s *Store) GetRecoveryProfile(ctx context.Context, userID string) (*RecoveryProfile, error) {
	p := &RecoveryProfile{}
	err := s.Pool.QueryRow(ctx,
		`SELECT focus, sober_since, created_at FROM recovery_profiles WHERE user_id = $1`,
		userID,
	).Scan(&p.Focus, &p.SoberSince, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get recovery profile: %w", err)
	}
	return p, nil
}

func (s *Store) DeleteRecoveryProfile(ctx context.Context, userID string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM recovery_profiles WHERE user_id = $1`, userID)
	return err
}

func (s *Store) CreateCravingLog(ctx context.Context, userID string, intensity int, trigger, note string, lapsed bool) (*CravingLog, error) {
	c := &CravingLog{Intensity: intensity, Trigger: trigger, Note: note, Lapsed: lapsed}
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO craving_logs (user_id, intensity, trigger, note, lapsed)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		userID, intensity, trigger, note, lapsed,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create craving log: %w", err)
	}
	return c, nil
}

func (s *Store) ListCravingLogs(ctx context.Context, userID string, limit int) ([]CravingLog, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, intensity, trigger, note, lapsed, created_at
		 FROM craving_logs
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list craving logs: %w", err)
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

// RecoveryStats computes the streak from sober_since, reset by the most
// recent lapse. Lapses reset the counter; they don't erase history.
func (s *Store) RecoveryStats(ctx context.Context, userID string, soberSince time.Time) (*RecoveryStats, error) {
	st := &RecoveryStats{TopTriggers: []TriggerCount{}}

	var lastLapse *time.Time
	err := s.Pool.QueryRow(ctx,
		`SELECT MAX(created_at) FROM craving_logs WHERE user_id = $1 AND lapsed`,
		userID,
	).Scan(&lastLapse)
	if err != nil {
		return nil, fmt.Errorf("last lapse: %w", err)
	}

	streakStart := soberSince
	if lastLapse != nil && lastLapse.After(streakStart) {
		streakStart = *lastLapse
	}
	if d := int(time.Since(streakStart).Hours() / 24); d > 0 {
		st.StreakDays = d
	}

	err = s.Pool.QueryRow(ctx,
		`SELECT
		   COUNT(*) FILTER (WHERE created_at >= now() - interval '7 days'),
		   COALESCE(AVG(intensity) FILTER (WHERE created_at >= now() - interval '30 days'), 0),
		   COUNT(*)
		 FROM craving_logs WHERE user_id = $1`,
		userID,
	).Scan(&st.CravingsWeek, &st.AvgIntensity, &st.TotalCravings)
	if err != nil {
		return nil, fmt.Errorf("craving stats: %w", err)
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT trigger, COUNT(*)
		 FROM craving_logs
		 WHERE user_id = $1 AND trigger <> ''
		 GROUP BY trigger
		 ORDER BY COUNT(*) DESC
		 LIMIT 5`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("top triggers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tc TriggerCount
		if err := rows.Scan(&tc.Trigger, &tc.Count); err != nil {
			return nil, err
		}
		st.TopTriggers = append(st.TopTriggers, tc)
	}
	return st, rows.Err()
}

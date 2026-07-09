package store

import (
	"context"
	"fmt"
	"time"
)

type MoodPoint struct {
	Date     string  `json:"date"` // YYYY-MM-DD
	AvgScore float64 `json:"avgScore"`
	Sessions int     `json:"sessions"`
	Labels   []string `json:"labels"`
}

type MoodLabelCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type MoodTrends struct {
	Points        []MoodPoint      `json:"points"`
	TopLabels     []MoodLabelCount `json:"topLabels"`
	AvgThisWeek   float64          `json:"avgThisWeek"`
	AvgLastWeek   float64          `json:"avgLastWeek"`
	JournalStreak int              `json:"journalStreak"` // consecutive days ending today/yesterday
	TotalSessions int              `json:"totalSessions"`
}

func (s *Store) MoodTrends(ctx context.Context, userID string, days int) (*MoodTrends, error) {
	since := time.Now().AddDate(0, 0, -days)

	rows, err := s.Pool.Query(ctx,
		`SELECT (ended_at AT TIME ZONE 'UTC')::date AS day,
		        COALESCE(AVG(mood_score) FILTER (WHERE mood_score > 0), 0),
		        COUNT(*),
		        ARRAY_AGG(mood_label) FILTER (WHERE mood_label <> '')
		 FROM journal_sessions
		 WHERE user_id = $1 AND ended_at >= $2
		 GROUP BY day
		 ORDER BY day ASC`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("mood trends: %w", err)
	}
	defer rows.Close()

	t := &MoodTrends{Points: []MoodPoint{}, TopLabels: []MoodLabelCount{}}
	for rows.Next() {
		var p MoodPoint
		var day time.Time
		var labels []string
		if err := rows.Scan(&day, &p.AvgScore, &p.Sessions, &labels); err != nil {
			return nil, err
		}
		p.Date = day.Format("2006-01-02")
		p.Labels = labels
		t.Points = append(t.Points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	err = s.Pool.QueryRow(ctx,
		`SELECT
		   COALESCE(AVG(mood_score) FILTER (WHERE mood_score > 0 AND ended_at >= now() - interval '7 days'), 0),
		   COALESCE(AVG(mood_score) FILTER (WHERE mood_score > 0 AND ended_at >= now() - interval '14 days' AND ended_at < now() - interval '7 days'), 0),
		   COUNT(*)
		 FROM journal_sessions WHERE user_id = $1`,
		userID,
	).Scan(&t.AvgThisWeek, &t.AvgLastWeek, &t.TotalSessions)
	if err != nil {
		return nil, fmt.Errorf("mood averages: %w", err)
	}

	labelRows, err := s.Pool.Query(ctx,
		`SELECT mood_label, COUNT(*)
		 FROM journal_sessions
		 WHERE user_id = $1 AND mood_label <> '' AND ended_at >= $2
		 GROUP BY mood_label
		 ORDER BY COUNT(*) DESC
		 LIMIT 6`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("mood labels: %w", err)
	}
	defer labelRows.Close()
	for labelRows.Next() {
		var lc MoodLabelCount
		if err := labelRows.Scan(&lc.Label, &lc.Count); err != nil {
			return nil, err
		}
		t.TopLabels = append(t.TopLabels, lc)
	}
	if err := labelRows.Err(); err != nil {
		return nil, err
	}

	// Consecutive journaling days; streak is alive if the latest entry is today or yesterday.
	err = s.Pool.QueryRow(ctx,
		`WITH days AS (
		   SELECT DISTINCT (ended_at AT TIME ZONE 'UTC')::date AS day
		   FROM journal_sessions WHERE user_id = $1
		 ),
		 anchor AS (SELECT MAX(day) AS a FROM days WHERE day >= CURRENT_DATE - 1),
		 numbered AS (
		   SELECT day, ROW_NUMBER() OVER (ORDER BY day DESC) AS rn FROM days
		 )
		 SELECT COUNT(*) FROM numbered, anchor
		 WHERE anchor.a IS NOT NULL AND day = anchor.a - (rn - 1)::int`,
		userID,
	).Scan(&t.JournalStreak)
	if err != nil {
		return nil, fmt.Errorf("journal streak: %w", err)
	}

	return t, nil
}

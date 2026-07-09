package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/llm"
	"github.com/DiableroTech/fern-server/internal/prompts"
	"github.com/DiableroTech/fern-server/internal/store"
)

type reportHandlers struct {
	store    *store.Store
	provider llm.Provider
}

func (h *reportHandlers) list(w http.ResponseWriter, r *http.Request) {
	reports, err := h.store.ListReports(r.Context(), auth.UserID(r.Context()), 52)
	if err != nil {
		slog.Error("list reports", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load reports")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

// generateWeekly builds a report for the trailing 7 days. Idempotent per
// day: regenerating on the same day overwrites, a new day makes a new report.
func (h *reportHandlers) generateWeekly(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	now := time.Now()
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	periodStart := periodEnd.AddDate(0, 0, -7)

	if existing, err := h.store.GetReportByPeriodEnd(r.Context(), userID, periodEnd); err == nil && r.URL.Query().Get("force") != "true" {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	sessions, err := h.store.SessionsBetween(r.Context(), userID, periodStart, periodEnd.AddDate(0, 0, 1))
	if err != nil {
		slog.Error("report sessions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to gather your week")
		return
	}
	if len(sessions) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no sessions this week — journal first, then come back for your report")
		return
	}
	cravings, err := h.store.CravingsBetween(r.Context(), userID, periodStart, periodEnd.AddDate(0, 0, 1))
	if err != nil {
		slog.Error("report cravings", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to gather your week")
		return
	}

	content, err := h.generate(r.Context(), buildWeekDigest(sessions, cravings))
	if err != nil {
		slog.Error("generate weekly report", "user", userID, "error", err)
		writeError(w, http.StatusBadGateway, "report generation is unavailable right now")
		return
	}

	report, err := h.store.UpsertReport(r.Context(), userID, periodStart, periodEnd, content)
	if err != nil {
		slog.Error("save report", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to save your report")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func buildWeekDigest(sessions []store.WeekSession, cravings []store.CravingLog) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("The past week: %d journaling session(s).\n\nSessions:\n", len(sessions)))
	for _, s := range sessions {
		b.WriteString(fmt.Sprintf("- %s (%s, mood: %s", s.EndedAt.Format("Mon Jan 2"), s.Modality, s.MoodLabel))
		if s.MoodScore > 0 {
			b.WriteString(fmt.Sprintf(" %d/10", s.MoodScore))
		}
		b.WriteString(") " + s.Summary + "\n")
		for _, in := range s.Insights {
			b.WriteString("    insight: " + in + "\n")
		}
	}
	if len(cravings) > 0 {
		b.WriteString(fmt.Sprintf("\nRecovery activity: %d craving(s) logged.\n", len(cravings)))
		for _, c := range cravings {
			b.WriteString(fmt.Sprintf("- %s intensity %d/10", c.CreatedAt.Format("Mon Jan 2"), c.Intensity))
			if c.Trigger != "" {
				b.WriteString(", trigger: " + c.Trigger)
			}
			if c.Lapsed {
				b.WriteString(" (lapsed)")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (h *reportHandlers) generate(ctx context.Context, digest string) (json.RawMessage, error) {
	chunks, err := h.provider.StreamChat(ctx, prompts.WeeklyReportSystem, []llm.Message{
		{Role: llm.RoleUser, Content: digest},
	})
	if err != nil {
		return nil, err
	}
	var raw strings.Builder
	for chunk := range chunks {
		if chunk.Err != nil {
			return nil, chunk.Err
		}
		raw.WriteString(chunk.Delta)
	}

	text := strings.TrimSpace(raw.String())
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	if !json.Valid([]byte(text)) {
		return nil, errors.New("model returned invalid JSON")
	}
	return json.RawMessage(text), nil
}

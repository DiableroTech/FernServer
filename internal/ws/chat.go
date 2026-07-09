package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/llm"
	"github.com/DiableroTech/fern-server/internal/prompts"
	"github.com/DiableroTech/fern-server/internal/safety"
	"github.com/DiableroTech/fern-server/internal/store"
)

const (
	// Cap conversation context sent to the provider; oldest turns fall off.
	maxHistoryMessages = 40
	// How many past session summaries Fern "remembers" going into a session.
	memoryDepth = 8
)

type ChatHandler struct {
	Provider       llm.Provider
	Store          *store.Store
	JWTSecret      string
	OriginPatterns []string
}

type clientMessage struct {
	Type     string `json:"type"` // "message" | "wrap_up"
	Text     string `json:"text"`
	Modality string `json:"modality"`
}

type serverMessage struct {
	Type      string   `json:"type"` // "delta" | "done" | "error" | "summary"
	Text      string   `json:"text,omitempty"`
	Message   string   `json:"message,omitempty"`
	SessionID string   `json:"sessionId,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Insights  []string `json:"insights,omitempty"`
	MoodLabel string   `json:"moodLabel,omitempty"`
	MoodScore int      `json:"moodScore,omitempty"`
}

type session struct {
	history   []llm.Message
	modality  prompts.Modality
	startedAt time.Time
	memory    string
	hasMemory bool
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.VerifyAccessToken(h.JWTSecret, r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	patterns := h.OriginPatterns
	if len(patterns) == 0 {
		patterns = []string{"localhost:*"}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: patterns,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	slog.Info("chat connected", "user", userID)
	ctx := r.Context()
	sess := &session{modality: prompts.ModalityFreeform}

	for {
		var in clientMessage
		if err := wsjson.Read(ctx, conn, &in); err != nil {
			slog.Info("chat disconnected", "user", userID)
			return
		}

		switch in.Type {
		case "message":
			if strings.TrimSpace(in.Text) == "" {
				continue
			}
			if !h.handleMessage(ctx, conn, userID, sess, in) {
				return
			}
		case "wrap_up":
			if !h.handleWrapUp(ctx, conn, userID, sess) {
				return
			}
		}
	}
}

func (h *ChatHandler) handleMessage(ctx context.Context, conn *websocket.Conn, userID string, sess *session, in clientMessage) bool {
	if len(sess.history) == 0 {
		sess.startedAt = time.Now()
	}
	if m := prompts.Modality(in.Modality); m != "" {
		sess.modality = m
	}
	if !sess.hasMemory {
		sess.memory = h.loadMemory(ctx, userID)
		sess.hasMemory = true
	}

	systemPrompt, err := prompts.System(sess.modality, sess.memory)
	if err != nil {
		systemPrompt, _ = prompts.System(prompts.ModalityFreeform, sess.memory)
	}

	// Crisis tripwire: surface helplines client-side; the conversation continues.
	if safety.DetectCrisis(in.Text) {
		slog.Info("crisis language detected", "user", userID)
		if wsjson.Write(ctx, conn, serverMessage{Type: "crisis"}) != nil {
			return false
		}
	}

	sess.history = append(sess.history, llm.Message{Role: llm.RoleUser, Content: in.Text})

	windowed := sess.history
	if len(windowed) > maxHistoryMessages {
		windowed = windowed[len(windowed)-maxHistoryMessages:]
	}

	reply, ok := h.streamResponse(ctx, conn, systemPrompt, windowed)
	if reply != "" {
		sess.history = append(sess.history, llm.Message{Role: llm.RoleAssistant, Content: reply})
	}
	return ok
}

func (h *ChatHandler) handleWrapUp(ctx context.Context, conn *websocket.Conn, userID string, sess *session) bool {
	if len(sess.history) == 0 {
		return wsjson.Write(ctx, conn, serverMessage{Type: "error", Message: "nothing to wrap up yet"}) == nil
	}

	summary, insights, moodLabel, moodScore := h.generateSummary(ctx, sess.history)

	transcript, err := json.Marshal(sess.history)
	if err != nil {
		transcript = []byte("[]")
	}
	sessionID, err := h.Store.CreateSession(ctx, userID, string(sess.modality), transcript, summary, insights, moodLabel, moodScore, sess.startedAt)
	if err != nil {
		slog.Error("save session", "user", userID, "error", err)
		return wsjson.Write(ctx, conn, serverMessage{Type: "error", Message: "failed to save your session"}) == nil
	}

	slog.Info("session wrapped up", "user", userID, "session", sessionID, "turns", len(sess.history))

	// Reset for a fresh session on the same connection; memory reloads so
	// the session just saved becomes part of what Fern remembers.
	*sess = session{modality: sess.modality}

	return wsjson.Write(ctx, conn, serverMessage{
		Type:      "summary",
		SessionID: sessionID,
		Summary:   summary,
		Insights:  insights,
		MoodLabel: moodLabel,
		MoodScore: moodScore,
	}) == nil
}

func (h *ChatHandler) loadMemory(ctx context.Context, userID string) string {
	summaries, err := h.Store.RecentSummaries(ctx, userID, memoryDepth)
	if err != nil {
		slog.Error("load memory", "user", userID, "error", err)
		return ""
	}
	entries := make([]prompts.MemoryEntry, len(summaries))
	for i, s := range summaries {
		entries[i] = prompts.MemoryEntry{
			Date:      s.EndedAt.Format("Jan 2, 2006"),
			Modality:  s.Modality,
			MoodLabel: s.MoodLabel,
			Summary:   s.Summary,
		}
	}
	return prompts.MemoryContext(entries) + h.loadRecoveryContext(ctx, userID)
}

func (h *ChatHandler) loadRecoveryContext(ctx context.Context, userID string) string {
	profile, err := h.Store.GetRecoveryProfile(ctx, userID)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			slog.Error("load recovery context", "user", userID, "error", err)
		}
		return ""
	}
	stats, err := h.Store.RecoveryStats(ctx, userID, profile.SoberSince)
	if err != nil {
		slog.Error("load recovery stats", "user", userID, "error", err)
		return ""
	}
	triggers := make([]string, len(stats.TopTriggers))
	for i, t := range stats.TopTriggers {
		triggers[i] = t.Trigger
	}
	return prompts.RecoveryContext(profile.Focus, stats.StreakDays, stats.CravingsWeek, triggers)
}

// generateSummary asks the model for a JSON summary of the session and
// parses it leniently — on any failure it degrades to plain text.
func (h *ChatHandler) generateSummary(ctx context.Context, history []llm.Message) (summary string, insights []string, moodLabel string, moodScore int) {
	var b strings.Builder
	for _, m := range history {
		if m.Role == llm.RoleUser {
			b.WriteString("Person: ")
		} else {
			b.WriteString("Fern: ")
		}
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}

	chunks, err := h.Provider.StreamChat(ctx, prompts.SummarySystem, []llm.Message{
		{Role: llm.RoleUser, Content: "Session transcript:\n\n" + b.String()},
	})
	if err != nil {
		return "Session saved. Summary generation was unavailable.", nil, "", 0
	}

	var raw strings.Builder
	for chunk := range chunks {
		if chunk.Err != nil {
			slog.Error("summary generation", "error", chunk.Err)
			return "Session saved. Summary generation was unavailable.", nil, "", 0
		}
		raw.WriteString(chunk.Delta)
	}

	text := strings.TrimSpace(raw.String())
	// Models occasionally wrap JSON in fences despite instructions.
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var parsed struct {
		Summary   string   `json:"summary"`
		Insights  []string `json:"insights"`
		MoodLabel string   `json:"moodLabel"`
		MoodScore int      `json:"moodScore"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil || parsed.Summary == "" {
		return text, nil, "", 0
	}
	return parsed.Summary, parsed.Insights, parsed.MoodLabel, parsed.MoodScore
}

// streamResponse relays provider chunks to the client. Returns the full
// assistant reply and false if the connection is unusable.
func (h *ChatHandler) streamResponse(ctx context.Context, conn *websocket.Conn, systemPrompt string, history []llm.Message) (string, bool) {
	chunks, err := h.Provider.StreamChat(ctx, systemPrompt, history)
	if err != nil {
		_ = wsjson.Write(ctx, conn, serverMessage{Type: "error", Message: "the AI is unavailable right now"})
		return "", true
	}

	var full strings.Builder
	for chunk := range chunks {
		switch {
		case chunk.Err != nil:
			slog.Error("llm stream error", "provider", h.Provider.Name(), "error", chunk.Err)
			_ = wsjson.Write(ctx, conn, serverMessage{Type: "error", Message: "the AI is unavailable right now"})
			return full.String(), true
		case chunk.Done:
			if err := wsjson.Write(ctx, conn, serverMessage{Type: "done"}); err != nil {
				return full.String(), false
			}
		default:
			full.WriteString(chunk.Delta)
			if err := wsjson.Write(ctx, conn, serverMessage{Type: "delta", Text: chunk.Delta}); err != nil {
				return full.String(), false
			}
		}
	}
	return full.String(), true
}

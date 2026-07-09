package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/store"
)

type moodHandlers struct {
	store *store.Store
}

func (h *moodHandlers) trends(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d >= 7 && d <= 365 {
		days = d
	}
	trends, err := h.store.MoodTrends(r.Context(), auth.UserID(r.Context()), days)
	if err != nil {
		slog.Error("mood trends", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load mood trends")
		return
	}
	writeJSON(w, http.StatusOK, trends)
}

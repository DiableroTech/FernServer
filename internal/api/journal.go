package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/store"
)

type journalHandlers struct {
	store *store.Store
}

func (h *journalHandlers) list(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.store.ListSessions(r.Context(), auth.UserID(r.Context()), 100)
	if err != nil {
		slog.Error("list sessions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load sessions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (h *journalHandlers) get(w http.ResponseWriter, r *http.Request) {
	sess, err := h.store.GetSession(r.Context(), auth.UserID(r.Context()), chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		slog.Error("get session", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load session")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

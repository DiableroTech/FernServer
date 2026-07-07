package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/DiableroTech/fern-server/internal/config"
)

func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"fern"}`))
	})

	// Phase 1: /api/v1/auth (register, login, refresh)
	// Phase 1: /api/v1/chat/ws (WebSocket streaming)
	// Phase 2: /api/v1/journal, /api/v1/mood
	// Phase 3: /api/v1/recovery

	return r
}

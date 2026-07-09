package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/config"
	"github.com/DiableroTech/fern-server/internal/llm"
	"github.com/DiableroTech/fern-server/internal/store"
	"github.com/DiableroTech/fern-server/internal/ws"
)

type Deps struct {
	Config   *config.Config
	Store    *store.Store
	Provider llm.Provider
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: d.Config.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         300,
	}))
	r.Use(rateLimit(20, 60)) // global per-IP ceiling

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "fern"})
	})

	ah := &authHandlers{store: d.Store, jwtSecret: d.Config.JWTSecret}
	jh := &journalHandlers{store: d.Store}
	mh := &moodHandlers{store: d.Store}
	rh := &recoveryHandlers{store: d.Store}
	reph := &reportHandlers{store: d.Store, provider: d.Provider}
	meh := &meHandlers{store: d.Store}
	chat := &ws.ChatHandler{
		Provider:       d.Provider,
		Store:          d.Store,
		JWTSecret:      d.Config.JWTSecret,
		OriginPatterns: wsOriginPatterns(d.Config.AllowedOrigins),
	}

	r.Route("/api/v1", func(r chi.Router) {
		// Tight limit on credential endpoints (brute-force protection).
		r.Group(func(r chi.Router) {
			r.Use(rateLimit(0.5, 10))
			r.Post("/auth/register", ah.register)
			r.Post("/auth/login", ah.login)
			r.Post("/auth/refresh", ah.refresh)
		})

		// WS authenticates via ?token= itself (browsers can't set headers on WS).
		r.Get("/chat/ws", chat.ServeHTTP)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(d.Config.JWTSecret))
			r.Get("/journal", jh.list)
			r.Get("/journal/{id}", jh.get)
			r.Get("/mood/trends", mh.trends)
			r.Get("/recovery", rh.overview)
			r.Put("/recovery/profile", rh.upsertProfile)
			r.Delete("/recovery/profile", rh.deleteProfile)
			r.Post("/recovery/craving", rh.logCraving)
			r.Get("/reports", reph.list)
			r.Post("/reports/weekly", reph.generateWeekly)
			r.Get("/me", meh.get)
			r.Patch("/me", meh.update)
			r.Get("/me/export", meh.export)
			r.Delete("/me", meh.delete)
		})
	})

	return r
}

// wsOriginPatterns converts CORS-style origins (scheme://host) into the
// host patterns coder/websocket matches against. Bare wildcards are dropped
// so a scheme wildcard like "file://*" can't become allow-all.
func wsOriginPatterns(origins []string) []string {
	var out []string
	for _, o := range origins {
		host := o
		if i := strings.Index(o, "://"); i >= 0 {
			host = o[i+3:]
		}
		if host == "" || host == "*" {
			continue
		}
		out = append(out, host)
	}
	if len(out) == 0 {
		out = []string{"localhost:*"}
	}
	return out
}

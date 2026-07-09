package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DiableroTech/fern-server/internal/api"
	"github.com/DiableroTech/fern-server/internal/config"
	"github.com/DiableroTech/fern-server/internal/crypto"
	"github.com/DiableroTech/fern-server/internal/llm"
	"github.com/DiableroTech/fern-server/internal/store"
	"github.com/DiableroTech/fern-server/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	st, err := store.New(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer st.Close()

	if cfg.EncryptionKey != "" {
		enc, err := crypto.New(cfg.EncryptionKey)
		if err != nil {
			return err
		}
		st.Enc = enc
	} else {
		slog.Warn("ENCRYPTION_KEY not set — transcripts stored in plaintext (dev only)")
	}

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 60*time.Second)
	err = st.Migrate(migrateCtx, migrations.FS)
	cancelMigrate()
	if err != nil {
		return err
	}

	apiKey := cfg.OpenAIAPIKey
	if cfg.LLMProvider == "anthropic" {
		apiKey = cfg.AnthropicAPIKey
	}
	if apiKey == "" {
		slog.Warn("no API key set for LLM provider — chat will fail until configured", "provider", cfg.LLMProvider)
	}
	provider, err := llm.New(cfg.LLMProvider, llm.ProviderConfig{APIKey: apiKey, Model: cfg.Model})
	if err != nil {
		return err
	}

	router := api.NewRouter(api.Deps{Config: cfg, Store: st, Provider: provider})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // streaming responses manage their own deadlines
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("fern server listening", "port", cfg.Port, "provider", provider.Name(), "model", cfg.Model)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-stop:
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("fern server stopped")
	return nil
}

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string

	// Base64-encoded 32-byte key for transcript encryption at rest.
	// Empty = plaintext (dev only).
	EncryptionKey string

	// Comma-separated origins allowed for CORS and WebSocket upgrades.
	AllowedOrigins []string

	// LLM provider: "anthropic" or "openai"
	LLMProvider     string
	AnthropicAPIKey string
	OpenAIAPIKey    string
	Model           string
}

func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional; real env vars win in production

	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://fern:fern@localhost:5432/fern?sslmode=disable"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		EncryptionKey:   os.Getenv("ENCRYPTION_KEY"),
		AllowedOrigins:  splitList(getEnv("ALLOWED_ORIGINS", "http://localhost:*,app://*,file://*")),
		LLMProvider:     getEnv("LLM_PROVIDER", "openai"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		Model:           getEnv("LLM_MODEL", "gpt-4o-mini"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

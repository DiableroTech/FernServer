package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string

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

package llm

import (
	"context"
	"fmt"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type StreamChunk struct {
	Delta string
	Done  bool
	Err   error
}

// Provider abstracts a hosted LLM (Anthropic, OpenAI) behind one interface
// so the engine can be swapped via config.
type Provider interface {
	// StreamChat sends the conversation and streams response tokens on the
	// returned channel. The channel closes after a Done or Err chunk.
	StreamChat(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error)

	Name() string
}

type ProviderConfig struct {
	APIKey string
	Model  string
}

func New(provider string, cfg ProviderConfig) (Provider, error) {
	switch provider {
	case "anthropic":
		return newAnthropic(cfg), nil
	case "openai":
		return newOpenAI(cfg), nil
	default:
		return nil, fmt.Errorf("unknown llm provider %q", provider)
	}
}

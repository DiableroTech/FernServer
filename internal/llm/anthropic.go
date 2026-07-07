package llm

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicProvider struct {
	client anthropic.Client
	model  string
}

func newAnthropic(cfg ProviderConfig) Provider {
	return &anthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(cfg.APIKey)),
		model:  cfg.Model,
	}
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) StreamChat(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 2048,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
	}
	for _, m := range messages {
		switch m.Role {
		case RoleUser:
			params.Messages = append(params.Messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case RoleAssistant:
			params.Messages = append(params.Messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		}
	}

	out := make(chan StreamChunk)
	go func() {
		defer close(out)
		stream := p.client.Messages.NewStreaming(ctx, params)
		for stream.Next() {
			event := stream.Current()
			switch ev := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if delta, ok := ev.Delta.AsAny().(anthropic.TextDelta); ok {
					select {
					case out <- StreamChunk{Delta: delta.Text}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			out <- StreamChunk{Err: err}
			return
		}
		out <- StreamChunk{Done: true}
	}()
	return out, nil
}

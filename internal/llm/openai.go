package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openaiProvider struct {
	client openai.Client
	model  string
}

func newOpenAI(cfg ProviderConfig) Provider {
	return &openaiProvider{
		client: openai.NewClient(option.WithAPIKey(cfg.APIKey)),
		model:  cfg.Model,
	}
}

func (p *openaiProvider) Name() string { return "openai" }

func (p *openaiProvider) StreamChat(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error) {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(systemPrompt)},
	}
	for _, m := range messages {
		switch m.Role {
		case RoleUser:
			params.Messages = append(params.Messages, openai.UserMessage(m.Content))
		case RoleAssistant:
			params.Messages = append(params.Messages, openai.AssistantMessage(m.Content))
		}
	}

	out := make(chan StreamChunk)
	go func() {
		defer close(out)
		stream := p.client.Chat.Completions.NewStreaming(ctx, params)
		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) == 0 {
				continue
			}
			if delta := chunk.Choices[0].Delta.Content; delta != "" {
				select {
				case out <- StreamChunk{Delta: delta}:
				case <-ctx.Done():
					return
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

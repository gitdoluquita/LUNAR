package llm

import (
    "context"
    "time"

    openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements Provider using OpenAI-compatible chat completions.
type OpenAIProvider struct {
    client *openai.Client
}

func NewOpenAIProvider(apiKey string, baseURL string) *OpenAIProvider {
    cfg := openai.DefaultConfig(apiKey)
    if baseURL != "" {
        cfg.BaseURL = baseURL
    }
    return &OpenAIProvider{client: openai.NewClientWithConfig(cfg)}
}

func (p *OpenAIProvider) Chat(ctx context.Context, model string, messages []ChatMessage, temperature float32) (string, time.Duration, error) {
    req := openai.ChatCompletionRequest{
        Model:       model,
        Temperature: float32(temperature),
        N:           1,
    }
    for _, m := range messages {
        req.Messages = append(req.Messages, openai.ChatCompletionMessage{Role: m.Role, Content: m.Content})
    }

    start := time.Now()
    resp, err := p.client.CreateChatCompletion(ctx, req)
    dur := time.Since(start)
    if err != nil {
        return "", dur, err
    }
    if len(resp.Choices) == 0 {
        return "", dur, nil
    }
    return resp.Choices[0].Message.Content, dur, nil
}


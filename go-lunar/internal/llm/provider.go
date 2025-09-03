package llm

import (
    "context"
    "time"
)

// ChatMessage represents a single role/content message for chat-based LLMs.
type ChatMessage struct {
    Role    string
    Content string
}

// Provider is an abstract LLM provider interface.
// Different providers (OpenAI-compatible, Anthropic, Azure, local) can implement this.
type Provider interface {
    Chat(ctx context.Context, model string, messages []ChatMessage, temperature float32) (string, time.Duration, error)
}


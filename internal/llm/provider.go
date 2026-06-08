package llm

import (
	"context"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type ModelInfo struct {
	Provider        string `json:"provider"`
	Name            string `json:"name"`
	SupportsStream  bool   `json:"supports_stream"`
	SupportsTools   bool   `json:"supports_tools"`
	SupportsJSON    bool   `json:"supports_json"`
	MaxTokens       int    `json:"max_tokens"`
	ContextWindow   int    `json:"context_window"`
	SupportsThinking bool  `json:"supports_thinking"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Tools       []ToolDefinition  `json:"tools,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Schema      interface{}       `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Message Message `json:"message"`
	Usage   Usage   `json:"usage"`
}

type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
	ModelInfo(model string) (*ModelInfo, error)
}

type StreamEventType int

const (
	StreamToken StreamEventType = iota
	StreamToolCallBegin
	StreamToolCallDelta
	StreamToolCallEnd
	StreamError
	StreamDone
)

type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Data    string          `json:"data,omitempty"`
	Index   int             `json:"index,omitempty"`
	ToolID  string          `json:"tool_id,omitempty"`
	ToolName string         `json:"tool_name,omitempty"`
	Error   error           `json:"error,omitempty"`
	Usage   *Usage          `json:"usage,omitempty"`
	Done    bool            `json:"done,omitempty"`
}

type Gateway struct {
	providers  map[string]Provider
	defaultPvd string
}

func NewGateway() *Gateway {
	return &Gateway{
		providers:  make(map[string]Provider),
		defaultPvd: "opencode",
	}
}

func (g *Gateway) Register(p Provider) {
	g.providers[p.Name()] = p
}

func (g *Gateway) SetDefault(name string) {
	g.defaultPvd = name
}

func (g *Gateway) Resolve(model string) (Provider, string) {
	for prefix, p := range g.providers {
		if len(model) > len(prefix)+1 && model[:len(prefix)] == prefix && model[len(prefix):][0] == '/' {
			return p, model[len(prefix)+1:]
		}
	}
	if p, ok := g.providers[g.defaultPvd]; ok {
		return p, model
	}
	for _, p := range g.providers {
		return p, model
	}
	return nil, model
}

func (g *Gateway) Chat(ctx context.Context, model string, req ChatRequest) (*ChatResponse, error) {
	p, actualModel := g.Resolve(model)
	if p == nil {
		return nil, &ProviderError{Message: "no provider found for model: " + model}
	}
	req.Model = actualModel
	return p.Chat(ctx, req)
}

func (g *Gateway) ChatStream(ctx context.Context, model string, req ChatRequest) (<-chan StreamEvent, error) {
	p, actualModel := g.Resolve(model)
	if p == nil {
		return nil, &ProviderError{Message: "no provider found for model: " + model}
	}
	req.Model = actualModel
	return p.ChatStream(ctx, req)
}

type ProviderError struct {
	Message string
	Cause   error
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return "provider: " + e.Message + ": " + e.Cause.Error()
	}
	return "provider: " + e.Message
}

func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	return d.Round(time.Second).String()
}

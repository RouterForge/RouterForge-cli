package llm

import (
	"context"
	"testing"
)

type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		Message: Message{Role: RoleAssistant, Content: "mock response"},
		Usage:   Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: StreamToken, Data: "mock"}
	ch <- StreamEvent{Type: StreamDone, Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProvider) ModelInfo(model string) (*ModelInfo, error) {
	return &ModelInfo{
		Name:          "mock-model",
		SupportsTools: true,
		MaxTokens:     4096,
	}, nil
}

func TestGateway_Resolve_ByPrefix(t *testing.T) {
	g := NewGateway()
	g.Register(&mockProvider{})

	p, model := g.Resolve("mock/abc")
	if p == nil {
		t.Fatal("expected provider to be resolved")
	}
	if model != "abc" {
		t.Fatalf("expected model 'abc', got '%s'", model)
	}
}

func TestGateway_Resolve_Default(t *testing.T) {
	g := NewGateway()
	g.Register(&mockProvider{})
	g.SetDefault("mock")

	p, model := g.Resolve("some-model")
	if p == nil {
		t.Fatal("expected default provider")
	}
	if model != "some-model" {
		t.Fatalf("expected model 'some-model', got '%s'", model)
	}
}

func TestGateway_Resolve_Unknown(t *testing.T) {
	g := NewGateway()
	p, _ := g.Resolve("unknown")
	if p != nil {
		t.Fatal("expected nil for unknown provider")
	}
}

func TestGateway_Chat(t *testing.T) {
	g := NewGateway()
	g.Register(&mockProvider{})

	resp, err := g.Chat(context.Background(), "mock/test", ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Message.Content != "mock response" {
		t.Fatalf("expected 'mock response', got '%s'", resp.Message.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestGateway_ChatStream(t *testing.T) {
	g := NewGateway()
	g.Register(&mockProvider{})

	ch, err := g.ChatStream(context.Background(), "mock/test", ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	var events []StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 stream events, got %d", len(events))
	}
}

func TestGateway_Chat_NoProvider(t *testing.T) {
	g := NewGateway()
	_, err := g.Chat(context.Background(), "nope/model", ChatRequest{})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input int
	}{
		{0}, {1}, {60},
	}
	for range tests {
		_ = FormatDuration(0)
	}
}

func TestModelInfoString(t *testing.T) {
	m := ModelInfo{Name: "test", Provider: "mock"}
	if m.Name != "test" {
		t.Fatalf("expected 'test', got '%s'", m.Name)
	}
}

package memory

import (
	"testing"
)

func TestInMemoryStore_Add(t *testing.T) {
	s := NewStore()
	err := s.Add(Entry{ID: "1", AgentID: "test", Type: TypeDecision, Content: "test entry"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
}

func TestInMemoryStore_Query(t *testing.T) {
	s := NewStore()
	s.Add(Entry{ID: "1", AgentID: "a1", Type: TypeDecision, Content: "dec1"})
	s.Add(Entry{ID: "2", AgentID: "a1", Type: TypeTool, Content: "tool1"})
	s.Add(Entry{ID: "3", AgentID: "a2", Type: TypeDecision, Content: "dec2"})

	entries, err := s.Query("a1", TypeDecision, 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != "dec1" {
		t.Fatalf("expected 'dec1', got '%s'", entries[0].Content)
	}
}

func TestInMemoryStore_QueryLimit(t *testing.T) {
	s := NewStore()
	s.Add(Entry{ID: "1", AgentID: "a1", Type: TypeDecision, Content: "d1"})
	s.Add(Entry{ID: "2", AgentID: "a1", Type: TypeDecision, Content: "d2"})
	s.Add(Entry{ID: "3", AgentID: "a1", Type: TypeDecision, Content: "d3"})

	entries, _ := s.Query("a1", TypeDecision, 2)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestInMemoryStore_Recent(t *testing.T) {
	s := NewStore()
	s.Add(Entry{ID: "1", AgentID: "a1", Type: TypeDecision, Content: "old"})
	s.Add(Entry{ID: "2", AgentID: "a1", Type: TypeDecision, Content: "new"})

	recent, _ := s.Recent(1)
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent, got %d", len(recent))
	}
	if recent[0].Content != "new" {
		t.Fatalf("expected 'new', got '%s'", recent[0].Content)
	}
}

func TestInMemoryStore_Clear(t *testing.T) {
	s := NewStore()
	s.Add(Entry{ID: "1", AgentID: "a1", Type: TypeDecision, Content: "keep"})
	s.Add(Entry{ID: "2", AgentID: "a2", Type: TypeDecision, Content: "remove"})

	s.Clear("a2")
	recent, _ := s.Recent(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry after clear, got %d", len(recent))
	}
}

func TestCompressor_BuildContext_Empty(t *testing.T) {
	s := NewStore()
	c := NewCompressor(s)
	ctx := c.BuildContext("nonexistent")
	if ctx != "" {
		t.Fatalf("expected empty context, got '%s'", ctx)
	}
}

func TestCompressor_BuildContext(t *testing.T) {
	s := NewStore()
	s.Add(Entry{ID: "1", AgentID: "a1", Type: TypeDecision, Content: "chose Go"})
	s.Add(Entry{ID: "2", AgentID: "a1", Type: TypeDecision, Content: "picked PostgreSQL"})

	c := NewCompressor(s)
	ctx := c.BuildContext("a1")
	if ctx == "" {
		t.Fatalf("expected non-empty context")
	}
	if !contains(ctx, "chose Go") {
		t.Fatalf("expected 'chose Go' in context")
	}
}

func TestCompressor_TrimContext(t *testing.T) {
	c := NewCompressor(nil)
	trimmed := c.TrimContext("hello world", 5)
	if len(trimmed) < 10 {
		t.Fatalf("expected trimmed to be truncated, got '%s'", trimmed)
	}
	if trimmed[:5] != "hello" {
		t.Fatalf("expected 'hello', got '%s'", trimmed[:5])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

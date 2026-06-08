package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	r.Register(sk("test_tool", "a test tool",
		json.RawMessage(`{"type":"object"}`),
		func(ctx context.Context, p json.RawMessage) (string, error) {
			return "ok", nil
		},
	))

	if _, ok := r.Get("test_tool"); !ok {
		t.Fatal("expected tool to be registered")
	}
}

func TestRegistry_Get_Unknown(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("unknown"); ok {
		t.Fatal("expected unknown tool to not be found")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(sk("a", "", nil, func(context.Context, json.RawMessage) (string, error) {
		return "", nil
	}))
	r.Register(sk("b", "", nil, func(context.Context, json.RawMessage) (string, error) {
		return "", nil
	}))

	tools := r.List()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestRegistry_Definitions(t *testing.T) {
	r := NewRegistry()
	r.Register(sk("my_tool", "does things",
		json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`),
		nil,
	))

	defs := r.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
}

func TestPermissionEvaluator_DefaultAllow(t *testing.T) {
	p := NewPermissionEvaluator()
	if !p.CanExecute("anything") {
		t.Fatal("expected default allow")
	}
}

func TestPermissionEvaluator_Deny(t *testing.T) {
	p := NewPermissionEvaluator()
	p.AddRule(PermissionRule{ToolPattern: "run_command", Effect: "deny"})
	if p.CanExecute("run_command") {
		t.Fatal("expected deny")
	}
	if !p.CanExecute("other_tool") {
		t.Fatal("expected other tools still allowed")
	}
}

func TestPermissionEvaluator_Evaluate(t *testing.T) {
	p := NewPermissionEvaluator()
	p.AddRule(PermissionRule{ToolPattern: "dangerous", Effect: "ask"})
	got := p.Evaluate("dangerous")
	if got != "ask" {
		t.Fatalf("expected 'ask', got '%s'", got)
	}
}

func TestRegisteredTool_Execute(t *testing.T) {
	tool := sk("echo", "echoes",
		json.RawMessage(`{"type":"object"}`),
		func(ctx context.Context, p json.RawMessage) (string, error) {
			return "echo", nil
		},
	)

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "echo" {
		t.Fatalf("expected 'echo', got '%s'", result)
	}
}

func TestRegisterAll(t *testing.T) {
	r := NewRegistry()
	RegisterAll(r)

	expected := []string{
		"run_command",
		"read_file",
		"write_file",
		"search_code",
		"glob_files",
		"web_fetch",
		"ask_user",
	}

	for _, name := range expected {
		if _, ok := r.Get(name); !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
}

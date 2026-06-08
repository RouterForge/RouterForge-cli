package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/routerforge/cli/internal/agent"
	"github.com/routerforge/cli/internal/memory"
	"github.com/routerforge/cli/pkg/models"
)

func TestNewHeadManager(t *testing.T) {
	hm := NewHeadManager("test-model")
	if hm == nil {
		t.Fatal("expected non-nil head manager")
	}
	if hm.model != "test-model" {
		t.Fatalf("expected 'test-model', got '%s'", hm.model)
	}
	if hm.Project().Phase != models.PhaseIdle {
		t.Fatalf("expected PhaseIdle, got %v", hm.Project().Phase)
	}
}

func TestHeadManager_SetUserProxy(t *testing.T) {
	hm := NewHeadManager("test")
	hm.SetUserProxy(agent.NewTerminalUserProxy())
	if hm.userProxy == nil {
		t.Fatal("expected user proxy to be set")
	}
}

func TestHeadManager_SetBus(t *testing.T) {
	hm := NewHeadManager("test")
	hm.SetBus(hm.Bus())
	if hm.bus == nil {
		t.Fatal("expected bus to be set")
	}
}

func TestHeadManager_SetMemory(t *testing.T) {
	hm := NewHeadManager("test")
	hm.SetMemory(memory.NewStore())
	if hm.mem == nil {
		t.Fatal("expected memory to be set")
	}
}

func TestHeadManager_CreateTeam(t *testing.T) {
	hm := NewHeadManager("test")
	tm, err := hm.CreateTeam("frontend")
	if err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if tm == nil {
		t.Fatal("expected non-nil team manager")
	}
	if tm.Role() != "frontend_lead" {
		t.Fatalf("expected 'frontend_lead', got '%s'", tm.Role())
	}
	if len(hm.Teams()) != 1 {
		t.Fatalf("expected 1 team, got %d", len(hm.Teams()))
	}
}

func TestHeadManager_SendMessage(t *testing.T) {
	hm := NewHeadManager("test")
	hm.SendMessage("from", "to", "broadcast", "hello")
	if len(hm.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(hm.messages))
	}
}

func TestMicroAgent(t *testing.T) {
	agent := &models.Agent{
		ID:     "test-agent",
		Role:   "tester",
		Model:  "test-model",
		Status: models.StatusCreated,
		Tasks: []models.Task{
			{ID: "t1", Description: "test task", Status: models.TaskPending},
		},
	}
	ctx := &Context{
		Project: &models.Project{ID: "proj1"},
		Model:   "test-model",
		Data:    make(map[string]string),
		Memory:  memory.NewStore(),
	}

	ma := NewMicroAgent(agent, ctx)
	if ma == nil {
		t.Fatal("expected non-nil micro agent")
	}
	if ma.ID() != "test-agent" {
		t.Fatalf("expected 'test-agent', got '%s'", ma.ID())
	}
	if ma.Role() != "tester" {
		t.Fatalf("expected 'tester', got '%s'", ma.Role())
	}

	ma.SetMemory(memory.NewStore())
	ma.Checkpoint()

	_ = context.Background()
	artifactsDir := filepath.Join(".", ".routerforge-test-artifacts")
	os.MkdirAll(artifactsDir, 0755)
	defer os.RemoveAll(artifactsDir)
}

func TestStateMachine(t *testing.T) {
	sm := NewStateMachine()
	if sm.Current() != PhaseIdle {
		t.Fatalf("expected Idle, got %s", sm.Current())
	}

	if err := sm.Transition(PhaseUnderstand, "start"); err != nil {
		t.Fatalf("Transition failed: %v", err)
	}
	if sm.Current() != PhaseUnderstand {
		t.Fatalf("expected Understand, got %s", sm.Current())
	}

	if sm.CanTransition(PhaseDesign) != true {
		t.Fatal("expected CanTransition(Design) to be true")
	}

	if sm.CanTransition(PhaseIdle) != false {
		t.Fatal("expected CanTransition(Idle) to be false (reverse)")
	}

	sm.Transition(PhaseDesign, "next")
	sm.Transition(PhaseExecute, "next")
	sm.Transition(PhaseReview, "final")

	if len(sm.History()) != 4 {
		t.Fatalf("expected 4 transitions, got %d", len(sm.History()))
	}
}

func TestPhaseStrings(t *testing.T) {
	tests := []struct {
		p    Phase
		want string
	}{
		{PhaseIdle, "Idle"},
		{PhaseUnderstand, "Understand"},
		{PhaseDesign, "Design"},
		{PhaseExecute, "Execute"},
		{PhaseReview, "Review"},
		{Phase(99), "Unknown(99)"},
	}
	for _, tt := range tests {
		got := tt.p.String()
		if got != tt.want {
			t.Errorf("Phase(%d).String() = '%s', want '%s'", tt.p, got, tt.want)
		}
	}
}

func TestPhaseToModelMapping(t *testing.T) {
	if PhaseIdle.ToModel() != models.PhaseIdle {
		t.Fatal("PhaseIdle mapping wrong")
	}
	if PhaseUnderstand.ToModel() != models.PhaseUnderstand {
		t.Fatal("PhaseUnderstand mapping wrong")
	}
	if PhaseDesign.ToModel() != models.PhaseDesign {
		t.Fatal("PhaseDesign mapping wrong")
	}
}

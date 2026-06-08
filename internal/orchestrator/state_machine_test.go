package orchestrator

import (
	"testing"
)

func TestStateMachineInitialState(t *testing.T) {
	sm := NewStateMachine()
	if sm.Current() != PhaseIdle {
		t.Errorf("Expected idle, got %s", sm.Current())
	}
}

func TestStateMachineValidTransitions(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		from Phase
		to   Phase
		want bool
	}{
		{PhaseIdle, PhaseUnderstand, true},
		{PhaseUnderstand, PhaseDesign, true},
		{PhaseDesign, PhaseExecute, true},
		{PhaseExecute, PhaseReview, true},
		{PhaseIdle, PhaseDesign, true},
		{PhaseIdle, PhaseExecute, true},
		{PhaseDesign, PhaseIdle, false},
		{PhaseReview, PhaseIdle, false},
	}

	for _, tt := range tests {
		sm.current = tt.from
		got := sm.CanTransition(tt.to)
		if got != tt.want {
			t.Errorf("CanTransition(%s -> %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestStateMachineTransition(t *testing.T) {
	sm := NewStateMachine()

	err := sm.Transition(PhaseUnderstand, "test")
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}
	if sm.Current() != PhaseUnderstand {
		t.Errorf("Expected Understand, got %s", sm.Current())
	}

	err = sm.Transition(PhaseDesign, "design phase")
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}

	if len(sm.History()) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(sm.History()))
	}
}

func TestStateMachineInvalidTransition(t *testing.T) {
	sm := NewStateMachine()
	sm.current = PhaseExecute
	err := sm.Transition(PhaseIdle, "cannot go backwards to idle")
	if err == nil {
		t.Error("Expected error for backward transition to Idle")
	}
}

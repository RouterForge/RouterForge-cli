package orchestrator

import (
	"testing"
)

func TestRuntimeFlowInitialState(t *testing.T) {
	f := NewRuntimeFlow()
	if f.Current() != RuntimeIdle {
		t.Errorf("Expected idle, got %s", f.Current())
	}
}

func TestRuntimeFlowValidTransitions(t *testing.T) {
	f := NewRuntimeFlow()

	tests := []struct {
		from RuntimePhase
		to   RuntimePhase
		want bool
	}{
		{RuntimeIdle, RuntimeUnderstand, true},
		{RuntimeUnderstand, RuntimeDesign, true},
		{RuntimeDesign, RuntimeExecute, true},
		{RuntimeExecute, RuntimeRepair, true},
		{RuntimeRepair, RuntimeReview, true},
		{RuntimeReview, RuntimeExecute, true},
		{RuntimeRepair, RuntimeExecute, true},
		{RuntimeExecute, RuntimeUnderstand, true},
		{RuntimeIdle, RuntimeDesign, true},
		{RuntimeIdle, RuntimeExecute, true},
		{RuntimeDesign, RuntimeIdle, false},
		{RuntimeReview, RuntimeIdle, false},
		{RuntimeReview, RuntimeRepair, false},
	}

	for _, tt := range tests {
		f.current = tt.from
		got := f.CanTransition(tt.to)
		if got != tt.want {
			t.Errorf("CanTransition(%s -> %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestRuntimeFlowTransition(t *testing.T) {
	f := NewRuntimeFlow()

	err := f.Transition(RuntimeUnderstand, "test")
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}
	if f.Current() != RuntimeUnderstand {
		t.Errorf("Expected Understand, got %s", f.Current())
	}

	err = f.Transition(RuntimeDesign, "design phase")
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}

	if len(f.History()) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(f.History()))
	}
}

func TestRuntimeFlowInvalidTransition(t *testing.T) {
	f := NewRuntimeFlow()
	f.current = RuntimeExecute
	err := f.Transition(RuntimeIdle, "cannot go backwards to idle")
	if err == nil {
		t.Error("Expected error for backward transition to Idle")
	}
}

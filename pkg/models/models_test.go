package models

import (
	"testing"
)

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase Phase
		want  string
	}{
		{PhaseIdle, "Idle"},
		{PhaseUnderstand, "Understand"},
		{PhaseDesign, "Design"},
		{PhaseExecute, "Execute"},
		{PhaseReview, "Review"},
	}
	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.want {
			t.Errorf("Phase(%d).String() = %q, want %q", tt.phase, got, tt.want)
		}
	}
}

func TestAgentStatus(t *testing.T) {
	if StatusCreated != "created" {
		t.Errorf("StatusCreated = %q, want %q", StatusCreated, "created")
	}
	if StatusActive != "active" {
		t.Errorf("StatusActive = %q, want %q", StatusActive, "active")
	}
	if StatusCompleted != "completed" {
		t.Errorf("StatusCompleted = %q, want %q", StatusCompleted, "completed")
	}
}

func TestTaskStatus(t *testing.T) {
	if TaskPending != "pending" {
		t.Errorf("TaskPending = %q, want %q", TaskPending, "pending")
	}
	if TaskCompleted != "completed" {
		t.Errorf("TaskCompleted = %q, want %q", TaskCompleted, "completed")
	}
	if TaskFailed != "failed" {
		t.Errorf("TaskFailed = %q, want %q", TaskFailed, "failed")
	}
}

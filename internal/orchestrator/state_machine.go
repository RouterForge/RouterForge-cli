package orchestrator

import (
	"fmt"
	"time"

	"github.com/routerforge/cli/pkg/models"
)

type StateMachine struct {
	current Phase
	history []models.PhaseTransition
}

type Phase int

const (
	PhaseIdle       Phase = 0
	PhaseUnderstand Phase = 1
	PhaseDesign     Phase = 2
	PhaseExecute    Phase = 3
	PhaseReview     Phase = 4
)

func (p Phase) String() string {
	switch p {
	case PhaseIdle:
		return "Idle"
	case PhaseUnderstand:
		return "Understand"
	case PhaseDesign:
		return "Design"
	case PhaseExecute:
		return "Execute"
	case PhaseReview:
		return "Review"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}

func (p Phase) ToModel() models.Phase {
	switch p {
	case PhaseIdle:
		return models.PhaseIdle
	case PhaseUnderstand:
		return models.PhaseUnderstand
	case PhaseDesign:
		return models.PhaseDesign
	case PhaseExecute:
		return models.PhaseExecute
	case PhaseReview:
		return models.PhaseReview
	default:
		return models.PhaseIdle
	}
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		current: PhaseIdle,
		history: []models.PhaseTransition{},
	}
}

func (sm *StateMachine) Current() Phase {
	return sm.current
}

func (sm *StateMachine) CurrentStr() string {
	return sm.current.String()
}

func (sm *StateMachine) CanTransition(to Phase) bool {
	order := []Phase{PhaseIdle, PhaseUnderstand, PhaseDesign, PhaseExecute, PhaseReview}
	fromIdx := -1
	toIdx := -1
	for i, p := range order {
		if p == sm.current {
			fromIdx = i
		}
		if p == to {
			toIdx = i
		}
	}
	if fromIdx == -1 || toIdx == -1 {
		return false
	}
	// Allow any forward jump (not just +1) or back to Understand for re-iteration
	if toIdx >= fromIdx {
		return true
	}
	if to == PhaseUnderstand {
		return true
	}
	return false
}

func (sm *StateMachine) Transition(to Phase, reason string) error {
	if !sm.CanTransition(to) {
		return fmt.Errorf("cannot transition from %s to %s", sm.current, to)
	}
	from := sm.current
	sm.current = to
	sm.history = append(sm.history, models.PhaseTransition{
		From:      from.ToModel(),
		To:        to.ToModel(),
		Reason:    reason,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (sm *StateMachine) History() []models.PhaseTransition {
	return sm.history
}

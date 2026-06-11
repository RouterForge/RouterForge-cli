package orchestrator

import (
	"fmt"
	"time"

	"github.com/routerforge/cli/pkg/models"
)

// RuntimeFlow represents the runtime flow of the lifecycle:
// Idle → Understand → Design → Execute → Repair → Review
type RuntimeFlow struct {
	current RuntimePhase
	history []models.PhaseTransition
}

type RuntimePhase int

const (
	RuntimeIdle       RuntimePhase = 0
	RuntimeUnderstand RuntimePhase = 1
	RuntimeDesign     RuntimePhase = 2
	RuntimeExecute    RuntimePhase = 3
	RuntimeRepair     RuntimePhase = 4
	RuntimeReview     RuntimePhase = 5
)

func (p RuntimePhase) String() string {
	switch p {
	case RuntimeIdle:
		return "Idle"
	case RuntimeUnderstand:
		return "Understand"
	case RuntimeDesign:
		return "Design"
	case RuntimeExecute:
		return "Execute"
	case RuntimeRepair:
		return "Repair"
	case RuntimeReview:
		return "Review"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}

func (p RuntimePhase) ToModel() models.Phase {
	switch p {
	case RuntimeIdle:
		return models.PhaseIdle
	case RuntimeUnderstand:
		return models.PhaseUnderstand
	case RuntimeDesign:
		return models.PhaseDesign
	case RuntimeExecute:
		return models.PhaseExecute
	case RuntimeRepair:
		return models.PhaseRepair
	case RuntimeReview:
		return models.PhaseReview
	default:
		return models.PhaseIdle
	}
}

func NewRuntimeFlow() *RuntimeFlow {
	return &RuntimeFlow{
		current: RuntimeIdle,
		history: []models.PhaseTransition{},
	}
}

func (f *RuntimeFlow) Current() RuntimePhase {
	return f.current
}

func (f *RuntimeFlow) CurrentStr() string {
	return f.current.String()
}

func (f *RuntimeFlow) CanTransition(to RuntimePhase) bool {
	order := []RuntimePhase{RuntimeIdle, RuntimeUnderstand, RuntimeDesign, RuntimeExecute, RuntimeRepair, RuntimeReview}
	fromIdx := -1
	toIdx := -1
	for i, p := range order {
		if p == f.current {
			fromIdx = i
		}
		if p == to {
			toIdx = i
		}
	}
	if fromIdx == -1 || toIdx == -1 {
		return false
	}
	// Allow any forward jump
	if toIdx > fromIdx {
		return true
	}
	// Allow going back to Understand (re-iteration at design level)
	if to == RuntimeUnderstand {
		return true
	}
	// Allow going back to Execute from Repair or Review (re-execution)
	if to == RuntimeExecute && (f.current == RuntimeRepair || f.current == RuntimeReview) {
		return true
	}
	return false
}

func (f *RuntimeFlow) Transition(to RuntimePhase, reason string) error {
	if !f.CanTransition(to) {
		return fmt.Errorf("cannot transition from %s to %s", f.current, to)
	}
	from := f.current
	f.current = to
	f.history = append(f.history, models.PhaseTransition{
		From:      from.ToModel(),
		To:        to.ToModel(),
		Reason:    reason,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (f *RuntimeFlow) History() []models.PhaseTransition {
	return f.history
}

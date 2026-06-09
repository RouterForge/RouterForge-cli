package orchestrator

import (
	"fmt"
	"time"

	"github.com/routerforge/cli/pkg/models"
)

type LifecycleStateMachine struct {
	current LifecyclePhase
	history []models.LifecycleTransition
}

type LifecyclePhase int

const (
	LifecycleDemo       LifecyclePhase = 0
	LifecyclePrototype  LifecyclePhase = 1
	LifecycleMVP        LifecyclePhase = 2
	LifecycleProduction LifecyclePhase = 3
)

func (p LifecyclePhase) String() string {
	switch p {
	case LifecycleDemo:
		return "Demo"
	case LifecyclePrototype:
		return "Prototype"
	case LifecycleMVP:
		return "MVP"
	case LifecycleProduction:
		return "Production"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}

func (p LifecyclePhase) ToModel() models.LifecyclePhase {
	switch p {
	case LifecycleDemo:
		return models.LifecycleDemo
	case LifecyclePrototype:
		return models.LifecyclePrototype
	case LifecycleMVP:
		return models.LifecycleMVP
	case LifecycleProduction:
		return models.LifecycleProduction
	default:
		return models.LifecycleDemo
	}
}

func NewLifecycleStateMachine() *LifecycleStateMachine {
	return &LifecycleStateMachine{
		current: LifecycleDemo,
	}
}

func (lsm *LifecycleStateMachine) Current() LifecyclePhase {
	return lsm.current
}

func (lsm *LifecycleStateMachine) CurrentStr() string {
	return lsm.current.String()
}

func (lsm *LifecycleStateMachine) CanTransition(to LifecyclePhase) bool {
	order := []LifecyclePhase{LifecycleDemo, LifecyclePrototype, LifecycleMVP, LifecycleProduction}
	fromIdx := -1
	toIdx := -1
	for i, p := range order {
		if p == lsm.current {
			fromIdx = i
		}
		if p == to {
			toIdx = i
		}
	}
	if fromIdx == -1 || toIdx == -1 {
		return false
	}
	return toIdx > fromIdx
}

func (lsm *LifecycleStateMachine) Transition(to LifecyclePhase, reason string, approvals []string) error {
	if !lsm.CanTransition(to) {
		return fmt.Errorf("cannot transition from %s to %s", lsm.current, to)
	}
	from := lsm.current
	lsm.current = to
	entry := models.LifecycleTransition{
		From:      from.ToModel(),
		To:        to.ToModel(),
		Reason:    reason,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	lsm.history = append(lsm.history, entry)
	return nil
}

func (lsm *LifecycleStateMachine) History() []models.LifecycleTransition {
	return lsm.history
}

package orchestrator

import (
	"fmt"
	"time"

	"github.com/routerforge/cli/pkg/models"
)

// MaturityStateMachine tracks the project maturity stage:
// Demo → Prototype → MVP → Production
type MaturityStateMachine struct {
	current MaturityStage
	history []models.LifecycleTransition
}

type MaturityStage int

const (
	MaturityDemo       MaturityStage = 0
	MaturityPrototype  MaturityStage = 1
	MaturityMVP        MaturityStage = 2
	MaturityProduction MaturityStage = 3
)

func (s MaturityStage) String() string {
	switch s {
	case MaturityDemo:
		return "Demo"
	case MaturityPrototype:
		return "Prototype"
	case MaturityMVP:
		return "MVP"
	case MaturityProduction:
		return "Production"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

func (s MaturityStage) ToModel() models.LifecyclePhase {
	switch s {
	case MaturityDemo:
		return models.LifecycleDemo
	case MaturityPrototype:
		return models.LifecyclePrototype
	case MaturityMVP:
		return models.LifecycleMVP
	case MaturityProduction:
		return models.LifecycleProduction
	default:
		return models.LifecycleDemo
	}
}

func NewMaturityStateMachine() *MaturityStateMachine {
	return &MaturityStateMachine{
		current: MaturityDemo,
	}
}

func (msm *MaturityStateMachine) Current() MaturityStage {
	return msm.current
}

func (msm *MaturityStateMachine) CurrentStr() string {
	return msm.current.String()
}

func (msm *MaturityStateMachine) CanTransition(to MaturityStage) bool {
	order := []MaturityStage{MaturityDemo, MaturityPrototype, MaturityMVP, MaturityProduction}
	fromIdx := -1
	toIdx := -1
	for i, s := range order {
		if s == msm.current {
			fromIdx = i
		}
		if s == to {
			toIdx = i
		}
	}
	if fromIdx == -1 || toIdx == -1 {
		return false
	}
	return toIdx > fromIdx
}

func (msm *MaturityStateMachine) Transition(to MaturityStage, reason string, approvals []string) error {
	if !msm.CanTransition(to) {
		return fmt.Errorf("cannot transition from %s to %s", msm.current, to)
	}
	from := msm.current
	msm.current = to
	entry := models.LifecycleTransition{
		From:      from.ToModel(),
		To:        to.ToModel(),
		Reason:    reason,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	msm.history = append(msm.history, entry)
	return nil
}

func (msm *MaturityStateMachine) History() []models.LifecycleTransition {
	return msm.history
}

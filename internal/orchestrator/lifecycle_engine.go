package orchestrator

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/pkg/models"
)

// LifecycleEngine unifies the Runtime Flow and Project Maturity into one
// coherent lifecycle. It owns both layers and synchronizes them:
//
// Runtime Flow:  Idle → Understand → Design → Execute → Repair → Review
// Project Maturity: Demo → Prototype → MVP → Production
//
// After each Review, the engine evaluates whether gates pass to advance maturity.
type LifecycleEngine struct {
	flow     *RuntimeFlow
	maturity *MaturityStateMachine
	gates    *ReviewGateManager
	hm       *HeadManager
}

func NewLifecycleEngine(hm *HeadManager) *LifecycleEngine {
	return &LifecycleEngine{
		flow:     NewRuntimeFlow(),
		maturity: NewMaturityStateMachine(),
		gates:    NewReviewGateManager(),
		hm:       hm,
	}
}

// --- Runtime Flow delegation ---

func (le *LifecycleEngine) FlowPhase() RuntimePhase     { return le.flow.Current() }
func (le *LifecycleEngine) FlowStr() string              { return le.flow.CurrentStr() }
func (le *LifecycleEngine) FlowHistory() []models.PhaseTransition { return le.flow.History() }
func (le *LifecycleEngine) CanFlowTransition(to RuntimePhase) bool { return le.flow.CanTransition(to) }

func (le *LifecycleEngine) TransitionFlow(to RuntimePhase, reason string) error {
	return le.flow.Transition(to, reason)
}

// --- Project Maturity delegation ---

func (le *LifecycleEngine) MaturityStage() MaturityStage          { return le.maturity.Current() }
func (le *LifecycleEngine) MaturityStr() string                    { return le.maturity.CurrentStr() }
func (le *LifecycleEngine) MaturityHistory() []models.LifecycleTransition { return le.maturity.History() }
func (le *LifecycleEngine) CanAdvanceMaturity() bool               { return le.gates.AllRequiredPassed() }

func (le *LifecycleEngine) AdvanceMaturity() error {
	if !le.CanAdvanceMaturity() {
		failed := le.gates.GetFailedRequired()
		return fmt.Errorf("cannot advance maturity: %d required gates not passed: %v", len(failed), failed)
	}
	next := le.maturity.Current() + 1
	if next > MaturityProduction {
		return fmt.Errorf("already at final maturity stage")
	}
	approvals := []string{"lifecycle_engine"}
	err := le.maturity.Transition(next, "Advancing project maturity", approvals)
	if err != nil {
		return err
	}
	le.hm.project.LifecyclePhase = next.ToModel()
	le.hm.project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	pterm.Success.Printfln("Project maturity advanced to: %s", le.maturity.CurrentStr())
	return nil
}

// --- Review Gates ---

func (le *LifecycleEngine) Gates() *ReviewGateManager { return le.gates }

func (le *LifecycleEngine) ApproveGate(gateType GateType, approvedBy, notes string) {
	le.gates.SetGatePassed(gateType, approvedBy, notes)
}

// --- Synchronized lifecycle flow ---

// Run executes the full lifecycle: Understand → Design → Execute → Repair → Review
// After Review, evaluates if gates pass to advance maturity.
func (le *LifecycleEngine) Run() error {
	pterm.DefaultSection.Printfln("Project Lifecycle — Starting Runtime Flow")
	pterm.Println()

	if err := le.transitionAndRun(RuntimeUnderstand, "Starting understand phase", le.hm.Understand); err != nil {
		return err
	}
	if err := le.transitionAndRun(RuntimeDesign, "Starting design phase", le.hm.Design); err != nil {
		return err
	}
	if err := le.transitionAndRun(RuntimeExecute, "Starting execute phase", le.hm.Execute); err != nil {
		return err
	}
	if err := le.transitionAndRun(RuntimeRepair, "Starting repair phase", func() error {
		return le.hm.RepairUntilValid(2)
	}); err != nil {
		return err
	}
	if err := le.transitionAndRun(RuntimeReview, "Starting review phase", le.hm.Review); err != nil {
		return err
	}

	// After review: evaluate maturity advancement
	le.evaluateMaturityAdvancement()

	pterm.DefaultSection.Printfln("Lifecycle Runtime Flow Complete")
	return nil
}

// RunFlowTo runs the runtime flow up to and including the given phase.
func (le *LifecycleEngine) RunFlowTo(target RuntimePhase) error {
	flow := []struct {
		phase RuntimePhase
		name  string
		run   func() error
	}{
		{RuntimeUnderstand, "Understand", le.hm.Understand},
		{RuntimeDesign, "Design", le.hm.Design},
		{RuntimeExecute, "Execute", le.hm.Execute},
		{RuntimeRepair, "Repair", func() error { return le.hm.RepairUntilValid(2) }},
		{RuntimeReview, "Review", le.hm.Review},
	}

	for _, step := range flow {
		if step.phase > target {
			break
		}
		if err := le.transitionAndRun(step.phase, "Starting "+step.name+" phase", step.run); err != nil {
			return err
		}
	}
	return nil
}

func (le *LifecycleEngine) transitionAndRun(to RuntimePhase, reason string, fn func() error) error {
	if le.flow.Current() != to {
		if err := le.flow.Transition(to, reason); err != nil {
			return fmt.Errorf("flow transition to %s failed: %w", to, err)
		}
	}
	return fn()
}

// evaluateMaturityAdvancement checks if review gates pass and advances maturity if so.
// If gates don't pass, it logs what failed and suggests re-entering execution.
func (le *LifecycleEngine) evaluateMaturityAdvancement() {
	if le.gates.AllRequiredPassed() {
		pterm.Success.Println("All review gates passed. Evaluating maturity advancement...")
		if le.maturity.Current() < MaturityProduction {
			if err := le.AdvanceMaturity(); err != nil {
				pterm.Warning.Printfln("Could not advance maturity: %v", err)
			}
		} else {
			pterm.Info.Println("Already at maximum maturity stage (Production).")
		}
	} else {
		failed := le.gates.GetFailedRequired()
		pterm.Warning.Printfln("Review gates not all passed — maturity advancement pending.")
		for _, g := range failed {
			pterm.Printfln("  ❌ %s (required)", g.Name)
		}
		pterm.Info.Println("Use 'routerforge gate approve <type>' to pass gates, then advance maturity with 'routerforge lifecycle advance'.")
		pterm.Info.Println("Or re-enter execution with 'routerforge lifecycle run execute' to fix issues.")
	}
}

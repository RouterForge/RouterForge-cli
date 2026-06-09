package orchestrator

import (
	"fmt"
	"sync"
	"time"
)

type GateType string

const (
	GateArchitectureReview GateType = "architecture_review"
	GateSecurityReview     GateType = "security_review"
	GateTestingRequirement GateType = "testing_requirement"
	GateDocumentation      GateType = "documentation"
	GateApproval           GateType = "approval_workflow"
)

type ReviewGate struct {
	Type        GateType `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Passed      bool     `json:"passed"`
	ApprovedBy  string   `json:"approved_by,omitempty"`
	ApprovedAt  string   `json:"approved_at,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

type ReviewGateManager struct {
	mu    sync.Mutex
	gates []ReviewGate
}

func NewReviewGateManager() *ReviewGateManager {
	return &ReviewGateManager{
		gates: defaultGates(),
	}
}

func defaultGates() []ReviewGate {
	return []ReviewGate{
		{
			Type:        GateArchitectureReview,
			Name:        "Architecture Review",
			Description: "Architecture must be reviewed and approved before proceeding",
			Required:    true,
		},
		{
			Type:        GateSecurityReview,
			Name:        "Security Review",
			Description: "Security implications must be reviewed and addressed",
			Required:    true,
		},
		{
			Type:        GateTestingRequirement,
			Name:        "Testing Requirements",
			Description: "All tests must pass before phase transition",
			Required:    true,
		},
		{
			Type:        GateDocumentation,
			Name:        "Documentation",
			Description: "Documentation must be complete and up to date",
			Required:    false,
		},
		{
			Type:        GateApproval,
			Name:        "Phase Approval",
			Description: "Phase transition requires explicit approval",
			Required:    true,
		},
	}
}

func (gm *ReviewGateManager) SetGatePassed(gateType GateType, approvedBy, notes string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	for i, g := range gm.gates {
		if g.Type == gateType {
			gm.gates[i].Passed = true
			gm.gates[i].ApprovedBy = approvedBy
			gm.gates[i].ApprovedAt = time.Now().UTC().Format(time.RFC3339)
			gm.gates[i].Notes = notes
			return
		}
	}
}

func (gm *ReviewGateManager) AllRequiredPassed() bool {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	for _, g := range gm.gates {
		if g.Required && !g.Passed {
			return false
		}
	}
	return true
}

func (gm *ReviewGateManager) GetFailedRequired() []ReviewGate {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	var failed []ReviewGate
	for _, g := range gm.gates {
		if g.Required && !g.Passed {
			failed = append(failed, g)
		}
	}
	return failed
}

func (gm *ReviewGateManager) AllGates() []ReviewGate {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	out := make([]ReviewGate, len(gm.gates))
	copy(out, gm.gates)
	return out
}

func (gm *ReviewGateManager) Summary() string {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	passed := 0
	required := 0
	for _, g := range gm.gates {
		if g.Required {
			required++
		}
		if g.Passed {
			passed++
		}
	}
	return fmt.Sprintf("%d/%d required gates passed (%d total)", passed, required, len(gm.gates))
}

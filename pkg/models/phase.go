package models

import "fmt"

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

type PhaseTransition struct {
	From      Phase  `json:"from"`
	To        Phase  `json:"to"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

type LifecyclePhase int

const (
	LifecycleDemo       LifecyclePhase = 0
	LifecyclePrototype  LifecyclePhase = 1
	LifecycleMVP        LifecyclePhase = 2
	LifecycleProduction LifecyclePhase = 3
)

func (lp LifecyclePhase) String() string {
	switch lp {
	case LifecycleDemo:
		return "Demo"
	case LifecyclePrototype:
		return "Prototype"
	case LifecycleMVP:
		return "MVP"
	case LifecycleProduction:
		return "Production"
	default:
		return fmt.Sprintf("Unknown(%d)", lp)
	}
}

type LifecycleTransition struct {
	From      LifecyclePhase `json:"from"`
	To        LifecyclePhase `json:"to"`
	Reason    string         `json:"reason"`
	Timestamp string         `json:"timestamp"`
}

const (
	DemoDeliverable       string = "Product concept, initial user flow, demo artifacts"
	PrototypeDeliverable  string = "Working prototype, technical architecture, core integrations"
	MVPDeliverable        string = "Authentication, persistence, monitoring, core functionality, test coverage"
	ProductionDeliverable string = "Security reviews, performance testing, observability, deployment readiness"
)

func LifecycleDeliverable(lp LifecyclePhase) string {
	switch lp {
	case LifecycleDemo:
		return DemoDeliverable
	case LifecyclePrototype:
		return PrototypeDeliverable
	case LifecycleMVP:
		return MVPDeliverable
	case LifecycleProduction:
		return ProductionDeliverable
	default:
		return ""
	}
}

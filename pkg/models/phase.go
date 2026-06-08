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

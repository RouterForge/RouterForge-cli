package models

import "time"

type Plan struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	Summary     string        `json:"summary"`
	Teams       []PlanTeam    `json:"teams"`
	Decisions   []PlanDecision `json:"decisions"`
	CreatedAt   string        `json:"created_at"`
}

type PlanTeam struct {
	Domain  string      `json:"domain"`
	Lead    string      `json:"lead"`
	Agents  []PlanAgent `json:"agents"`
}

type PlanAgent struct {
	Role            string   `json:"role"`
	Description     string   `json:"description"`
	Model           string   `json:"model,omitempty"`
	Tools           []string `json:"tools"`
	Tasks           []string `json:"tasks"`
	SuccessCriteria []string `json:"success_criteria"`
}

type PlanDecision struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

func NewPlan(projectID string) *Plan {
	return &Plan{
		ID:        "plan-" + projectID,
		ProjectID: projectID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

type TraceEntry struct {
	Timestamp  string `json:"timestamp"`
	Event      string `json:"event"`
	AgentID    string `json:"agent_id,omitempty"`
	Phase      string `json:"phase,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Status     string `json:"status,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

package models

type AgentType string

const (
	TypeHeadManager AgentType = "head_manager"
	TypeTeamManager AgentType = "team_manager"
	TypeMicroAgent  AgentType = "micro_agent"
)

type AgentStatus string

const (
	StatusCreated   AgentStatus = "created"
	StatusAssigned  AgentStatus = "assigned"
	StatusActive    AgentStatus = "active"
	StatusBlocked   AgentStatus = "blocked"
	StatusCompleted AgentStatus = "completed"
	StatusFailed    AgentStatus = "failed"
)

type Agent struct {
	ID              string      `json:"id"`
	Role            string      `json:"role"`
	Type            AgentType   `json:"type"`
	ParentID        string      `json:"parent_id"`
	Model           string      `json:"model"`
	Status          AgentStatus `json:"status"`
	SystemPrompt    string      `json:"system_prompt"`
	Tasks           []Task      `json:"tasks"`
	SuccessCriteria []string    `json:"success_criteria"`
	Tools           []string    `json:"tools"`
	MemoryScope     []string    `json:"memory_scope"`
	ReportingFormat string      `json:"reporting_format"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

package models

type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
	TaskBlocked    TaskStatus = "blocked"
)

type Task struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	AssignedTo  string     `json:"assigned_to,omitempty"`
	DependsOn   []string   `json:"depends_on,omitempty"`
	Priority    string     `json:"priority"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

type Decision struct {
	ID        string `json:"id"`
	AgentID   string `json:"agent_id"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
	Timestamp string `json:"timestamp"`
}

type AgentMessage struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
	Timestamp string `json:"timestamp"`
}

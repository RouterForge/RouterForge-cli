package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/routerforge/cli/internal/engine"
	"github.com/routerforge/cli/internal/event"
	"github.com/routerforge/cli/internal/memory"
	"github.com/routerforge/cli/pkg/models"
	"github.com/google/uuid"
)

type MicroAgent struct {
	agent    *models.Agent
	context  *Context
	tasks    []*TaskRunner
	statusCh chan AgentEvent
	bus      event.Bus
	mem      memory.Store
}

type AgentEvent struct {
	AgentID string
	Type    string
	Payload string
	Error   error
}

type Context struct {
	Project *models.Project
	Model   string
	Data    map[string]string
	Memory  memory.Store
}

func NewMicroAgent(agent *models.Agent, ctx *Context) *MicroAgent {
	runners := make([]*TaskRunner, len(agent.Tasks))
	mem := ctx.Memory
	for i := range agent.Tasks {
		runners[i] = &TaskRunner{task: &agent.Tasks[i], mem: mem, agentID: agent.ID}
	}
	return &MicroAgent{
		agent:    agent,
		context:  ctx,
		tasks:    runners,
		statusCh: make(chan AgentEvent, 10),
		mem:      mem,
	}
}

func (ma *MicroAgent) ID() string                 { return ma.agent.ID }
func (ma *MicroAgent) Role() string               { return ma.agent.Role }
func (ma *MicroAgent) Status() models.AgentStatus { return ma.agent.Status }
func (ma *MicroAgent) Agent() *models.Agent       { return ma.agent }
func (ma *MicroAgent) Events() <-chan AgentEvent  { return ma.statusCh }
func (ma *MicroAgent) SetBus(b event.Bus) {
	ma.bus = b
	for _, r := range ma.tasks {
		r.bus = b
	}
}

func (ma *MicroAgent) SetMemory(m memory.Store) {
	ma.mem = m
	for _, r := range ma.tasks {
		r.mem = m
	}
}

func (ma *MicroAgent) Execute(ctx context.Context) error {
	ma.agent.Status = models.StatusActive
	ma.agent.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	ma.statusCh <- AgentEvent{AgentID: ma.agent.ID, Type: "started", Payload: fmt.Sprintf("Agent %s started", ma.agent.Role)}
	if ma.bus != nil {
		ma.bus.Publish(event.EvtAgentCreated, event.Event{
			Source:  ma.agent.ID,
			Payload: fmt.Sprintf("micro-agent %s started", ma.agent.Role),
		})
	}

	for _, runner := range ma.tasks {
		select {
		case <-ctx.Done():
			ma.agent.Status = models.StatusFailed
			return ctx.Err()
		default:
		}
		err := runner.Execute(ctx, ma.context, ma.statusCh)
		if err != nil {
			ma.agent.Status = models.StatusFailed
			ma.agent.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			ma.statusCh <- AgentEvent{AgentID: ma.agent.ID, Type: "failed", Error: err}
			return fmt.Errorf("agent %s task %s failed: %w", ma.agent.ID, runner.task.ID, err)
		}
	}

	ma.agent.Status = models.StatusCompleted
	ma.agent.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	ma.statusCh <- AgentEvent{AgentID: ma.agent.ID, Type: "completed", Payload: fmt.Sprintf("Agent %s completed all tasks", ma.agent.Role)}
	return nil
}

func (ma *MicroAgent) Escalate(issue string) error {
	ma.agent.Status = models.StatusBlocked
	ma.agent.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	ma.statusCh <- AgentEvent{
		AgentID: ma.agent.ID,
		Type:    "escalated",
		Payload: fmt.Sprintf("ESCALATION from %s: %s", ma.agent.Role, issue),
	}
	return nil
}

func (ma *MicroAgent) Checkpoint() error {
	if ma.mem == nil {
		return nil
	}
	summary := fmt.Sprintf("Checkpoint: agent=%s status=%s tasks=%d",
		ma.agent.ID, ma.agent.Status, len(ma.agent.Tasks))
	ma.mem.Add(memory.Entry{
		ID:      uuid.New().String(),
		AgentID: ma.agent.ID,
		Type:    memory.TypeTool,
		Content: summary,
	})
	return nil
}

func (ma *MicroAgent) Restore() ([]memory.Entry, error) {
	if ma.mem == nil {
		return nil, nil
	}
	entries, err := ma.mem.Query(ma.agent.ID, memory.TypeDecision, 5)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (ma *MicroAgent) Report() (*AgentReport, error) {
	report := &AgentReport{
		AgentID:   ma.agent.ID,
		Role:      ma.agent.Role,
		Status:    string(ma.agent.Status),
		TaskCount: len(ma.agent.Tasks),
		Model:     ma.agent.Model,
	}
	for _, t := range ma.agent.Tasks {
		report.Tasks = append(report.Tasks, TaskResult{
			ID:     t.ID,
			Status: string(t.Status),
			Result: t.Result,
		})
	}
	return report, nil
}

type AgentReport struct {
	AgentID   string       `json:"agent_id"`
	Role      string       `json:"role"`
	Status    string       `json:"status"`
	TaskCount int          `json:"task_count"`
	Tasks     []TaskResult `json:"tasks"`
	Model     string       `json:"model"`
}

type TaskResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type TaskRunner struct {
	task    *models.Task
	bus     event.Bus
	mem     memory.Store
	agentID string
}

func (tr *TaskRunner) Execute(ctx context.Context, c *Context, statusCh chan<- AgentEvent) error {
	tr.task.Status = models.TaskInProgress
	tr.task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	statusCh <- AgentEvent{AgentID: c.Project.ID, Type: "task_started", Payload: tr.task.Description}
	if tr.bus != nil {
		tr.bus.Publish(event.EvtTaskStarted, event.Event{
			Source:  c.Project.ID,
			Payload: tr.task.Description,
		})
	}

	select {
	case <-ctx.Done():
		tr.task.Status = models.TaskFailed
		return ctx.Err()
	default:
	}

	llm := engine.NewLLMClient(c.Model)

	artifactsDir := filepath.Join(".", ".routerforge", "artifacts")
	os.MkdirAll(artifactsDir, 0755)

	ctxPrompt := tr.task.Description
	if tr.mem != nil {
		comp := memory.NewCompressor(tr.mem)
		past := comp.BuildContext(tr.agentID)
		if past != "" {
			ctxPrompt = past + "\n\nCurrent task: " + tr.task.Description
		}
	}

	result, err := llm.Chat("You are a senior engineer. Generate production-quality code or documentation for the following task. Return ONLY the raw content, no markdown fences, no explanations.", ctxPrompt)
	if err != nil {
		statusCh <- AgentEvent{AgentID: c.Project.ID, Type: "task_failed", Error: err}
		tr.task.Status = models.TaskFailed
		tr.task.Error = err.Error()
		if tr.bus != nil {
			tr.bus.Publish(event.EvtTaskFailed, event.Event{
				Source:  c.Project.ID,
				Payload: fmt.Sprintf("task %s: %s", tr.task.ID, err.Error()),
			})
		}
		return fmt.Errorf("LLM call failed: %w", err)
	}

	safeName := filepath.Base(tr.task.Description)
	ext := ".md"
	if strings.Contains(tr.task.Description, "HTML") || strings.Contains(tr.task.Description, "html") {
		ext = ".html"
	}
	filename := filepath.Join(artifactsDir, safeName+ext)
	if err := os.WriteFile(filename, []byte(result), 0644); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}

	tr.task.Result = fmt.Sprintf("Generated artifact (%d bytes) via %s", len(result), c.Model)

	if tr.mem != nil {
		tr.mem.Add(memory.Entry{
			ID:      uuid.New().String(),
			AgentID: tr.agentID,
			Type:    memory.TypeDecision,
			Content: fmt.Sprintf("Completed task: %s -> %s", tr.task.Description, tr.task.Result),
		})
	}

	tr.task.Status = models.TaskCompleted
	tr.task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	statusCh <- AgentEvent{AgentID: c.Project.ID, Type: "task_completed", Payload: tr.task.Description}
	if tr.bus != nil {
		tr.bus.Publish(event.EvtTaskCompleted, event.Event{
			Source:  c.Project.ID,
			Payload: tr.task.Description,
		})
	}
	return nil
}

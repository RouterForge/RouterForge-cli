package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/event"
	"github.com/routerforge/cli/pkg/models"
)

type TeamManager struct {
	agent       *models.Agent
	context     *Context
	microAgents map[string]*MicroAgent
	bus         event.Bus
}

func (tm *TeamManager) SetBus(b event.Bus) {
	tm.bus = b
	for _, ma := range tm.microAgents {
		ma.SetBus(b)
	}
}

func NewTeamManager(agent *models.Agent, ctx *Context) *TeamManager {
	return &TeamManager{
		agent:       agent,
		context:     ctx,
		microAgents: make(map[string]*MicroAgent),
	}
}

func (tm *TeamManager) ID() string                     { return tm.agent.ID }
func (tm *TeamManager) Role() string                   { return tm.agent.Role }
func (tm *TeamManager) Agents() map[string]*MicroAgent { return tm.microAgents }

func (tm *TeamManager) CreateMicroAgent(role string, tasks []TaskDef, model string) (*MicroAgent, error) {
	agentTasks := make([]models.Task, len(tasks))
	for i, t := range tasks {
		agentTasks[i] = models.Task{
			ID:          fmt.Sprintf("%s-%s-%d", tm.agent.ID, role, i),
			Description: t.Description,
			Status:      models.TaskPending,
			Priority:    t.Priority,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}
	}

	agent := &models.Agent{
		ID:              fmt.Sprintf("%s-%s", tm.agent.ID, role),
		Role:            role,
		Type:            models.TypeMicroAgent,
		ParentID:        tm.agent.ID,
		Model:           model,
		Status:          models.StatusCreated,
		SystemPrompt:    fmt.Sprintf("You are a %s agent. You work under %s. Use model %s. Complete your assigned tasks and report results.", role, tm.agent.Role, model),
		Tasks:           agentTasks,
		SuccessCriteria: []string{fmt.Sprintf("All %s tasks completed", role)},
		Tools:           []string{"read", "write", "search"},
		MemoryScope:     []string{"project", tm.agent.Role},
		ReportingFormat: "json",
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	micro := NewMicroAgent(agent, tm.context)
	tm.microAgents[agent.ID] = micro

	pterm.Info.Printfln("  Created micro-agent: %s (model: %s)", role, model)
	return micro, nil
}

func (tm *TeamManager) AdoptPreBuiltAgent(agent *models.Agent, tasks []TaskDef) *MicroAgent {
	agentTasks := make([]models.Task, len(tasks))
	for i, t := range tasks {
		agentTasks[i] = models.Task{
			ID:          fmt.Sprintf("%s-%s-%d", tm.agent.ID, agent.Role, i),
			Description: t.Description,
			Status:      models.TaskPending,
			Priority:    t.Priority,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}
	}
	agent.ID = fmt.Sprintf("%s-%s", tm.agent.ID, agent.Role)
	agent.ParentID = tm.agent.ID
	agent.Tasks = agentTasks
	agent.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	agent.UpdatedAt = agent.CreatedAt

	micro := NewMicroAgent(agent, tm.context)
	tm.microAgents[agent.ID] = micro
	return micro
}

func (tm *TeamManager) ExecuteAll(ctx context.Context) error {
	pterm.DefaultSection.Printfln("Team: %s executing...", tm.agent.Role)

	for id, agent := range tm.microAgents {
		pterm.Info.Printfln("  Starting agent: %s", id)
		err := agent.Execute(ctx)
		if err != nil {
			pterm.Error.Printfln("  Agent %s failed: %v", id, err)
			continue
		}
		report, _ := agent.Report()
		pterm.Success.Printfln("  Agent %s completed: %d tasks", id, report.TaskCount)
	}

	return nil
}

func (tm *TeamManager) CollectReports() []*AgentReport {
	var reports []*AgentReport
	for _, agent := range tm.microAgents {
		r, err := agent.Report()
		if err == nil {
			reports = append(reports, r)
		}
	}
	return reports
}

type TaskDef struct {
	Description string
	Priority    string
}

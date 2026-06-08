package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/agent"
	"github.com/routerforge/cli/internal/event"
	"github.com/routerforge/cli/internal/memory"
	"github.com/routerforge/cli/pkg/models"
)

type HeadManager struct {
	project      *models.Project
	stateMachine *StateMachine
	teams        map[string]*TeamManager
	decisions    []models.Decision
	messages     []models.AgentMessage
	model        string
	userProxy    agent.UserInputProvider
	bus          event.Bus
	mem          memory.Store
}

func NewHeadManager(model string) *HeadManager {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	return &HeadManager{
		project: &models.Project{
			ID:        id,
			Phase:     models.PhaseIdle,
			Model:     model,
			CreatedAt: now,
			UpdatedAt: now,
		},
		stateMachine: NewStateMachine(),
		teams:        make(map[string]*TeamManager),
		decisions:    []models.Decision{},
		messages:     []models.AgentMessage{},
		model:        model,
		userProxy:    agent.NewTerminalUserProxy(),
		bus:          event.NewBus(),
	}
}

func (hm *HeadManager) SetBus(b event.Bus) {
	hm.bus = b
}

func (hm *HeadManager) Bus() event.Bus {
	return hm.bus
}

func (hm *HeadManager) SetMemory(m memory.Store) {
	hm.mem = m
}

func (hm *HeadManager) AttachConsoleLogger() {
	hm.bus.Subscribe("*", func(evt event.Event) {
		switch evt.Type {
		case event.EvtPhaseChanged:
			p := ""
			if m, ok := evt.Payload.(map[string]string); ok {
				p = m["phase"]
			}
			pterm.Info.Printfln("⏩ Phase: %s", p)
		case event.EvtTaskStarted:
			pterm.Info.Printfln("⏳ Task started: %v", evt.Payload)
		case event.EvtTaskCompleted:
			pterm.Success.Printfln("✅ Task complete: %v", evt.Payload)
		case event.EvtTaskFailed:
			pterm.Error.Printfln("❌ Task failed: %v", evt.Payload)
		case event.EvtAgentCreated:
			pterm.Info.Printfln("🤖 Agent created: %v", evt.Payload)
		case event.EvtEscalation:
			pterm.Warning.Printfln("🚨 Escalation: %v", evt.Payload)
		}
	})
}

func (hm *HeadManager) SetUserProxy(up agent.UserInputProvider) {
	hm.userProxy = up
}

func (hm *HeadManager) Project() *models.Project       { return hm.project }
func (hm *HeadManager) Teams() map[string]*TeamManager { return hm.teams }
func (hm *HeadManager) State() Phase                   { return hm.stateMachine.current }

func (hm *HeadManager) Understand() error {
	if err := hm.stateMachine.Transition(PhaseUnderstand, "Starting understand phase"); err != nil {
		return err
	}
	hm.bus.Publish(event.EvtPhaseChanged, event.Event{
		Source:  "head_manager",
		Payload: map[string]string{"phase": "understand"},
	})

	pterm.DefaultBigText.WithLetters(pterm.NewLettersFromString("RouterForge")).Render()
	pterm.DefaultHeader.WithFullWidth().Println("AI Software Company Operating System")
	pterm.Println()

	questions := []struct {
		prompt string
		field  string
	}{
		{"What is the name of your project?", "Name"},
		{"Describe your project goal — what are you building?", "Goal"},
		{"Who are the target users?", "TargetUsers"},
		{"What tech stack do you prefer? (e.g., Go, React, Python)", "TechStack"},
		{"What are the key features?", "Features"},
		{"Any constraints or special requirements?", "Constraints"},
	}

	answers := make(map[string]string)
	for _, q := range questions {
		answer, _ := hm.userProxy.Ask(q.prompt)
		answers[q.field] = answer
		hm.logDecision("understand", fmt.Sprintf("%s: %s", q.field, answer))
	}

	hm.project.Name = answers["Name"]
	hm.project.Goal = answers["Goal"]
	hm.project.TechStack = answers["TechStack"]
	hm.project.Description = fmt.Sprintf("Target users: %s | Features: %s | Constraints: %s",
		answers["TargetUsers"], answers["Features"], answers["Constraints"])
	hm.project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	model := hm.askModelChoice()
	hm.model = model
	hm.project.Model = model
	hm.logDecision("model_selection", fmt.Sprintf("Selected model: %s", model))

	pterm.Success.Printfln("Project understood: %s", hm.project.Name)
	return nil
}

func (hm *HeadManager) askModelChoice() string {
	pterm.Info.Println("❓ Choose an AI model for your agents:")
	pterm.Println("Available free models:")
	pterm.Println("  1) big-pickle         (default, recommended)")
	pterm.Println("  2) deepseek-v4-flash-free")
	pterm.Println("  3) qwen3.6-plus-free")
	pterm.Println("  4) mimo-v2.5-free")
	pterm.Println("  5) minimax-m3-free")
	pterm.Println("  6) nemotron-3-super-free")
	pterm.Println("  7) nemotron-3-ultra-free")
	modelOpts := []string{
		"big-pickle",
		"deepseek-v4-flash-free",
		"qwen3.6-plus-free",
		"mimo-v2.5-free",
		"minimax-m3-free",
		"nemotron-3-super-free",
		"nemotron-3-ultra-free",
	}
	choice, _ := hm.userProxy.Choose("Choose an AI model for your agents:", modelOpts)
	return choice
}

func (hm *HeadManager) Design() error {
	if err := hm.stateMachine.Transition(PhaseDesign, "Starting design phase"); err != nil {
		return err
	}
	hm.bus.Publish(event.EvtPhaseChanged, event.Event{
		Source:  "head_manager",
		Payload: map[string]string{"phase": "design"},
	})

	pterm.DefaultSection.Printfln("Design Phase — Creating Teams")

	pterm.Info.Println("Which domains does your project need?")
	pterm.Println("  (e.g., frontend, backend, database, security, qa, devops, browser, content)")
	input, _ := hm.userProxy.Ask("Enter comma-separated domains [backend, frontend]:")
	if input == "" {
		input = "backend, frontend"
	}

	domains := strings.Split(input, ",")

	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		hm.CreateTeam(d)
	}

	hm.logDecision("design", fmt.Sprintf("Teams created: %s", strings.Join(domains, ", ")))
	pterm.Success.Printfln("Design complete — %d teams created", len(hm.teams))
	return nil
}

func (hm *HeadManager) CreateTeam(domain string) (*TeamManager, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	agent := &models.Agent{
		ID:              fmt.Sprintf("team-%s", domain),
		Role:            fmt.Sprintf("%s_lead", domain),
		Type:            models.TypeTeamManager,
		ParentID:        hm.project.ID,
		Model:           hm.model,
		Status:          models.StatusCreated,
		SystemPrompt:    fmt.Sprintf("You are the %s team lead. You manage micro-agents for %s domain.", domain, domain),
		Tasks:           []models.Task{},
		SuccessCriteria: []string{fmt.Sprintf("%s domain completed", domain)},
		Tools:           []string{"read", "write", "search", "bash"},
		MemoryScope:     []string{"project", domain},
		ReportingFormat: "json",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	ctx := &Context{
		Project: hm.project,
		Model:   hm.model,
		Data:    make(map[string]string),
		Memory:  hm.mem,
	}

	tm := NewTeamManager(agent, ctx)
	hm.teams[agent.ID] = tm

	hm.bus.Publish(event.EvtAgentCreated, event.Event{
		Source:  "head_manager",
		Payload: fmt.Sprintf("team: %s (lead: %s)", domain, agent.Role),
	})

	pterm.Success.Printfln("  Created team: %s (lead: %s, model: %s)", domain, agent.Role, hm.model)
	return tm, nil
}

func (hm *HeadManager) Execute() error {
	hm.stateMachine.Transition(PhaseExecute, "Starting execute phase")
	hm.bus.Publish(event.EvtPhaseChanged, event.Event{
		Source:  "head_manager",
		Payload: map[string]string{"phase": "execute"},
	})

	pterm.DefaultSection.Printfln("Execute Phase — Running Agents")

	ctx := context.Background()

	for id, tm := range hm.teams {
		tm.SetBus(hm.bus)
		pterm.Info.Printfln("Executing team: %s", id)
		err := tm.ExecuteAll(ctx)
		if err != nil {
			pterm.Error.Printfln("Team %s failed: %v", id, err)
		}
	}

	hm.logDecision("execute", "All teams executed")
	pterm.Success.Println("Execute phase complete")
	return nil
}

func (hm *HeadManager) Review() error {
	hm.stateMachine.Transition(PhaseReview, "Starting review phase")
	hm.bus.Publish(event.EvtPhaseChanged, event.Event{
		Source:  "head_manager",
		Payload: map[string]string{"phase": "review"},
	})

	pterm.DefaultSection.Printfln("Review Phase — Checking Results")

	for id, tm := range hm.teams {
		reports := tm.CollectReports()
		pterm.Info.Printfln("Team %s: %d agents completed", id, len(reports))
		for _, r := range reports {
			pterm.Printfln("  Agent: %s | Status: %s | Tasks: %d", r.Role, r.Status, r.TaskCount)
			for _, t := range r.Tasks {
				status := "✅"
				if t.Status != "completed" {
					status = "❌"
				}
				pterm.Printfln("    %s %s: %s", status, t.ID, t.Status)
			}
		}
	}

	hm.logDecision("review", "Review complete")
	pterm.Success.Println("Review phase complete. All agents accounted for.")
	return nil
}

func (hm *HeadManager) logDecision(action, payload string) {
	hm.decisions = append(hm.decisions, models.Decision{
		ID:        fmt.Sprintf("dec-%d", len(hm.decisions)+1),
		AgentID:   "head_manager",
		Type:      action,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (hm *HeadManager) SendMessage(from, to, msgType, payload string) {
	hm.messages = append(hm.messages, models.AgentMessage{
		From:      from,
		To:        to,
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (hm *HeadManager) RunFullPipeline() error {
	pterm.DefaultSection.Printfln("🚀 RouterForge Pipeline Starting")
	pterm.Println()

	if err := hm.Understand(); err != nil {
		return fmt.Errorf("understand phase failed: %w", err)
	}

	hm.SendMessage("head_manager", "all", "broadcast", fmt.Sprintf("Model selected: %s", hm.model))

	if err := hm.Design(); err != nil {
		return fmt.Errorf("design phase failed: %w", err)
	}

	hm.SendMessage("head_manager", "all_teams", "broadcast", fmt.Sprintf("Teams created. Using model: %s", hm.model))

	if err := hm.Execute(); err != nil {
		return fmt.Errorf("execute phase failed: %w", err)
	}

	if err := hm.Review(); err != nil {
		return fmt.Errorf("review phase failed: %w", err)
	}

	pterm.DefaultSection.Printfln("✅ Pipeline Complete")
	return nil
}

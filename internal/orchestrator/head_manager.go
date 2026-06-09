package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/agent"
	"github.com/routerforge/cli/internal/engine"
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
	plan         *models.Plan
	tracePath    string
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

func (hm *HeadManager) Project() *models.Project            { return hm.project }
func (hm *HeadManager) Teams() map[string]*TeamManager      { return hm.teams }
func (hm *HeadManager) State() Phase                        { return hm.stateMachine.current }
func (hm *HeadManager) Model() string                       { return hm.model }
func (hm *HeadManager) Decisions() []models.Decision        { return hm.decisions }
func (hm *HeadManager) StateHistory() []models.PhaseTransition { return hm.stateMachine.History() }

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

	pterm.DefaultSection.Printfln("Design Phase — Generating Dynamic Team Plan")

	plan, err := hm.GeneratePlan()
	if err != nil {
		pterm.Warning.Printfln("LLM plan generation failed (%v), falling back to interactive mode", err)
		return hm.designInteractive()
	}

	hm.plan = plan
	hm.logDecision("plan", fmt.Sprintf("Generated plan with %d teams via LLM", len(plan.Teams)))

	for _, pt := range plan.Teams {
		tm, err := hm.CreateTeam(pt.Domain)
		if err != nil {
			pterm.Warning.Printfln("Failed to create team %s: %v", pt.Domain, err)
			continue
		}
		for _, pa := range pt.Agents {
			tasks := make([]TaskDef, len(pa.Tasks))
			for i, t := range pa.Tasks {
				prio := "medium"
				if i == 0 {
					prio = "high"
				}
				tasks[i] = TaskDef{Description: t, Priority: prio}
			}
			tm.CreateMicroAgent(pa.Role, tasks, hm.model)
		}
	}

	pterm.Success.Printfln("Design complete — %d teams, %d agents created dynamically",
		len(plan.Teams), countAgents(plan))
	return nil
}

func countAgents(p *models.Plan) int {
	n := 0
	for _, t := range p.Teams {
		n += len(t.Agents)
	}
	return n
}

func (hm *HeadManager) designInteractive() error {
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
	hm.logDecision("design", fmt.Sprintf("Teams created (interactive): %s", strings.Join(domains, ", ")))
	pterm.Success.Printfln("Design complete — %d teams created", len(hm.teams))
	return nil
}

func (hm *HeadManager) GeneratePlan() (*models.Plan, error) {
	prompt := fmt.Sprintf(`You are a software architecture planner. Given the following project, design the optimal team structure, agent roles, and tasks.

Project: %s
Goal: %s
Tech Stack: %s
Description: %s

Return ONLY a JSON object with this structure:
{
  "summary": "brief summary of the plan",
  "teams": [
    {
      "domain": "the domain name (e.g., frontend, backend)",
      "lead": "role name for team lead",
      "agents": [
        {
          "role": "agent_role_name",
          "description": "what this agent does",
          "tools": ["read","write","search","bash"],
          "tasks": ["specific task 1","specific task 2"],
          "success_criteria": ["criterion 1"]
        }
      ]
    }
  ],
  "decisions": [
    {"reason": "why a decision was made", "detail": "explanation"}
  ]
}

Include at least one agent per team. Each agent needs 2-4 concrete tasks.`,
		hm.project.Name, hm.project.Goal, hm.project.TechStack, hm.project.Description)

	llm := engine.NewLLMClient(hm.model)
	result, err := llm.Chat("You are a senior software architect. Generate structured JSON plans only.", prompt)
	if err != nil {
		return nil, err
	}

	result = stripMarkdown(result)
	plan := models.NewPlan(hm.project.ID)
	if err := json.Unmarshal([]byte(result), plan); err != nil {
		return nil, fmt.Errorf("parse plan: %w\nraw: %s", err, result[:min(len(result), 200)])
	}

	hm.savePlanArtifact(plan)
	return plan, nil
}

func (hm *HeadManager) savePlanArtifact(p *models.Plan) {
	artifactsDir := filepath.Join(".", ".routerforge", "artifacts")
	os.MkdirAll(artifactsDir, 0755)
	data, _ := json.MarshalIndent(p, "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, "plan.json"), data, 0644)
}

func (hm *HeadManager) Plan() *models.Plan { return hm.plan }

func (hm *HeadManager) SetTracePath(path string) { hm.tracePath = path }

func (hm *HeadManager) WriteTrace(eventType, agentID, phase, taskID, status, detail string) {
	if hm.tracePath == "" {
		hm.tracePath = filepath.Join(".", ".routerforge", "artifacts", "trace.jsonl")
	}
	os.MkdirAll(filepath.Dir(hm.tracePath), 0755)
	entry := models.TraceEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Event:     eventType,
		AgentID:   agentID,
		Phase:     phase,
		TaskID:    taskID,
		Status:    status,
		Detail:    detail,
	}
	data, _ := json.Marshal(entry)
	f, _ := os.OpenFile(hm.tracePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.Write(data)
		f.Write([]byte("\n"))
		f.Close()
	}
}

func stripMarkdown(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		idx := strings.Index(s[3:], "\n")
		if idx >= 0 {
			s = s[3+idx+1:]
		}
	}
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	return s
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
	hm.WriteTrace("phase_start", "head_manager", "execute", "", "", "executing teams")

	pterm.DefaultSection.Printfln("Execute Phase — Running Agents")

	ctx := context.Background()

	for id, tm := range hm.teams {
		tm.SetBus(hm.bus)
		pterm.Info.Printfln("Executing team: %s", id)
		start := time.Now()
		err := tm.ExecuteAll(ctx)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			pterm.Error.Printfln("Team %s failed: %v", id, err)
			hm.WriteTrace("team_failed", id, "execute", "", "failed", err.Error())
		} else {
			hm.WriteTrace("team_completed", id, "execute", "", "completed", fmt.Sprintf("took %dms", elapsed))
		}
	}

	hm.logDecision("execute", "All teams executed")
	hm.WriteTrace("phase_end", "head_manager", "execute", "", "completed", "all teams executed")
	pterm.Success.Println("Execute phase complete")
	return nil
}

func (hm *HeadManager) Review() error {
	hm.stateMachine.Transition(PhaseReview, "Starting review phase")
	hm.bus.Publish(event.EvtPhaseChanged, event.Event{
		Source:  "head_manager",
		Payload: map[string]string{"phase": "review"},
	})
	hm.WriteTrace("phase_start", "head_manager", "review", "", "", "reviewing results")

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
	hm.WriteTrace("phase_end", "head_manager", "review", "", "completed", "review complete")
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

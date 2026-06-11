package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	project          *models.Project
	projectDir       string
	stateMachine     *StateMachine
	lifecycleMachine *LifecycleStateMachine
	teams            map[string]*TeamManager
	decisions        []models.Decision
	messages         []models.AgentMessage
	model            string
	userProxy        agent.UserInputProvider
	bus              event.Bus
	mem              memory.Store
	plan             *models.Plan
	tracePath        string
	conversationsDir string
	tokenBudget      *TokenBudget
	tokenTracker     *TokenTracker
	costTracker      *CostTracker
	reviewGates      *ReviewGateManager
	memPolicy        *MemoryPolicy
	memEnforcer      *MemoryPolicyEnforcer
	sandbox          *ToolSandbox
	spawner          *engine.AgentSpawner
	scheduler        *Scheduler
	runtime          *Runtime
}

func NewHeadManager(model string) *HeadManager {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	mp := NewMemoryPolicy()
	sp := NewSandboxPolicy()
	hm := &HeadManager{
		project: &models.Project{
			ID:             id,
			Phase:          models.PhaseIdle,
			LifecyclePhase: models.LifecycleDemo,
			Model:          model,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		stateMachine:     NewStateMachine(),
		lifecycleMachine: NewLifecycleStateMachine(),
		teams:            make(map[string]*TeamManager),
		decisions:        []models.Decision{},
		messages:         []models.AgentMessage{},
		model:            model,
		userProxy:        agent.NewTerminalUserProxy(),
		bus:              event.NewBus(),
		tokenBudget:      NewTokenBudget(100000),
		tokenTracker:     NewTokenTracker(),
		costTracker:      NewCostTracker(),
		reviewGates:      NewReviewGateManager(),
		memPolicy:        mp,
		memEnforcer:      NewMemoryPolicyEnforcer(mp),
		sandbox:          NewToolSandbox(sp),
		spawner:          engine.NewAgentSpawner(model),
		scheduler:        NewScheduler(3),
	}
	hm.scheduler.Start()
	hm.runtime = NewRuntime(hm.sandbox, hm.scheduler)
	return hm
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

func (hm *HeadManager) Project() *models.Project               { return hm.project }
func (hm *HeadManager) Teams() map[string]*TeamManager         { return hm.teams }
func (hm *HeadManager) State() Phase                           { return hm.stateMachine.current }
func (hm *HeadManager) Model() string                          { return hm.model }
func (hm *HeadManager) Decisions() []models.Decision           { return hm.decisions }
func (hm *HeadManager) Messages() []models.AgentMessage        { return hm.messages }
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

type modelInfo struct {
	Name        string
	Description string
}

var (
	muModels      sync.RWMutex
	cachedModels  []modelInfo
	modelsFetched time.Time
	modelsTTL     = 5 * time.Minute
)

var modelDescriptions = map[string]string{
	"big-pickle":             "default, balanced",
	"deepseek-v4-flash-free": "fast, good for simple/repetitive tasks",
	"mimo-v2.5-free":         "code generation strength",
	"minimax-m3-free":        "general purpose free model",
	"nemotron-3-super-free":  "larger context, reasoning",
	"nemotron-3-ultra-free":  "highest quality, complex reasoning",
	"north-mini-code-free":   "code generation, lightweight",
	"qwen3.6-plus-free":      "general purpose free model",
}

func describeModel(name string) string {
	if d, ok := modelDescriptions[name]; ok {
		return d
	}
	if strings.HasSuffix(name, "ultra-free") {
		return "highest quality, complex reasoning"
	}
	if strings.HasSuffix(name, "super-free") {
		return "larger context, reasoning"
	}
	if strings.HasSuffix(name, "flash-free") {
		return "fast, good for simple/repetitive tasks"
	}
	if strings.HasSuffix(name, "mini-code-free") {
		return "code generation, lightweight"
	}
	if strings.HasSuffix(name, "code-free") {
		return "code generation"
	}
	if strings.HasSuffix(name, "free") {
		return "general purpose free model"
	}
	return ""
}

func fetchAvailableModels() []modelInfo {
	muModels.RLock()
	if time.Since(modelsFetched) < modelsTTL && len(cachedModels) > 0 {
		defer muModels.RUnlock()
		return cachedModels
	}
	muModels.RUnlock()

	muModels.Lock()
	defer muModels.Unlock()

	if time.Since(modelsFetched) < modelsTTL && len(cachedModels) > 0 {
		return cachedModels
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://opencode.ai/zen/v1/models")
	if err != nil {
		pterm.Warning.Printfln("Failed to fetch model list: %v, using defaults", err)
		return defaultModels()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		pterm.Warning.Printfln("Failed to parse model list: %v, using defaults", err)
		return defaultModels()
	}

	var models []modelInfo
	for _, m := range result.Data {
		// Only include free models + big-pickle
		if m.ID == "big-pickle" || strings.HasSuffix(m.ID, "-free") {
			models = append(models, modelInfo{
				Name:        m.ID,
				Description: describeModel(m.ID),
			})
		}
	}
	if len(models) == 0 {
		models = defaultModels()
	}
	cachedModels = models
	modelsFetched = time.Now()
	return models
}

func defaultModels() []modelInfo {
	return []modelInfo{
		{Name: "big-pickle", Description: "default, balanced"},
		{Name: "deepseek-v4-flash-free", Description: "fast, good for simple/repetitive tasks"},
	}
}

func (hm *HeadManager) askModelChoice() string {
	models := fetchAvailableModels()
	pterm.Info.Println("❓ Choose an AI model for your agents:")
	pterm.Println("Available free models (fetched from API):")
	modelOpts := make([]string, len(models))
	for i, m := range models {
		desc := ""
		if m.Description != "" {
			desc = " (" + m.Description + ")"
		}
		if i == 0 {
			desc += " (default, recommended)"
		}
		pterm.Printf("  %d) %s%s\n", i+1, m.Name, desc)
		modelOpts[i] = m.Name
	}
	choice, _ := hm.userProxy.Choose("Choose an AI model for your agents:", modelOpts)
	return choice
}

func modelPromptBlock() string {
	models := fetchAvailableModels()
	var b strings.Builder
	b.WriteString("Available free models (pick the best fit per agent):\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("- %s (%s)\n", m.Name, m.Description))
	}
	b.WriteString("\nFor each agent you can suggest a model via \"model\" field. Leave empty to use the default.\n")
	return b.String()
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
		pterm.Warning.Printfln("LLM plan generation failed (%v), synthesizing from requirements...", err)
		synthErr := hm.designSynthesized()
		if synthErr == nil {
			return nil
		}
		pterm.Warning.Printfln("Synthesis failed (%v), falling back to interactive mode", synthErr)
		return hm.designInteractive()
	}

	hm.plan = plan
	hm.logDecision("plan", fmt.Sprintf("Generated plan with %d teams via LLM", len(plan.Teams)))
	hm.applyPlan(plan)

	pterm.Success.Printfln("Design complete — %d teams, %d agents created dynamically",
		len(plan.Teams), countAgents(plan))
	return nil
}

func (hm *HeadManager) applyPlan(plan *models.Plan) {
	for _, pt := range plan.Teams {
		tm, err := hm.CreateTeam(pt.Domain, pt.Model)
		if err != nil {
			pterm.Warning.Printfln("Failed to create team %s: %v", pt.Domain, err)
			continue
		}
		for _, pa := range pt.Agents {
			agentModel := pa.Model
			if agentModel == "" {
				agentModel = hm.model
			}
			agent := BuildAgentFromPlan(hm.project.Goal, agentModel, pa, tm.agent.ID)
			agent.SystemPrompt = GenerateSystemPromptFromLLM(pa.Role, pa.Description, hm.project.Goal, pa.Tools, pa.Tasks, agentModel)
			tasks := make([]TaskDef, len(agent.Tasks))
			for i, t := range agent.Tasks {
				prio := "medium"
				if i == 0 {
					prio = "high"
				}
				tasks[i] = TaskDef{Description: t.Description, Priority: prio}
			}
			tm.AdoptPreBuiltAgent(agent, tasks)
			hm.memPolicy.Grant(agent.ID, pa.Role, AccessWrite, fmt.Sprintf("Agent %s scope", pa.Role))
			hm.sandbox.RegisterAgent(agent.ID)
		}
	}
}

func (hm *HeadManager) RestorePlan(p *models.Plan) {
	hm.plan = p
	hm.logDecision("plan", fmt.Sprintf("Restored plan with %d teams from saved artifact", len(p.Teams)))
	hm.applyPlan(p)
	pterm.Success.Printfln("Plan restored — %d teams, %d agents ready",
		len(p.Teams), countAgents(p))
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
	pterm.Println("  Suggested from your requirements: frontend, backend")
	input, _ := hm.userProxy.Ask("Enter comma-separated domains [auto-synthesize from requirements]:")
	if input == "" {
		return hm.designSynthesized()
	}
	domains := strings.Split(input, ",")
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		hm.CreateTeam(d, hm.model)
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

%s
Return ONLY a JSON object with this structure:
{
  "summary": "brief summary of the plan",
  "teams": [
    {
      "domain": "the domain name (e.g., frontend, backend)",
      "lead": "role name for team lead",
      "model": "suggested model for the team lead or empty string",
      "agents": [
        {
          "role": "agent_role_name",
          "description": "what this agent does",
          "model": "suggested model or empty string",
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
		hm.project.Name, hm.project.Goal, hm.project.TechStack, hm.project.Description, modelPromptBlock())

	llm := engine.NewLLMClient(hm.model)
	llm.ConversationsDir = hm.conversationsDir
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

func (hm *HeadManager) Plan() *models.Plan                { return hm.plan }
func (hm *HeadManager) TokenBudget() *TokenBudget         { return hm.tokenBudget }
func (hm *HeadManager) TokenTracker() *TokenTracker       { return hm.tokenTracker }
func (hm *HeadManager) MemoryPolicy() *MemoryPolicy       { return hm.memPolicy }
func (hm *HeadManager) Sandbox() *ToolSandbox             { return hm.sandbox }
func (hm *HeadManager) Spawner() *engine.AgentSpawner     { return hm.spawner }
func (hm *HeadManager) Scheduler() *Scheduler             { return hm.scheduler }
func (hm *HeadManager) Runtime() *Runtime                 { return hm.runtime }
func (hm *HeadManager) ResourceManager() *ResourceManager { return hm.runtime.ResourceMgr }
func (hm *HeadManager) MemoryPool() *MemoryPool           { return hm.runtime.MemoryPool }

func (hm *HeadManager) SetTracePath(path string) { hm.tracePath = path }
func (hm *HeadManager) SetProjectDir(dir string) { hm.projectDir = dir }
func (hm *HeadManager) ProjectDir() string       { return hm.projectDir }
func (hm *HeadManager) SetConversationsDir(dir string) { hm.conversationsDir = dir }

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
func (hm *HeadManager) CreateTeam(domain, model string) (*TeamManager, error) {
	if model == "" {
		model = hm.model
	}
	now := time.Now().UTC().Format(time.RFC3339)
	agent := &models.Agent{
		ID:              fmt.Sprintf("team-%s", domain),
		Role:            fmt.Sprintf("%s_lead", domain),
		Type:            models.TypeTeamManager,
		ParentID:        hm.project.ID,
		Model:           model,
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
		Project:          hm.project,
		ProjectDir:       hm.projectDir,
		Model:            model,
		Data:             make(map[string]string),
		Memory:           hm.mem,
		ConversationsDir: hm.conversationsDir,
		CostHandler: func(model, agentID, phase string, usage engine.Usage) {
			hm.costTracker.Track(model, agentID, phase, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.Cost)
		},
	}

	tm := NewTeamManager(agent, ctx)
	hm.teams[agent.ID] = tm

	hm.bus.Publish(event.EvtAgentCreated, event.Event{
		Source:  "head_manager",
		Payload: fmt.Sprintf("team: %s (lead: %s)", domain, agent.Role),
	})

	pterm.Success.Printfln("  Created team: %s (lead: %s, model: %s)", domain, agent.Role, model)

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

	totalTasks := 0
	failedTasks := 0

	for id, tm := range hm.teams {
		reports := tm.CollectReports()
		pterm.Info.Printfln("Team %s: %d agents completed", id, len(reports))
		for _, r := range reports {
			pterm.Printfln("  Agent: %s | Status: %s | Tasks: %d", r.Role, r.Status, r.TaskCount)
			for _, t := range r.Tasks {
				totalTasks++
				status := "✅"
				if t.Status != "completed" {
					status = "❌"
					failedTasks++
				}
				pterm.Printfln("    %s %s: %s", status, t.ID, t.Status)
			}
		}
	}

	// Task completion is reported for observability only. Software validation and
	// repair now define build success.
	if totalTasks == 0 {
		pterm.Warning.Println("No agent tasks were executed")
		hm.logDecision("review", "Review complete: no agent tasks were executed")
		hm.WriteTrace("phase_end", "head_manager", "review", "", "completed", "review complete with no tasks")
		return nil
	}
	failRate := float64(failedTasks) / float64(totalTasks)
	if failRate > 0.5 {
		pterm.Warning.Printfln("Task reliability warning: %.0f%% of tasks failed (%d/%d)", failRate*100, failedTasks, totalTasks)
	}

	hm.logDecision("review", fmt.Sprintf("Review complete: %d/%d tasks passed", totalTasks-failedTasks, totalTasks))
	hm.WriteTrace("phase_end", "head_manager", "review", "", "completed", "review complete")
	pterm.Success.Printfln("Review phase complete. %d/%d tasks passed.", totalTasks-failedTasks, totalTasks)
	return nil
}

func (hm *HeadManager) LifecyclePhase() LifecyclePhase  { return hm.lifecycleMachine.Current() }
func (hm *HeadManager) LifecycleStr() string            { return hm.lifecycleMachine.CurrentStr() }
func (hm *HeadManager) CostTracker() *CostTracker       { return hm.costTracker }
func (hm *HeadManager) ReviewGates() *ReviewGateManager { return hm.reviewGates }
func (hm *HeadManager) CanAdvanceLifecycle() bool       { return hm.reviewGates.AllRequiredPassed() }

func (hm *HeadManager) AdvanceLifecycle() error {
	if !hm.CanAdvanceLifecycle() {
		failed := hm.reviewGates.GetFailedRequired()
		return fmt.Errorf("cannot advance lifecycle: %d required gates not passed: %v", len(failed), failed)
	}
	next := hm.lifecycleMachine.Current() + 1
	if next > LifecycleProduction {
		return fmt.Errorf("already at final lifecycle phase")
	}
	approvals := []string{"head_manager"}
	err := hm.lifecycleMachine.Transition(next, "Advancing lifecycle phase", approvals)
	if err != nil {
		return err
	}
	hm.project.LifecyclePhase = next.ToModel()
	hm.project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	hm.WriteTrace("lifecycle_advance", "head_manager", hm.lifecycleMachine.CurrentStr(), "", "completed",
		fmt.Sprintf("Advanced to %s phase. Deliverable: %s", hm.lifecycleMachine.CurrentStr(), models.LifecycleDeliverable(next.ToModel())))
	hm.bus.Publish(event.EvtPhaseChanged, event.Event{
		Source:  "head_manager",
		Payload: map[string]string{"phase": hm.lifecycleMachine.CurrentStr(), "type": "lifecycle"},
	})
	return nil
}

func (hm *HeadManager) ApproveGate(gateType GateType, approvedBy, notes string) {
	hm.reviewGates.SetGatePassed(gateType, approvedBy, notes)
	hm.WriteTrace("gate_approved", approvedBy, hm.lifecycleMachine.CurrentStr(), "", "approved",
		fmt.Sprintf("Gate %s approved by %s", gateType, approvedBy))
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
		hm.logDecision("execute", fmt.Sprintf("Execute phase had errors: %v", err))
	}

	if err := hm.RepairUntilValid(2); err != nil {
		return fmt.Errorf("repair loop failed: %w", err)
	}

	if err := hm.Review(); err != nil {
		hm.logDecision("review", fmt.Sprintf("Review: %v", err))
	}

	pterm.DefaultSection.Printfln("✅ Pipeline Complete")
	return nil
}
